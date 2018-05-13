package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/itsdalmo/github-pr-resource/src/models"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// NewGithubManager ...
func NewGithubManager(s *models.Source) (*GithubManager, error) {
	oauth := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: s.AccessToken},
	)
	client := oauth2.NewClient(context.Background(), oauth)
	owner, repository, err := parseRepository(s.Repository)
	if err != nil {
		return nil, err
	}
	return &GithubManager{
		V3:         github.NewClient(client),
		V4:         githubv4.NewClient(client),
		Owner:      owner,
		Repository: repository,
	}, nil
}

// GithubManager for handling requests to the Github V3 and V4 APIs.
type GithubManager struct {
	V3         *github.Client
	V4         *githubv4.Client
	Repository string
	Owner      string
}

// ListOpenPullRequests gets the last commit on all open pull requests.
func (m *GithubManager) ListOpenPullRequests() ([]*models.PullRequest, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Edges []struct {
					Node struct {
						models.PullRequestObject
						Commits struct {
							Edges []struct {
								Node struct {
									Commit models.CommitObject
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

	var response []*models.PullRequest
	for {
		if err := m.V4.Query(context.Background(), &query, vars); err != nil {
			return nil, err
		}
		for _, p := range query.Repository.PullRequests.Edges {
			for _, c := range p.Node.Commits.Edges {
				response = append(response, &models.PullRequest{
					PullRequestObject: &p.Node.PullRequestObject,
					Tip:               &c.Node.Commit,
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
func (m *GithubManager) ListModifiedFiles(prNumber int) ([]string, error) {
	var files []string

	opt := &github.ListOptions{
		PerPage: 100,
	}
	for {
		result, response, err := m.V3.PullRequests.ListFiles(
			context.Background(),
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
func (m *GithubManager) PostComment(objectID, comment string) error {
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
	err := m.V4.Mutate(context.Background(), &mutation, input, nil)
	return err
}

// GetPullRequestByID ...
func (m *GithubManager) GetPullRequestByID(objectID string) (*models.PullRequestObject, error) {
	var query struct {
		Node struct {
			PullRequest models.PullRequestObject `graphql:"... on PullRequest"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubv4.ID(objectID),
	}
	if err := m.V4.Query(context.Background(), &query, vars); err != nil {
		return nil, err
	}
	return &query.Node.PullRequest, nil
}

// GetCommitByID ...
func (m *GithubManager) GetCommitByID(objectID string) (*models.CommitObject, error) {
	var query struct {
		Node struct {
			Commit models.CommitObject `graphql:"... on Commit"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubv4.ID(objectID),
	}
	if err := m.V4.Query(context.Background(), &query, vars); err != nil {
		return nil, err
	}
	return &query.Node.Commit, nil
}

// UpdateCommitStatus for a given commit (not supported by V4 API).
func (m *GithubManager) UpdateCommitStatus(objectID, statusContext, status string) error {
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
		context.Background(),
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
