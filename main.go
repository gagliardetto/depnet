package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	. "github.com/gagliardetto/utilz"
)

func main() {

	if false { // Repos:
		if true {
			// Request the HTML page.
			res, err := os.Open(os.ExpandEnv("$GOPATH/src/github.com/gagliardetto/depnet/test-data/REPOSITORY.html"))
			if err != nil {
				log.Fatal(err)
			}
			deps := getDependantRepos(res)
			for _, v := range deps {
				Ln(v)
			}
		} else {
			// Request the HTML page.
			res, err := http.Get("https://github.com/eslint/eslint/network/dependents?dependent_type=REPOSITORY")
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			}
			deps := getDependantRepos(res.Body)
			for _, v := range deps {
				Ln(v)
			}
		}
	}
	// Packages:

	if true {
		if true {
			// Request the HTML page.
			res, err := os.Open(os.ExpandEnv("$GOPATH/src/github.com/gagliardetto/depnet/test-data/PACKAGE.html"))
			if err != nil {
				log.Fatal(err)
			}
			deps := getDependantPackages(res)
			for _, v := range deps {
				Ln(v)
			}
		} else {
			// Request the HTML page.
			res, err := http.Get("https://github.com/eslint/eslint/network/dependents?dependent_type=PACKAGE")
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			}
			deps := getDependantPackages(res.Body)
			for _, v := range deps {
				Ln(v)
			}
		}
	}
}

func getDependantRepos(reader io.Reader) []string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	var rawDependants []string

	// Find the review items
	doc.Find("[data-repository-hovercards-enabled]").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title

		repository := s.ChildrenFiltered("[data-hovercard-type='repository']")
		repositoryHref, repositoryHrefOk := repository.Attr("href")

		if repositoryHrefOk {
			trimmed := strings.TrimPrefix(repositoryHref, `/`)
			rawDependants = append(rawDependants, trimmed)
		}
	})

	next, ok := doc.Find(`[data-test-selector="pagination"]`).ChildrenFiltered("a").Attr("href")
	if ok {
		Ln("Next:", next)
	}

	return rawDependants
}
func getDependantPackages(reader io.Reader) []string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	var rawDependants []string

	// Find the review items
	doc.Find("[data-repository-hovercards-enabled]").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title

		repository := s.ChildrenFiltered("[data-hovercard-type='repository']")
		repositoryHref, repositoryHrefOk := repository.Attr("href")

		if repositoryHrefOk {
			trimmed := strings.TrimPrefix(repositoryHref, `/`)
			rawDependants = append(rawDependants, trimmed)
		}
	})

	next, ok := doc.Find(`[data-test-selector="pagination"]`).ChildrenFiltered("a").Attr("href")
	if ok {
		Ln("Next:", next)
	}

	return rawDependants
}
