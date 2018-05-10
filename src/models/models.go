package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shurcooL/githubql"
)

// Source represents the configuration for the resource.
type Source struct {
	Repository    string `json:"repository"`
	AccessToken   string `json:"access_token"`
	Path          string `json:"path"`
	IgnorePath    string `json:"ignore_path"`
	DisableCISkip string `json:"disable_ci_skip"`
}

// Validate the source configuration.
func (s *Source) Validate() error {
	if s.AccessToken == "" {
		return errors.New("access_token must be set")
	}
	// TODO: Regexp this one?
	if s.Repository == "" {
		return errors.New("repository must be set")
	}
	return nil
}

// Metadata for the resource.
type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Version for the resource. ID is the Github Global ID.
type Version struct {
	PR         string    `json:"pr"`
	Commit     string    `json:"commit"`
	PushedDate time.Time `json:"pushed,omitempty"`
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (p CheckResponse) Len() int {
	return len(p)
}

func (p CheckResponse) Less(i, j int) bool {
	return p[j].PushedDate.After(p[i].PushedDate)
}

func (p CheckResponse) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// GetParameters for the resource.
type GetParameters struct{}

// Validate the get parameters.
func (p *GetParameters) Validate() error {
	return nil
}

// GetRequest ...
type GetRequest struct {
	Source  Source        `json:"source"`
	Version Version       `json:"version"`
	Params  GetParameters `json:"params"`
}

// GetResponse ...
type GetResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

// PutParameters for the resource.
type PutParameters struct {
	Path        string `json:"path"`
	Context     string `json:"context"`
	Status      string `json:"status"`
	CommentFile string `json:"comment_file"`
	Comment     string `json:"comment"`
}

// Validate the put parameters.
func (p *PutParameters) Validate() error {
	if p.Status == "" {
		return nil
	}
	// Make sure we are setting an allowed status
	var allowedStatus bool

	status := strings.ToLower(p.Status)
	allowed := []string{"success", "pending", "failure", "error"}

	for _, a := range allowed {
		if status == a {
			allowedStatus = true
		}
	}

	if !allowedStatus {
		return fmt.Errorf("unknown status: %s", p.Status)
	}

	return nil
}

// PutRequest ...
type PutRequest struct {
	Source Source        `json:"source"`
	Params PutParameters `json:"params"`
}

// PutResponse ...
type PutResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

// PullRequestCommits represents the GraphQL node with PR/Commit.
// https://developer.github.com/v4/object/pullrequest/
type PullRequestCommits struct {
	PullRequest
	Commits struct {
		Edges []struct {
			Node struct {
				Commit Commit
			}
		}
	} `graphql:"commits(last:$commitsLast)"`
}

// GetCommits returns the commits in a PullRequestAndCommits
func (p *PullRequestCommits) GetCommits() []Commit {
	var commits []Commit
	for _, c := range p.Commits.Edges {
		commits = append(commits, c.Node.Commit)
	}
	return commits
}

// PullRequest represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type PullRequest struct {
	ID     string
	Number int
	URL    string
}

// Commit represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type Commit struct {
	ID         string
	OID        string
	PushedDate githubql.DateTime
	Message    string
	Author     struct {
		User struct {
			Login string
		}
	}
}
