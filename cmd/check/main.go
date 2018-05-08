package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Println(err)
	}
}

func run() error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubql.NewClient(httpClient)
	{

		var q struct {
			Repository struct {
				PullRequests struct {
					Edges []struct {
						Node struct {
							Title       githubql.String
							HeadRefName githubql.String
							URL         githubql.String
							Author      struct {
								Login githubql.String
							}
							Commits struct {
								Edges []struct {
									Node struct {
										Commit struct {
											AbbreviatedOid githubql.String
											CommittedDate  githubql.DateTime
											Message        githubql.String
											Status         struct {
												Context struct {
													State githubql.StatusState
												} `graphql:"context(name:$contextName)"`
											}
										}
									}
								}
							} `graphql:"commits(last:$commitsLast)"`
						}
					}
				} `graphql:"pullRequests(last:$pullrequestLast,states:$pullrequestStates)"`
			} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`

			Viewer struct {
				Login      githubql.String
				CreatedAt  githubql.DateTime
				ID         githubql.ID
				DatabaseID githubql.Int
			}
		}
		variables := map[string]interface{}{
			"repositoryOwner":   githubql.String("itsdalmo"),
			"repositoryName":    githubql.String("test-repository"),
			"pullrequestLast":   githubql.Int(100),
			"pullrequestStates": []githubql.PullRequestState{githubql.PullRequestStateOpen},
			"commitsLast":       githubql.Int(1),
			"contextName":       githubql.String("concourse-ci/status"),
		}
		err := client.Query(context.Background(), &q, variables)
		if err != nil {
			return err
		}
		printJSON(q)
	}

	return nil
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
func printJSON(v interface{}) {
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		panic(err)
	}
}
