package check_test

import (
	"strings"
	"testing"

	"github.com/itsdalmo/github-pr-resource/src/check"
)

func TestContainsSkipCI(t *testing.T) {
	cases := map[string]bool{
		"(":                        false,
		"none":                     false,
		"[ci skip]":                true,
		"[skip ci]":                true,
		"trailing [skip ci]":       true,
		"[skip ci] leading":        true,
		"case[Skip CI]insensitive": true,
	}
	for c, expected := range cases {
		t.Run(c, func(t *testing.T) {
			actual := check.ContainsSkipCI(c)
			if actual != expected {
				t.Errorf("expected '%s' to return %v, but got %v", c, expected, actual)
			}
		})
	}
}

func TestAllFilesMatch(t *testing.T) {
	pattern := "test/*.txt"
	cases := []struct {
		Test     string
		Files    []string
		Expected bool
	}{
		{
			Test: "files that should match",
			Files: []string{
				"test/file1.txt",
				"test/file2.txt",
			},
			Expected: true,
		},
		{
			Test: "files that should not match",
			Files: []string{
				"test/file1.go",
				"test/file2.txt",
			},
			Expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.Test, func(t *testing.T) {
			actual, err := check.AllFilesMatch(c.Files, pattern)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if actual != c.Expected {
				t.Errorf("expected '%s' to return %v, but got %v", pattern, c.Expected, actual)
			}
		})
	}
}

func TestAnyFilesMatch(t *testing.T) {
	pattern := "test/*.go"
	cases := []struct {
		Test     string
		Files    []string
		Expected bool
	}{
		{
			Test: "files that should match",
			Files: []string{
				"test/file1.go",
				"test/file2.txt",
			},
			Expected: true,
		},
		{
			Test: "files that should not match",
			Files: []string{
				"test/file1.txt",
				"test/file2.txt",
			},
			Expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.Test, func(t *testing.T) {
			actual, err := check.AnyFilesMatch(c.Files, pattern)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if actual != c.Expected {
				t.Errorf("expected '%s' to match any of these files:\n%s", pattern, strings.Join(c.Files, ", "))
			}
		})
	}
}
