package git

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/itsdalmo/github-pr-resource/src/models"
)

// New ...
func New(repository, token, dir string, output io.Writer) *Git {
	return &Git{
		Repository:  repository,
		AccessToken: token,
		Directory:   dir,
		Output:      output,
	}
}

// Git ...
type Git struct {
	Repository  string
	AccessToken string
	Directory   string
	Output      io.Writer
}

// Auth ...
func (g *Git) Auth() string {
	if g.AccessToken != "" {
		return g.AccessToken + "@"
	}
	return ""
}

func (g *Git) command(subcommand string, args []string) *exec.Cmd {
	args = append([]string{subcommand}, args...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Directory
	cmd.Stdout = g.Output
	cmd.Stderr = g.Output
	return cmd
}

// CloneAndMerge a given commit in the Pullrequest into the latest version of base.
func (g *Git) CloneAndMerge(pr models.PullRequest, commit models.Commit) error {
	var args []string

	if err := g.command("init", args).Run(); err != nil {
		return fmt.Errorf("init failed: %s", err)
	}
	args = []string{"add", "origin", fmt.Sprintf("https://%sgithub.com/%s.git", g.Auth(), g.Repository)}
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
	args = []string{commit.OID}
	if err := g.command("merge", args).Run(); err != nil {
		return fmt.Errorf("merge failed: %s", err)
	}
	return nil
}

// RevParseBase retrieves the SHA of the base branch.
func (g *Git) RevParseBase(pr models.PullRequest) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", pr.BaseRefName)
	cmd.Dir = g.Directory
	sha, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}
