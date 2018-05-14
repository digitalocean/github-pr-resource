// +build e2e

package e2e_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itsdalmo/github-pr-resource/src/check"
	"github.com/itsdalmo/github-pr-resource/src/in"
	"github.com/itsdalmo/github-pr-resource/src/models"
	"github.com/itsdalmo/github-pr-resource/src/out"
)

const (
	targetCommitID      = "MDY6Q29tbWl0MTMyNTc2MjQ1OmE1MTE0ZjZhYjg5ZjRiNzM2NjU1NjQyYTExZThkMTVjZTM2M2Q4ODI="
	targetPullRequestID = "MDExOlB1bGxSZXF1ZXN0MTg3Mzg4MDE0"
	latestCommitID      = "MDY6Q29tbWl0MTMyNTc2MjQ1Ojg5MGE3ZTRmMGQ1YjA1YmRhOGVhMjFiOTFmNDYwNGUzZTAzMTM1ODE="
	latestPullRequestID = "MDExOlB1bGxSZXF1ZXN0MTg3Nzg4NjAy"
)

func TestCheck(t *testing.T) {
	t.Run("initial check works", func(t *testing.T) {
		input := check.Request{
			Source: models.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
			},
			Version: models.Version{
				PR:         "",
				Commit:     "",
				PushedDate: time.Time{},
			},
		}

		output, err := check.Run(input)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if n := len(output); n != 1 {
			t.Fatalf("expected 1 new version, got: %d", n)
		}
		v := output[0]
		if v.PR != latestPullRequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", latestPullRequestID, v.PR)
		}
		if v.Commit != latestCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", latestCommitID, v.Commit)
		}
	})

	t.Run("ignore paths work (latest commit only changes txt)", func(t *testing.T) {
		input := check.Request{
			Source: models.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
				IgnorePaths: []string{"*.txt"},
			},
			Version: models.Version{
				PR:         "",
				Commit:     "",
				PushedDate: time.Time{},
			},
		}

		output, err := check.Run(input)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if n := len(output); n != 1 {
			t.Fatalf("expected 1 new version, got: %d", n)
		}
		v := output[0]
		if v.PR != targetPullRequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", targetPullRequestID, v.PR)
		}
		if v.Commit != targetCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", targetCommitID, v.Commit)
		}
	})

	t.Run("paths work (latest commit does not change readme)", func(t *testing.T) {
		input := check.Request{
			Source: models.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
				Paths:       []string{"*.md"},
			},
			Version: models.Version{
				PR:         "",
				Commit:     "",
				PushedDate: time.Time{},
			},
		}

		output, err := check.Run(input)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if n := len(output); n != 1 {
			t.Fatalf("expected 1 new version, got: %d", n)
		}
		v := output[0]
		if v.PR != targetPullRequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", targetPullRequestID, v.PR)
		}
		if v.Commit != targetCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", targetCommitID, v.Commit)
		}
	})
}

func TestInAndOut(t *testing.T) {
	inRequest := in.Request{
		Source: models.Source{
			Repository:  "itsdalmo/test-repository",
			AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
		},
		Version: models.Version{
			PR:         targetPullRequestID,
			Commit:     targetCommitID,
			PushedDate: time.Time{},
		},
		Params: in.Parameters{},
	}
	outRequest := out.Request{
		Source: models.Source{
			Repository:  "itsdalmo/test-repository",
			AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
		},
		Params: out.Parameters{},
	}

	expectedVersion := strings.TrimSpace(`
{"pr":"MDExOlB1bGxSZXF1ZXN0MTg3Mzg4MDE0","commit":"MDY6Q29tbWl0MTMyNTc2MjQ1OmE1MTE0ZjZhYjg5ZjRiNzM2NjU1NjQyYTExZThkMTVjZTM2M2Q4ODI=","pushed":"0001-01-01T00:00:00Z"}
	`)

	expectedMetadata := strings.TrimSpace(`
[{"name":"pr","value":"4"},{"name":"url","value":"https://github.com/itsdalmo/test-repository/pull/4"},{"name":"head_sha","value":"a5114f6ab89f4b736655642a11e8d15ce363d882"},{"name":"base_sha","value":"93eeeedb8a16e6662062d1eca5655108977cc59a"},{"name":"message","value":"Push 2."},{"name":"author","value":"itsdalmo"}]
	`)

	dir, err := ioutil.TempDir("", "github-pr-resource")
	if err != nil {
		t.Fatalf("failed to create temporary directory")
	}
	defer os.RemoveAll(dir)

	t.Run("in/get works", func(t *testing.T) {
		output, err := in.Run(inRequest, dir)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if output.Version.PR != targetPullRequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", targetPullRequestID, output.Version.PR)
		}
		if output.Version.Commit != targetCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", targetCommitID, output.Version.Commit)
		}

		vFile := filepath.Join(dir, ".git", "resource", "version.json")
		v, err := ioutil.ReadFile(vFile)
		if err != nil {
			t.Fatalf("failed to read 'version.json': %s", err)
		}
		if string(v) != expectedVersion {
			t.Errorf("expected version:\n%s\nGot:\n%s\n", expectedVersion, string(v))
		}

		mFile := filepath.Join(dir, ".git", "resource", "metadata.json")
		m, err := ioutil.ReadFile(mFile)
		if err != nil {
			t.Fatalf("failed to read 'metadata.json': %s", err)
		}
		if string(m) != expectedMetadata {
			t.Errorf("expected metadata:\n%s\nGot:\n%s\n", expectedMetadata, string(m))
		}
	})

	t.Run("out/put works", func(t *testing.T) {
		output, err := out.Run(outRequest, dir)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if output.Version.PR != targetPullRequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", targetPullRequestID, output.Version.PR)
		}
		if output.Version.Commit != targetCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", targetCommitID, output.Version.Commit)
		}
	})
}
