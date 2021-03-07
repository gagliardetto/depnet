package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/depnet/depnetloader"
	. "github.com/gagliardetto/utilz"
)

// TODO:
// - Input repo path
// - Validate repo path
// - Load first page for REPOSITORY (if not specified).
// - Ask REPOSITORY or PACKAGE depdendents ? (show counts for both) (ask if not configured in a flag).
// - If there are more than one package specified, ask which to choose (if flag is not set on a precise choice).
// - Iterate and add to array.

func main() {

	if true { // Repos:
		if true {
			// Request the HTML page.
			res, err := os.Open(os.ExpandEnv("$GOPATH/src/github.com/gagliardetto/depnet/test-data/REPOSITORY.html"))
			if err != nil {
				Fataln(err)
			}
			deps, err := depnetloader.ExtractDependentsFromReader(res)
			if err != nil {
				panic(err)
			}
			for _, v := range deps {
				Ln(v)
			}
		} else {
			// Request the HTML page.
			res, err := http.Get("https://github.com/eslint/eslint/network/dependents?dependent_type=REPOSITORY")
			if err != nil {
				Fataln(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			}
			deps, err := depnetloader.ExtractDependentsFromReader(res.Body)
			if err != nil {
				panic(err)
			}
			for _, v := range deps {
				Ln(v)
			}
		}
	}

	Ln(strings.Repeat("-", 50))

	// Packages:
	if true {
		if true {
			// Request the HTML page.
			res, err := os.Open(os.ExpandEnv("$GOPATH/src/github.com/gagliardetto/depnet/test-data/PACKAGE.html"))
			if err != nil {
				Fataln(err)
			}
			deps, err := depnetloader.ExtractDependentsFromReader(res)
			if err != nil {
				panic(err)
			}
			for _, v := range deps {
				Ln(v)
			}
		} else {
			// Request the HTML page.
			res, err := http.Get("https://github.com/eslint/eslint/network/dependents?dependent_type=PACKAGE")
			if err != nil {
				Fataln(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			}
			deps, err := depnetloader.ExtractDependentsFromReader(res.Body)
			if err != nil {
				panic(err)
			}
			for _, v := range deps {
				Ln(v)
			}
		}
	}

	repos := []string{
		"eslint/eslint",     // NPM - javascript
		"numpy/numpy",       // python
		"symfony/symfony",   // Composer - php
		"dotnet/maui",       // dotnet - C#
		"apache/maven",      // Maven - java
		"rubygems/rubygems", // rubygems - ruby
		"yarnpkg/yarn",      // yarn - javascript
	}
	for _, repo := range repos { // Node:
		Ln(LimeBG(repo))
		{
			info, err :=
				depnetloader.NewLoader(repo).
					Type(depnetloader.TYPE_REPOSITORY).
					GetInfo()
			if err != nil {
				panic(err)
			}
			spew.Dump(info)
		}
		{
			count := int64(0)
			err :=
				depnetloader.NewLoader(repo).
					Type(depnetloader.TYPE_REPOSITORY).
					DoWithCallback(func(dep string) bool {
						count++
						Ln(dep)
						if count > 1000 {
							return false
						}
						return true
					})
			if err != nil {
				panic(err)
			}
		}
	}

}
