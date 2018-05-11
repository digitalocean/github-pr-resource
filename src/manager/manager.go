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
		V4:         githubv4.NewClient(client),
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
	V4         *githubv4.Client
	Repository string
	Owner      string
}

// GetLastCommits gets the last commit on all open Pull requests (costs 1/5000).
func (m *Manager) GetLastCommits() ([]models.PullRequestCommits, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Edges []struct {
					Node models.PullRequestCommits
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first:$prFirst,states:$prStates, after:$prCursor)"`
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
		"nodeId": githubv4.ID(objectID),
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
		"nodeId": githubv4.ID(objectID),
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

	// Format context
	c := []string{"concourse-ci"}
	if ctx == "" {
		c = append(c, "status")
	} else {
		c = append(c, ctx)
	}
	ctx = strings.Join(c, "/")

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
			Context:     github.String(ctx),
		},
	)
	return err
}

// GetChangedFiles in a PullRequest (not supported by V4 API).
func (m *Manager) GetChangedFiles(pr int) ([]string, error) {
	var files []string

	opt := &github.ListOptions{
		PerPage: 100,
	}
	for {
		result, response, err := m.V3.PullRequests.ListFiles(
			context.Background(),
			m.Owner,
			m.Repository,
			pr,
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

// AddComment to a PullRequest or issue (cost 1).
func (m *Manager) AddComment(subjectID string, comment string) error {
	var mutation struct {
		AddComment struct {
			Subject struct {
				ID githubv4.ID
			}
		} `graphql:"addComment(input: $input)"`
	}
	input := githubv4.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubv4.String(comment),
	}
	err := m.V4.Mutate(context.Background(), &mutation, input, nil)
	return err
}
