package in

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

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
	if pull.Mergeable != "MERGEABLE" {
		return nil, errors.New("pull request has merge conflict")
	}
	metadata := newMetadata(pull, commit)

	// Clone the PR at the given commit
	git := &Git{
		Directory: outputDir,
		Output:    os.Stderr,
	}
	if err := git.Clone(request.Source.Repository, pull.BaseRefName); err != nil {
		return nil, fmt.Errorf("clone failed: %s", err)
	}
	if err := git.Fetch(pull.HeadRefName, pull.Number); err != nil {
		return nil, fmt.Errorf("fetch failed: %s", err)
	}
	if err := git.Checkout(commit.OID); err != nil {
		return nil, fmt.Errorf("checkout failed: %s", err)
	}
	if err := git.Merge(pull.BaseRefName); err != nil {
		return nil, fmt.Errorf("pull request has merge conflict: %s", err)
	}

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

func newMetadata(pr models.PullRequest, commit models.Commit) []models.Metadata {
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
		Name:  "sha",
		Value: commit.OID,
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

// Git ...
type Git struct {
	Directory string
	Output    io.Writer
}

// Run ...
func (g *Git) Run(args []string, dir string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = g.Output
	cmd.Stderr = g.Output
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %s", err)
	}
	return cmd.Wait()
}

// Clone ...
func (g *Git) Clone(repository, baseName string) error {
	args := []string{
		"clone",
		"--branch",
		baseName,
		"https://github.com/" + repository + ".git",
		g.Directory,
	}
	return g.Run(args, "")
}

// Fetch ...
func (g *Git) Fetch(headName string, pr int) error {
	args := []string{
		"fetch",
		"-q",
		"origin",
		fmt.Sprintf("pull/%s/head:pr-%s", strconv.Itoa(pr), headName),
	}
	return g.Run(args, g.Directory)
}

// Checkout ...
func (g *Git) Checkout(ref string) error {
	args := []string{
		"checkout",
		"-b",
		"pr",
		ref,
	}
	return g.Run(args, g.Directory)
}

// Merge ...
func (g *Git) Merge(headName string) error {
	args := []string{
		"merge",
		headName,
	}
	return g.Run(args, g.Directory)
}
