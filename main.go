package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"

	"github.com/gagliardetto/depnet/depnetloader"
	ghc "github.com/gagliardetto/gh-client"
	. "github.com/gagliardetto/utilz"
	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

var (
	gitCommitSHA = ""
	ghClient     *ghc.Client
)

type M map[string]interface{}

func main() {
	var ghToken string
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	app := &cli.App{
		Name:        "depnet",
		Version:     gitCommitSHA,
		Description: "Unofficial Github Dependency Network CLI — https://github.com/gagliardetto/depnet",
		Before: func(c *cli.Context) error {
			if ghToken == "" {
				return nil
			}
			// Setup a new github client:
			ghClient = ghc.NewClient(ghToken)

			ghc.ResponseCallback = func(resp *github.Response) {
				if resp == nil {
					return
				}
				if resp.Rate.Remaining < 1000 {
					Warnf(
						"GitHub API rate: remaining %v/%v; resetting in %s",
						resp.Rate.Remaining,
						resp.Rate.Limit,
						resp.Rate.Reset.Sub(time.Now()).Round(time.Second),
					)
				}
			}

			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "token",
				Usage:       "GitHub API token need for enriching the results with repo info.",
				Destination: &ghToken,
				EnvVar:      "GH_TOKEN",
			},
			&cli.StringFlag{
				Name:  "type",
				Usage: "Type of dependents to select (default=REPOSITORY).",
			},
			&cli.StringFlag{
				Name:  "pkg",
				Usage: "Select a specific subpackage.",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "How many results to output.",
			},
			&cli.BoolFlag{
				Name:  "info",
				Usage: "Print dependents stats and exit.",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as json.",
			},
			&cli.BoolFlag{
				Name:  "rich",
				Usage: "Enrich JSON output with repository info.",
			},
			&cli.BoolFlag{
				Name:  "pretty",
				Usage: "Pretty-fy JSON; this is for debug purposes only.",
			},
		},
		Action: func(c *cli.Context) error {
			depnetloader.EnableDebug = true

			target := c.Args().First()

			if target == "" {
				cli.ShowAppHelp(c)
				return nil
			}
			Errorln(LimeBG(target))

			asJSON := c.Bool("json")
			infoOnly := c.Bool("info")
			enrich := c.Bool("rich")
			pretty := c.Bool("pretty")
			limit := c.Int("limit")
			subPackage := c.String("pkg")

			typ := c.String("type")
			if typ == "" {
				typ = depnetloader.TYPE_REPOSITORY
			}

			if infoOnly {
				info, err := depnetloader.NewLoader(target).
					Type(typ).
					GetInfo()
				if err != nil {
					panic(err)
				}

				JSON(pretty, info)
				return nil
			}

			{
				ghCache := map[string]*github.Repository{}
				count := 0
				err := depnetloader.
					NewLoader(target).
					SubPackage(subPackage).
					Type(typ).
					DoWithCallback(func(dependent string) bool {
						count++

						if limit > 0 && count > limit {
							return false
						}
						if asJSON {
							res := M{
								"full_name": dependent,
							}

							if enrich {
								if ghClient == nil {
									panic("The --rich mode needs a github token to function.")
								}
								owner, repo, err := depnetloader.SplitOwnerRepo(dependent)
								if err != nil {
									panic(err)
								}
								ghRepo, ok := ghCache[dependent]
								if !ok {
									ghRepo, err = ghClient.GetRepo(owner, repo)
									if err != nil {
										panic(err)
									}
									ghCache[dependent] = ghRepo
								}
								res["repo"] = ghRepo
							}
							JSON(pretty, res)
						} else {
							Ln(dependent)
						}
						return true
					})
				if err != nil {
					panic(err)
				}
			}
			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// JSON pretty prints v to stdout. It panics on any error.
func JSON(pretty bool, v interface{}) {
	w := json.NewEncoder(os.Stdout)
	if pretty {
		w.SetIndent("", " ")
	}
	err := w.Encode(v)
	if err != nil {
		panic(err)
	}
}
