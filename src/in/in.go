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
func Run(request Request, outputDir string) (*Response, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}

	manager, err := manager.NewGithubManager(&request.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}

	// Retrieve info for metadata/clone
	commit, err := manager.GetCommitByID(request.Version.Commit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pull request: %s", err)
	}
	pr, err := manager.GetPullRequestByID(request.Version.PR)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pull request: %s", err)
	}
	pull := &models.PullRequest{PullRequestObject: *pr, Tip: *commit}

	g, err := git.New(&request.Source, pull.Repository.URL, outputDir, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create new git client: %s", err)
	}

	// Clone the repository and fetch the PR
	if err := g.Init(); err != nil {
		return nil, err
	}
	if err := g.Pull(); err != nil {
		return nil, err
	}
	if err := g.Fetch(pull.Number); err != nil {
		return nil, err
	}

	// Create a branch from the base ref and merge PR into it
	baseSHA, err := g.RevParse(pull.BaseRefName)
	if err != nil {
		return nil, err
	}
	if err := g.Checkout(baseSHA); err != nil {
		return nil, err
	}
	if err := g.Merge(pull.Tip.OID); err != nil {
		return nil, err
	}

	// Create the metadata
	var metadata models.Metadata
	metadata.Add("pr", strconv.Itoa(pull.Number))
	metadata.Add("url", pull.URL)
	metadata.Add("head_sha", commit.OID)
	metadata.Add("base_sha", baseSHA)
	metadata.Add("message", commit.Message)
	metadata.Add("author", commit.Author.User.Login)

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

	return &Response{
		Version:  request.Version,
		Metadata: metadata,
	}, nil
}
