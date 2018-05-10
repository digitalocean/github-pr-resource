package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/itsdalmo/github-pr-resource/src/models"
	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

// New Manager
func New(repository, token string) (*Manager, error) {
	oauth := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client := oauth2.NewClient(context.Background(), oauth)
	owner, repository, err := parseRepository(repository)
	if err != nil {
		return nil, err
	}
	return &Manager{
		V3:         github.NewClient(client),
		V4:         githubql.NewClient(client),
		Owner:      owner,
		Repository: repository,
	}, nil
}

func parseRepository(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", errors.New("malformed repository")
	}
	return parts[0], parts[1], nil
}

// Manager for handling requests to the Github V3 and V4 APIs.
type Manager struct {
	V3         *github.Client
	V4         *githubql.Client
	Repository string
	Owner      string
}

// GetLastCommits gets the last commit on all open Pull requests (costs 1/5000).
// TODO: Pagination.
func (m *Manager) GetLastCommits() ([]models.PullRequestCommits, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Edges []struct {
					Node models.PullRequestCommits
				}
				PageInfo struct {
					EndCursor   githubql.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first:$prFirst,states:$prStates, after:$prCursor)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	vars := map[string]interface{}{
		"repositoryOwner": githubql.String(m.Owner),
		"repositoryName":  githubql.String(m.Repository),
		"prFirst":         githubql.Int(100),
		"prStates":        []githubql.PullRequestState{githubql.PullRequestStateOpen},
		"prCursor":        (*githubql.String)(nil),
		"commitsLast":     githubql.Int(1),
	}

	var response []models.PullRequestCommits
	for {
		if err := m.V4.Query(context.Background(), &query, vars); err != nil {
			return nil, err
		}
		for _, p := range query.Repository.PullRequests.Edges {
			response = append(response, p.Node)
		}
		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		vars["prCursor"] = query.Repository.PullRequests.PageInfo.EndCursor
	}
	return response, nil
}

// GetCommitByID ... (zero cost).
func (m *Manager) GetCommitByID(objectID string) (models.Commit, error) {
	var query struct {
		Node struct {
			Commit models.Commit `graphql:"... on Commit"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubql.ID(objectID),
	}
	if err := m.V4.Query(context.Background(), &query, vars); err != nil {
		return models.Commit{}, err
	}
	return query.Node.Commit, nil
}

// GetPullRequestByID ... (zero cost).
func (m *Manager) GetPullRequestByID(objectID string) (models.PullRequest, error) {
	var query struct {
		Node struct {
			PullRequest models.PullRequest `graphql:"... on PullRequest"`
		} `graphql:"node(id:$nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": githubql.ID(objectID),
	}
	if err := m.V4.Query(context.Background(), &query, vars); err != nil {
		return models.PullRequest{}, err
	}
	return query.Node.PullRequest, nil
}

// SetCommitStatus for a given commit (not supported by V4 API).
func (m *Manager) SetCommitStatus(subjectID, ctx, status string) error {
	commit, err := m.GetCommitByID(subjectID)
	if err != nil {
		return err
	}

	// Create context
	c := "concourse-ci"
	if ctx != "" {
		c = strings.Join([]string{c, ctx}, "/")
	}

	_, _, err = m.V3.Repositories.CreateStatus(
		context.Background(),
		m.Owner,
		m.Repository,
		commit.OID,
		&github.RepoStatus{
			State:       github.String(strings.ToLower(status)),
			TargetURL:   github.String(os.Getenv("ATC_EXTERNAL_URL")),
			Description: github.String(fmt.Sprintf("Concourse CI build %s", status)),
			Context:     github.String(c),
		},
	)
	return err
}

// GetChangedFiles in a PullRequest (not supported by V4 API).
// TODO: Pagination.
func (m *Manager) GetChangedFiles(pr int) ([]string, error) {
	var files []string
	result, _, err := m.V3.PullRequests.ListFiles(
		context.Background(),
		m.Owner,
		m.Repository,
		pr,
		nil,
	)
	if err != nil {
		return nil, err
	}
	for _, f := range result {
		files = append(files, *f.Filename)
	}
	return files, nil
}

// AddComment to a PullRequest or issue (cost 1).
func (m *Manager) AddComment(subjectID string, comment string) error {
	var mutation struct {
		AddComment struct {
			Subject struct {
				ID githubql.ID
			}
		} `graphql:"addComment(input: $input)"`
	}
	input := githubql.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubql.String(comment),
	}
	err := m.V4.Mutate(context.Background(), &mutation, input, nil)
	return err
}

// CloneRepository ...
func (m *Manager) CloneRepository(pr string, comment string) error {
	id, err := strconv.Atoi(pr)
	if err != nil {
		return fmt.Errorf("failed to convert pr number to int: %s", err)
	}
	_, _, err = m.V3.Issues.CreateComment(
		context.Background(),
		m.Owner,
		m.Repository,
		id,
		&github.IssueComment{
			Body: github.String(comment),
		},
	)
	if err != nil {
		return err
	}
	return nil
}
