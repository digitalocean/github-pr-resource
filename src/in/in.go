package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/itsdalmo/github-pr-resource/src/git"
	"github.com/itsdalmo/github-pr-resource/src/manager"
	"github.com/itsdalmo/github-pr-resource/src/models"
)

// Run (business logic)
func Run(request models.GetRequest, outputDir string) (*models.GetResponse, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}
	if err := request.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %s", err)
	}

	manager, err := manager.New(request.Source.Repository, request.Source.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}

	// Retrieve info for metadata/clone
	commit, err := manager.GetCommitByID(request.Version.Commit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pull request: %s", err)
	}
	pull, err := manager.GetPullRequestByID(request.Version.PR)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pull request: %s", err)
	}

	g := git.New(request.Source.Repository, outputDir, os.Stderr)

	// Clone the PR at the given commit
	if err := g.CloneAndMerge(pull, commit); err != nil {
		return nil, err
	}
	sha, err := g.RevParseBase(pull)
	if err != nil {
		return nil, err
	}
	metadata := newMetadata(pull, commit, sha)

	// Write version and metadata for reuse in PUT
	path := filepath.Join(outputDir, ".git", "resource")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %s", err)
	}
	b, err := json.Marshal(request.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "version.json"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write version: %s", err)
	}
	b, err = json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "metadata.json"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %s", err)
	}

	return &models.GetResponse{
		Version:  request.Version,
		Metadata: metadata,
	}, nil
}

func newMetadata(pr models.PullRequest, commit models.Commit, baseSHA string) []models.Metadata {
	var m []models.Metadata

	m = append(m, models.Metadata{
		Name:  "pr",
		Value: strconv.Itoa(pr.Number),
	})

	m = append(m, models.Metadata{
		Name:  "url",
		Value: pr.URL,
	})

	m = append(m, models.Metadata{
		Name:  "head_sha",
		Value: commit.OID,
	})

	m = append(m, models.Metadata{
		Name:  "base_sha",
		Value: baseSHA,
	})

	m = append(m, models.Metadata{
		Name:  "message",
		Value: commit.Message,
	})

	m = append(m, models.Metadata{
		Name:  "author",
		Value: commit.Author.User.Login,
	})

	return m
}
