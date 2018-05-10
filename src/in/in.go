package in

import (
	"bufio"
	"encoding/json"
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
	metadata := newMetadata(pull, commit)

	// Clone
	git := &Git{
		Directory: filepath.Join(outputDir, "experiment", "repo"),
		Output:    os.Stderr,
	}
	if err := git.Clone(
		"master",
		"https://github.com/itsdalmo/test-repository.git",
	); err != nil {
		return nil, err
	}
	if err := git.Fetch("itsdalmo-test-1", "1"); err != nil {
		return nil, err
	}
	if err := git.Checkout("2a125dab13a4c047dca178d49ada072617c691d6"); err != nil {
		return nil, err
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

// Git ...
type Git struct {
	Directory string
	Output    io.Writer
}

// Run ...
func (g *Git) Run(args []string, dir string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %s", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %s", err)
	}
	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			fmt.Fprintln(g.Output, s.Text())
		}
	}()
	return cmd.Wait()
}

// Clone ...
func (g *Git) Clone(baseName, url string) error {
	args := []string{
		"clone",
		"--branch",
		baseName,
		url,
		g.Directory,
	}
	return g.Run(args, "")
}

// Fetch ...
func (g *Git) Fetch(headName, branch string) error {
	args := []string{
		"fetch",
		"-q",
		"origin",
		fmt.Sprintf("pull/%s/merge:pr-%s", branch, headName),
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
