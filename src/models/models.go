package models

import (
	"errors"
	"time"

	"github.com/shurcooL/githubv4"
)

// Source represents the configuration for the resource.
type Source struct {
	Repository    string   `json:"repository"`
	AccessToken   string   `json:"access_token"`
	Paths         []string `json:"path"`
	IgnorePaths   []string `json:"ignore_path"`
	DisableCISkip string   `json:"disable_ci_skip"`
}

// Validate the source configuration.
func (s *Source) Validate() error {
	if s.AccessToken == "" {
		return errors.New("access_token must be set")
	}
	if s.Repository == "" {
		return errors.New("repository must be set")
	}
	return nil
}

// Metadata output from get/put steps.
type Metadata []*MetadataField

// Add a MetadataField to the Metadata.
func (m *Metadata) Add(name, value string) {
	*m = append(*m, &MetadataField{Name: name, Value: value})
}

// MetadataField ...
type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Version communicated with Concourse. ID is the Github Global ID.
type Version struct {
	PR         string    `json:"pr"`
	Commit     string    `json:"commit"`
	PushedDate time.Time `json:"pushed,omitempty"`
}

// NewVersion constructs a new Version.
func NewVersion(p *PullRequest) Version {
	return Version{
		PR:         p.ID,
		Commit:     p.Tip.ID,
		PushedDate: p.Tip.PushedDate.Time,
	}
}

// PullRequest represents a pull request and includes the tip (commit).
type PullRequest struct {
	PullRequestObject
	Tip CommitObject
}

// PullRequestObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type PullRequestObject struct {
	ID          string
	Number      int
	Title       string
	URL         string
	BaseRefName string
	HeadRefName string
}

// CommitObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type CommitObject struct {
	ID         string
	OID        string
	PushedDate githubv4.DateTime
	Message    string
	Author     struct {
		User struct {
			Login string
		}
	}
}
