package git

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/itsdalmo/github-pr-resource/src/models"
)

// New ...
func New(source *models.Source, commitURL string, dir string, output io.Writer) (*Git, error) {
	endpoint, err := url.Parse(commitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit url: %s", err)
	}
	endpoint.User = url.UserPassword("x-oauth-basic", source.AccessToken)

	return &Git{
		URL:       endpoint,
		Directory: dir,
		Output:    output,
	}, nil
}

// Git ...
type Git struct {
	URL       *url.URL
	Directory string
	Output    io.Writer
}

func (g *Git) command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Dir = g.Directory
	cmd.Stdout = g.Output
	cmd.Stderr = g.Output
	return cmd
}

// Init ...
func (g *Git) Init() error {
	if err := g.command("git", "init").Run(); err != nil {
		return fmt.Errorf("init failed: %s", err)
	}
	if err := g.command("git", "config", "user.name", "concourse-ci").Run(); err != nil {
		return fmt.Errorf("failed to configure git user: %s", err)
	}
	if err := g.command("git", "config", "user.email", "concourse@local").Run(); err != nil {
		return fmt.Errorf("failed to configure git email: %s", err)
	}
	return nil
}

// Pull ...
func (g *Git) Pull() error {
	cmd := g.command("git", "pull", g.URL.String()+".git")

	// Discard output to have zero chance of logging the access token.
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pull failed: %s", err)
	}
	return nil
}

// Fetch ...
func (g *Git) Fetch(prNumber int) error {
	cmd := g.command("git", "fetch", g.URL.String(), fmt.Sprintf("pull/%s/head", strconv.Itoa(prNumber)))

	// Discard output to have zero chance of logging the access token.
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fetch failed: %s", err)
	}
	return nil
}

// Checkout ...
func (g *Git) Checkout(name string) error {
	if err := g.command("git", "checkout", "-b", name).Run(); err != nil {
		return fmt.Errorf("failed to checkout new branch: %s", err)
	}
	return nil
}

// Merge ...
func (g *Git) Merge(sha string) error {
	if err := g.command("git", "merge", sha, "--no-stat").Run(); err != nil {
		return fmt.Errorf("merge failed: %s", err)
	}
	return nil
}

// RevParse retrieves the SHA of the given branch.
func (g *Git) RevParse(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = g.Directory
	sha, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}
