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

// GetLastCommits gets the last commit on all open Pull requests.
func (m *Manager) GetLastCommits(count int) ([]models.PullRequest, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Edges []struct {
					Node models.PullRequest
				}
			} `graphql:"pullRequests(last:$pullrequestLast,states:$pullrequestStates)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	vars := map[string]interface{}{
		"repositoryOwner":   githubql.String(m.Owner),
		"repositoryName":    githubql.String(m.Repository),
		"pullrequestLast":   githubql.Int(100),
		"pullrequestStates": []githubql.PullRequestState{githubql.PullRequestStateOpen},
		"commitsLast":       githubql.Int(count),
	}

	if err := m.V4.Query(context.Background(), &query, vars); err != nil {
		return nil, err
	}
	var response []models.PullRequest
	for _, p := range query.Repository.PullRequests.Edges {
		response = append(response, p.Node)
	}
	return response, nil
}

// GetCommitByID in a PullRequest.
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

// SetCommitStatus for a given commit.
func (m *Manager) SetCommitStatus(commitSHA, ctx, status string) error {
	c := []string{"concourse-ci"}
	if ctx != "" {
		c = append(c, ctx)
	}
	_, _, err := m.V3.Repositories.CreateStatus(
		context.Background(),
		m.Owner,
		m.Repository,
		commitSHA,
		&github.RepoStatus{
			State:       github.String(strings.ToLower(status)),
			TargetURL:   github.String(os.Getenv("ATC_EXTERNAL_URL")),
			Description: github.String(fmt.Sprintf("Concourse CI build %s", status)),
			Context:     github.String(strings.Join(c, "/")),
		},
	)
	return err
}

// GetChangedFiles in a PullRequest.
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

// AddComment in a PullRequest.
func (m *Manager) AddComment(pr string, comment string) error {
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
