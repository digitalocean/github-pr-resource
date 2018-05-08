package check

import (
	"context"
	"errors"
	"golang.org/x/oauth2"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/itsdalmo/github-pr-resource/src/models"
	"github.com/shurcooL/githubql"
)

// Run (business logic)
func Run(request models.CheckRequest) (models.CheckResponse, error) {
	var response models.CheckResponse

	auth := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: request.Source.AccessToken},
	)
	client := oauth2.NewClient(context.Background(), auth)

	v3Client := github.NewClient(client)
	v4Client := githubql.NewClient(client)

	var query struct {
		Repository Repository `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	owner, repository, err := parseRepository(request.Source.Repository)
	if err != nil {
		return response, err
	}

	vars := map[string]interface{}{
		"repositoryOwner":   githubql.String(owner),
		"repositoryName":    githubql.String(repository),
		"pullrequestLast":   githubql.Int(100),
		"pullrequestStates": []githubql.PullRequestState{githubql.PullRequestStateOpen},
		"commitsLast":       githubql.Int(1),
	}

	if err := v4Client.Query(context.Background(), &query, vars); err != nil {
		return response, err
	}

	for _, p := range query.Repository.PullRequests.Edges {
		c := p.Node.Commits.Edges[0].Node.Commit
		v := models.Version{
			PR:         strconv.Itoa(int(p.Node.Number)),
			Ref:        string(c.AbbreviatedOid),
			PushedDate: c.PushedDate.Time,
		}
		if !v.PushedDate.After(request.Version.PushedDate) {
			continue
		}
		files, _, err := v3Client.PullRequests.ListFiles(
			context.Background(),
			owner,
			repository,
			int(p.Node.Number),
			nil,
		)
		if err != nil {
			return response, err
		}

		// Ignore path is provided and ALL files match it.
		if glob := request.Source.IgnorePath; glob != "" {
			if allFilesMatch(files, glob) {
				continue
			}
		}

		// Path is provided but no files match it.
		if glob := request.Source.Path; glob != "" {
			if !anyFilesMatch(files, glob) {
				continue
			}
		}

		response = append(response, v)
	}

	// Sort the versions chronologically (oldest to newest)
	if len(response) > 0 {
		sort.Sort(response)
	} else {
		response = append(response, request.Version)
	}

	return response, nil
}

// True if all files match the glob pattern.
func allFilesMatch(files []*github.CommitFile, glob string) bool {
	for _, file := range files {
		match, err := path.Match(glob, *file.Filename)
		if err != nil {
			panic(err)
		}
		if !match {
			return false
		}
	}
	return true
}

// True if one file matches the glob pattern.
func anyFilesMatch(files []*github.CommitFile, glob string) bool {
	for _, file := range files {
		match, err := path.Match(glob, *file.Filename)
		if err != nil {
			panic(err)
		}
		if match {
			return true
		}
	}
	return false
}

func parseRepository(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", errors.New("malformed repository")
	}
	return parts[0], parts[1], nil
}

// Repository https://developer.github.com/v4/object/repository/
type Repository struct {
	PullRequests PullRequests `graphql:"pullRequests(last:$pullrequestLast,states:$pullrequestStates)"`
}

// PullRequests https://developer.github.com/v4/object/pullrequest/
type PullRequests struct {
	Edges []struct {
		Node struct {
			Number      githubql.Int
			Title       githubql.String
			HeadRefName githubql.String
			URL         githubql.String
			Author      struct {
				Login githubql.String
			}
			Commits Commits `graphql:"commits(last:$commitsLast)"`
		}
	}
}

// Commits https://developer.github.com/v4/object/pullrequestcommitconnection/
type Commits struct {
	Edges []struct {
		Node struct {
			Commit Commit
		}
	}
}

// Commit https://developer.github.com/v4/object/commit/
type Commit struct {
	AbbreviatedOid githubql.String
	PushedDate     githubql.DateTime
	Message        githubql.String
}
