package git

import (
	"fmt"
	"io"
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

func (g *Git) command(subcommand string, args []string) *exec.Cmd {
	args = append([]string{subcommand}, args...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Directory
	cmd.Stdout = g.Output
	cmd.Stderr = g.Output
	return cmd
}

// CloneAndMerge merges the tip of a pull request into the tip of base.
func (g *Git) CloneAndMerge(pr *models.PullRequest) error {
	var args []string

	if err := g.command("init", args).Run(); err != nil {
		return fmt.Errorf("init failed: %s", err)
	}
	args = []string{"add", "origin", g.URL.String() + ".git"}
	if err := g.command("remote", args).Run(); err != nil {
		return fmt.Errorf("failed to add origin: %s", err)
	}
	args = []string{"origin", pr.BaseRefName}
	if err := g.command("pull", args).Run(); err != nil {
		return fmt.Errorf("pull failed: %s", err)
	}
	args = []string{"user.name", "concourse-ci"}
	if err := g.command("config", args).Run(); err != nil {
		return fmt.Errorf("failed to configure git user: %s", err)
	}
	args = []string{"user.email", "concourse@local"}
	if err := g.command("config", args).Run(); err != nil {
		return fmt.Errorf("failed to configure git email: %s", err)
	}
	args = []string{"-q", "origin", fmt.Sprintf("pull/%s/head", strconv.Itoa(pr.Number))}
	if err := g.command("fetch", args).Run(); err != nil {
		return fmt.Errorf("fetch failed: %s", err)
	}
	args = []string{"-b", "pr", pr.BaseRefName}
	if err := g.command("checkout", args).Run(); err != nil {
		return fmt.Errorf("failed to checkout new branch: %s", err)
	}
	args = []string{pr.Tip.OID}
	if err := g.command("merge", args).Run(); err != nil {
		return fmt.Errorf("merge failed: %s", err)
	}
	return nil
}

// RevParseBase retrieves the SHA of the base branch.
func (g *Git) RevParseBase(pr *models.PullRequest) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", pr.BaseRefName)
	cmd.Dir = g.Directory
	sha, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}
