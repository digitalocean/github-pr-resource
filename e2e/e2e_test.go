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
)

const (
	latestCommitID      = "MDY6Q29tbWl0MTMyNTc2MjQ1OmE1MTE0ZjZhYjg5ZjRiNzM2NjU1NjQyYTExZThkMTVjZTM2M2Q4ODI="
	latestPullrequestID = "MDExOlB1bGxSZXF1ZXN0MTg3Mzg4MDE0"
)

func TestCheck(t *testing.T) {
	input := models.CheckRequest{
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

	t.Run("initial check works", func(t *testing.T) {
		output, err := check.Run(input)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if n := len(output); n != 1 {
			t.Fatalf("expected 1 new version, got: %d", n)
		}
		v := output[0]
		if v.PR != latestPullrequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", latestPullrequestID, v.PR)
		}
		if v.Commit != latestCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", latestCommitID, v.Commit)
		}
	})
}

func TestIn(t *testing.T) {
	input := models.GetRequest{
		Source: models.Source{
			Repository:  "itsdalmo/test-repository",
			AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
		},
		Version: models.Version{
			PR:         latestPullrequestID,
			Commit:     latestCommitID,
			PushedDate: time.Time{},
		},
		Params: models.GetParameters{},
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
		output, err := in.Run(input, dir)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if output.Version.PR != latestPullrequestID {
			t.Errorf("expected pull request to have id:\n%s\nGot:\n%s\n", latestPullrequestID, output.Version.PR)
		}
		if output.Version.Commit != latestCommitID {
			t.Errorf("expected commit to have id:\n%s\nGot:\n%s\n", latestCommitID, output.Version.Commit)
		}

		vFile := filepath.Join(dir, ".git", "resource", "version.json")
		v, err := ioutil.ReadFile(vFile)
		if err != nil {
			t.Fatalf("failed to read 'version.json': %s", err)
		}
		if string(v) != expectedVersion {
			t.Errorf("expected version:\n%s\nGot:\n%s\n", string(v), expectedVersion)
		}

		mFile := filepath.Join(dir, ".git", "resource", "metadata.json")
		m, err := ioutil.ReadFile(mFile)
		if err != nil {
			t.Fatalf("failed to read 'metadata.json': %s", err)
		}
		if string(m) != expectedMetadata {
			t.Errorf("expected metadata:\n%s\nGot:\n%s\n", string(m), expectedMetadata)
		}
	})
}
