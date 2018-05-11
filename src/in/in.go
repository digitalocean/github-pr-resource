package in

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

	// Clone the PR at the given commit
	git := &Git{
		Directory: outputDir,
		Output:    os.Stderr,
	}
	if err := git.Clone(request.Source.Repository, pull.BaseRefName); err != nil {
		return nil, fmt.Errorf("clone failed: %s", err)
	}
	sha, err := git.RevParse(pull.BaseRefName)
	if err != nil {
		return nil, fmt.Errorf("failed to get base ref: %s", err)
	}
	if err := git.Fetch(pull.Number); err != nil {
		return nil, fmt.Errorf("fetch failed: %s", err)
	}
	if err := git.Merge(commit.OID); err != nil {
		return nil, fmt.Errorf("merge failed: %s", err)
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

// Git ...
type Git struct {
	Repository string
	Directory  string
	Output     io.Writer
}

// Cmd ...
func (g *Git) Cmd(args []string, dir string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Stdout = g.Output
	cmd.Stderr = g.Output
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

// Clone ...
func (g *Git) Clone(repository, baseName string) error {
	args := []string{"clone", "--branch", baseName, "https://github.com/" + repository + ".git", g.Directory}
	err := g.Cmd(args, "").Run()
	return err
}

// Fetch ... (in case its a remote branch)
func (g *Git) Fetch(pr int) error {
	args := []string{"fetch", "-q", "origin", fmt.Sprintf("pull/%s/head", strconv.Itoa(pr))}
	err := g.Cmd(args, g.Directory).Run()
	return err
}

// Merge ...
func (g *Git) Merge(commitSHA string) error {
	args := []string{"merge", commitSHA}
	err := g.Cmd(args, g.Directory).Run()
	return err
}

// RevParse ...
func (g *Git) RevParse(refName string) (string, error) {
	args := []string{"rev-parse", "--verify", refName}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Directory
	sha, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}
