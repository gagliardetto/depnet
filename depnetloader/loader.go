package depnetloader

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gagliardetto/request"
	. "github.com/gagliardetto/utilz"
	"github.com/michenriksen/aquatone/agents"
	"go.uber.org/ratelimit"
)

var (
	// countCleaner is used to remove anything that is NOT a number.
	countCleaner = regexp.MustCompile("[^0-9]+")

	apiRateLimiter = ratelimit.New(1, ratelimit.WithoutSlack)
)

var (
	defaultMaxIdleConnsPerHost = 50
	defaultTimeout             = 5 * time.Minute
	defaultKeepAlive           = 180 * time.Second
)

var httpClient = NewHTTP()

func NewHTTPTransport() *http.Transport {
	return &http.Transport{
		IdleConnTimeout:     defaultTimeout,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		Proxy:               http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultKeepAlive,
		}).Dial,
		//TLSClientConfig: &tls.Config{
		//	InsecureSkipVerify: conf.InsecureSkipVerify,
		//},
	}
}

// NewHTTP returns a new Client from the provided config.
// Client is safe for concurrent use by multiple goroutines.
func NewHTTP() *http.Client {
	tr := NewHTTPTransport()

	return &http.Client{
		Timeout:   defaultTimeout,
		Transport: tr,
	}
}

type Loader struct {
	repoName string
	owner    string

	dependentType string
	subPackage    string
}

func NewLoader(repoRaw string) *Loader {
	if repoRaw == "" {
		panic("repo not set")
	}

	repoRaw = strings.TrimSpace(repoRaw)
	repoRaw = strings.TrimPrefix(repoRaw, "https://github.com")
	repoRaw = strings.TrimPrefix(repoRaw, "http://github.com")
	repoRaw = strings.TrimPrefix(repoRaw, "/")
	repoRaw = strings.TrimSuffix(repoRaw, "/")

	owner, repoName, err := SplitOwnerRepo(repoRaw)
	if err != nil {
		panic(err)
	}

	return &Loader{
		owner:    owner,
		repoName: repoName,
	}
}

func SplitOwnerRepo(raw string) (string, string, error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("raw not valid: %q", raw)
	}

	owner := parts[0]
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return "", "", fmt.Errorf("owner not valid: %q", owner)
	}

	repo := parts[1]
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", "", fmt.Errorf("repo not valid: %q", repo)
	}
	return owner, repo, nil
}

const (
	TYPE_PACKAGE    = "PACKAGE"
	TYPE_REPOSITORY = "REPOSITORY"
)

func (ldr *Loader) Type(typ string) *Loader {
	if !IsAnyOf(typ, TYPE_PACKAGE, TYPE_REPOSITORY) {
		panic(Sf(
			"Type not valid: %q; must be %q or %q",
			typ,
			TYPE_PACKAGE,
			TYPE_REPOSITORY,
		))
	}
	ldr.dependentType = typ
	return ldr
}

func (ldr *Loader) SubPackage(pkg string) *Loader {
	pkg = strings.TrimSpace(pkg)

	ldr.subPackage = pkg
	return ldr
}

func (ldr *Loader) DoWithCallback(callback func(dep string) bool) error {
	if callback == nil {
		return errors.New("callback is nil")
	}
	if err := ldr.validateBasic(); err != nil {
		return err
	}

	vals := url.Values{}
	{
		vals.Set("dependent_type", ldr.dependentType)
	}

	dst := Sf(
		"https://github.com/%s/%s/network/dependents?%s",
		ldr.owner,
		ldr.repoName,
		vals.Encode(),
	)
	fmt.Fprintf(os.Stderr, "Loading: first page of dependents...\n")
	doc, err := loadPage(dst)
	if err != nil {
		return err
	}

	// Check if the right subpackage is selected:
	{
		subs := extractSubPackages(doc)
		if ldr.subPackage != "" {
			is := subs.IsSelected(ldr.subPackage)
			if !is {
				sub := subs.ByName(ldr.subPackage)
				if sub == nil {
					return fmt.Errorf("subpackage %q not found", ldr.subPackage)
				}

				{
					// Reload doc with the right subpackage:
					doc, err = loadPage("https://github.com" + sub.URL)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	started := time.Now()
	defer func() {
		fmt.Fprintf(os.Stderr, "Done in %s\n", time.Since(started))
	}()

	// Process the first page:
	for _, val := range extractDependents(doc) {
		doContinue := callback(val)
		if !doContinue {
			return nil
		}
	}

	pageNum := 1
	nextPage := extractNextPage(doc)
	for {
		pageNum++
		if nextPage == "" {
			fmt.Fprintf(os.Stderr, "Loading: no more pages of dependents...\n")
			return nil
		}
		fmt.Fprintf(os.Stderr, "Loading: next page (%v, %s) of dependents...", pageNum, nextPage)
		doc, err := loadPage(nextPage)
		if err != nil {
			return err
		}
		dependents := extractDependents(doc)
		fmt.Fprintf(os.Stderr, Lime(" got %d dependents")+"\n", len(dependents))
		for _, val := range dependents {
			doContinue := callback(val)
			if !doContinue {
				return nil
			}
		}
		nextPage = extractNextPage(doc)
	}
}

func (ldr *Loader) validateBasic() error {
	if ldr.owner == "" {
		return errors.New("owner not set")
	}
	if ldr.repoName == "" {
		return errors.New("repoName not set")
	}
	if ldr.dependentType == "" {
		return errors.New("dependentType not set")
	}
	return nil
}

// Info provides information about the dependency network of
// a package.
type Info struct {
	Dependents *DependentsInfo `json:"dependents"`
	// TODO: add Dependencies??
}

type DependentsInfo struct {
	SubPackages SubPackageSlice `json:"subpackages"`
	Counts      *Counts         `json:"counts"`
}

type Counts struct {
	Repositories int `json:"repositories"`
	Packages     int `json:"packages"`
}

func (ldr *Loader) GetInfo() (*Info, error) {
	if err := ldr.validateBasic(); err != nil {
		return nil, err
	}

	vals := url.Values{}
	{
		vals.Set("dependent_type", ldr.dependentType)
	}

	dst := Sf(
		"https://github.com/%s/%s/network/dependents?%s",
		ldr.owner,
		ldr.repoName,
		vals.Encode(),
	)
	doc, err := loadPage(dst)
	if err != nil {
		return nil, err
	}

	info := new(Info)
	info.Dependents = new(DependentsInfo)

	// TODO: validate what has been extracted???
	info.Dependents.SubPackages = extractSubPackages(doc)

	repoCount, packageCount := extractCounts(doc)
	info.Dependents.Counts = &Counts{
		Repositories: repoCount,
		Packages:     packageCount,
	}

	return info, nil
}

func newRequest() *request.Request {
	apiRateLimiter.Take()

	req := request.NewRequest(httpClient)
	req.Headers = map[string]string{
		//"accept":                    "*/*",
		"authority":                 "github.com",
		"cache-control":             "max-age=0",
		"sec-ch-ua":                 `"Chromium";v="88", "Google Chrome";v="88", ";Not A Brand";v="99"`,
		"sec-ch-ua-mobile":          "?0",
		"dnt":                       "1",
		"upgrade-insecure-requests": "1",
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"sec-fetch-site":            "none",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-user":            "?1",
		"sec-fetch-dest":            "document",
		"accept-language":           "en-US,en;q=0.9",
		"sec-gpc":                   "1",
		"user-agent":                agents.RandomUserAgent(),
		"referer":                   "https://github.com",
		"accept-encoding":           "gzip",
	}

	return req
}

func loadPage(url string) (*goquery.Document, error) {
	req := newRequest()

	var resp *request.Response
	errs := RetryExponentialBackoff(7, time.Second, func() (err error) {
		resp, err = req.Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			// TODO: catch rate limit error, and wait
			return fmt.Errorf(
				"status code is: %v (%s)",
				resp.StatusCode,
				resp.Status,
			)
		}
		// nil on 200 and 404
		return nil
	})
	if len(errs) > 0 {
		return nil, errors.New(FormatErrorArray("", errs))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, formatRawResponseBodyError(resp)
	}

	reader, closer, err := resp.DecompressedReaderFromPool()
	if err != nil {
		return nil, fmt.Errorf("error while getting Reader: %s", err)
	}
	defer closer()
	// Load the HTML document
	return goquery.NewDocumentFromReader(reader)
}

func formatRawResponseBodyError(resp *request.Response) error {
	// Get the body as text:
	body, err := resp.Text()
	if err != nil {
		return fmt.Errorf("error while resp.Text: %w", err)
	}
	return fmt.Errorf(
		"Status code: %v\nHeader:\n%s\nBody:\n\n %s",
		resp.StatusCode,
		Sq(resp.Header),
		body,
	)
}

func extractDependents(doc *goquery.Document) []string {
	var dependants []string

	// Find the review items
	doc.Find("[data-repository-hovercards-enabled]").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title

		repository := s.ChildrenFiltered("[data-hovercard-type='repository']")
		repositoryHref, repositoryHrefOk := repository.Attr("href")

		if repositoryHrefOk {
			trimmed := strings.TrimPrefix(repositoryHref, `/`)
			dependants = append(dependants, trimmed)
		}
		// NOTE: only dependents that have a repository are extracted.
		// Others are ignored.
	})
	return dependants
}

func extractNextPage(doc *goquery.Document) string {
	pagination := doc.Find(`[data-test-selector="pagination"]`).ChildrenFiltered("a")
	last := pagination.Last()
	next, ok := last.Attr("href")
	if ok && !strings.Contains(last.Text(), "Previous") {
		return next
	}
	return ""
}

type SubPackage struct {
	Name     string `json:"name"`
	URL      string `json:"-"`
	Selected bool   `json:"-"`
}

type SubPackageSlice []*SubPackage

func (sl SubPackageSlice) IsSelected(name string) bool {
	for _, item := range sl {
		if item.Name == name {
			return item.Selected
		}
	}
	return false
}

func (sl SubPackageSlice) ByName(name string) *SubPackage {
	for _, item := range sl {
		if item.Name == name {
			return item
		}
	}
	return nil
}

func extractSubPackages(doc *goquery.Document) SubPackageSlice {
	res := make([]*SubPackage, 0)
	packages := doc.Find(`div.select-menu-list`).ChildrenFiltered("a.select-menu-item")

	packages.Each(func(i int, pkg *goquery.Selection) {
		// For each item found, get the band and title

		pkgURL, ok := pkg.Attr("href")
		if !ok {
			return
		}
		nameNode := pkg.ChildrenFiltered("span.select-menu-item-text")
		nameText := strings.TrimSpace(nameNode.Text())

		sub := &SubPackage{
			Name: nameText,
			URL:  pkgURL,
		}

		isSelectedText, _ := pkg.Attr("aria-checked")
		if isSelectedText == "true" {
			sub.Selected = true
		}

		res = append(res, sub)
	})
	return res
}

// extractCounts extracts the counts for repository and package dependents (or maybe also dependencies?)
func extractCounts(doc *goquery.Document) (repoCount int, packageCount int) {
	counts := doc.Find(`div.table-list-header-toggle`).ChildrenFiltered("a")

	counts.Each(func(i int, count *goquery.Selection) {
		// For each item found, get the band and title

		nameText := strings.TrimSpace(count.Text())

		processedString := countCleaner.ReplaceAllString(nameText, "")
		if strings.Contains(nameText, "Repositor") {
			parsed, err := Atoi(processedString)
			if err != nil {
				panic(err)
			}
			repoCount = parsed
		}
		if strings.Contains(nameText, "Package") {
			parsed, err := Atoi(processedString)
			if err != nil {
				panic(err)
			}
			packageCount = parsed
		}
	})

	return
}

func ExtractDependentsFromReader(reader io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	return extractDependents(doc), nil
}
