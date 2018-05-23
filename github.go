package resource

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Github for testing purposes.
//go:generate mockgen -destination=mocks/mock_github.go -package=mocks github.com/itsdalmo/github-pr-resource Github
type Github interface {
	ListOpenPullRequests() ([]*PullRequest, error)
	ListModifiedFiles(int) ([]string, error)
	PostComment(string, string) error
	GetPullRequestByID(string) (*PullRequestObject, error)
	GetCommitByID(string) (*CommitObject, error)
	UpdateCommitStatus(string, string, string) error
}

// GithubClient for handling requests to the Github V3 and V4 APIs.
type GithubClient struct {
	V3         *github.Client
	V4         *githubv4.Client
	Repository string
	Owner      string
}

// NewGithubClient ...
func NewGithubClient(s *Source) (*GithubClient, error) {
	owner, repository, err := parseRepository(s.Repository)
	if err != nil {
		return nil, err
	}

	client := oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: s.AccessToken},
	))

	var v3 *github.Client
	if s.V3Endpoint != "" {
		endpoint, err := url.Parse(s.V3Endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse v3 endpoint: %s", err)
		}
		v3, err = github.NewEnterpriseClient(endpoint.String(), endpoint.String(), client)
		if err != nil {
			return nil, err
		}
	} else {
		v3 = github.NewClient(client)
	}

	var v4 *githubv4.Client
	if s.V4Endpoint != "" {
		endpoint, err := url.Parse(s.V4Endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse v4 endpoint: %s", err)
		}
		v4 = githubv4.NewEnterpriseClient(endpoint.String(), client)
		if err != nil {
			return nil, err
		}
	} else {
		v4 = githubv4.NewClient(client)
	}

	return &GithubClient{
		V3:         v3,
		V4:         v4,
		Owner:      owner,
		Repository: repository,
	}, nil
}

// ListOpenPullRequests gets the last commit on all open pull requests.
func (m *GithubClient) ListOpenPullRequests() ([]*PullRequest, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Edges []struct {
					Node struct {
						PullRequestObject
						Commits struct {
							Edges []struct {
								Node struct {
									Commit CommitObject
								}
							}
						} `graphql:"commits(last:$commitsLast)"`
					}
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first:$prFirst,states:$prStates,after:$prCursor)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	vars := map[string]interface{}{
		"repositoryOwner": githubv4.String(m.Owner),
		"repositoryName":  githubv4.String(m.Repository),
		"prFirst":         githubv4.Int(100),
		"prStates":        []githubv4.PullRequestState{githubv4.PullRequestStateOpen},
		"prCursor":        (*githubv4.String)(nil),
		"commitsLast":     githubv4.Int(1),
	}

	var response []*PullRequest
	for {
		if err := m.V4.Query(context.TODO(), &query, vars); err != nil {
			return nil, err
		}
		for _, p := range query.Repository.PullRequests.Edges {
			for _, c := range p.Node.Commits.Edges {
				response = append(response, &PullRequest{
					PullRequestObject: p.Node.PullRequestObject,
					Tip:               c.Node.Commit,
				})
			}
		}
		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		vars["prCursor"] = query.Repository.PullRequests.PageInfo.EndCursor
	}
	return response, nil
}

// ListModifiedFiles in a pull request (not supported by V4 API).
func (m *GithubClient) ListModifiedFiles(prNumber int) ([]string, error) {
	var files []string

	opt := &github.ListOptions{
		PerPage: 100,
	}
	for {
		result, response, err := m.V3.PullRequests.ListFiles(
			context.TODO(),
			m.Owner,
			m.Repository,
			prNumber,
			opt,
		)
		if err != nil {
			return nil, err
		}
		for _, f := range result {
			files = append(files, *f.Filename)
		}
		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
	}
	return files, nil
}

// PostComment to a pull request or issue.
func (m *GithubClient) PostComment(objectID, comment string) error {
	var mutation struct {
		AddComment struct {
			Subject struct {
				ID githubv4.ID
			}
		} `graphql:"addComment(input: $input)"`
	}
	input := githubv4.AddCommentInput{
		SubjectID: objectID,
		Body:      githubv4.String(comment),
	}
	err := m.V4.Mutate(context.TODO(), &mutation, input, nil)
	return err
}

// GetPullRequestByID ...
func (m *GithubClient) GetPullRequestByID(objectID string) (*PullRequestObject, error) {
	var query struct {
		Node struct {
			PullRequest PullRequestObject `graphql:"... on PullRequest"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubv4.ID(objectID),
	}
	if err := m.V4.Query(context.TODO(), &query, vars); err != nil {
		return nil, err
	}
	return &query.Node.PullRequest, nil
}

// GetCommitByID ...
func (m *GithubClient) GetCommitByID(objectID string) (*CommitObject, error) {
	var query struct {
		Node struct {
			Commit CommitObject `graphql:"... on Commit"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubv4.ID(objectID),
	}
	if err := m.V4.Query(context.TODO(), &query, vars); err != nil {
		return nil, err
	}
	return &query.Node.Commit, nil
}

// UpdateCommitStatus for a given commit (not supported by V4 API).
func (m *GithubClient) UpdateCommitStatus(objectID, statusContext, status string) error {
	commit, err := m.GetCommitByID(objectID)
	if err != nil {
		return err
	}

	// Format context
	c := []string{"concourse-ci"}
	if statusContext == "" {
		c = append(c, "status")
	} else {
		c = append(c, statusContext)
	}
	statusContext = strings.Join(c, "/")

	// Format build page
	build := os.Getenv("ATC_EXTERNAL_URL")
	if build != "" {
		build = strings.Join([]string{build, "builds", os.Getenv("BUILD_ID")}, "/")
	}

	_, _, err = m.V3.Repositories.CreateStatus(
		context.TODO(),
		m.Owner,
		m.Repository,
		commit.OID,
		&github.RepoStatus{
			State:       github.String(strings.ToLower(status)),
			TargetURL:   github.String(build),
			Description: github.String(fmt.Sprintf("Concourse CI build %s", status)),
			Context:     github.String(statusContext),
		},
	)
	return err
}

func parseRepository(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", errors.New("malformed repository")
	}
	return parts[0], parts[1], nil
}
