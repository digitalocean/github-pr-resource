package resource_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	resource "github.com/telia-oss/github-pr-resource"
	"github.com/telia-oss/github-pr-resource/fakes"
	"github.com/telia-oss/github-pr-resource/pullrequest"
)

var (
	testPullRequests = []pullrequest.PullRequest{
		// earliest
		createTestPR(1, "master", true, false, false, false),
		createTestPR(2, "master", false, false, false, false),
		// new pr
		createTestPR(3, "master", false, false, true, false),
		// new pr w/old commitdate
		createTestPR(4, "master", false, false, true, true),
		// old pr w/old commitdate
		createTestPR(5, "master", false, true, false, true),
		createTestPR(6, "master", false, false, false, false),
		createTestPR(7, "develop", false, false, false, false),
		createTestPR(8, "master", true, false, false, false),
		// latest
	}
)

func TestCheck(t *testing.T) {
	tests := []struct {
		description  string
		source       resource.Source
		version      resource.Version
		files        [][]string
		pullRequests []pullrequest.PullRequest
		expected     resource.CheckResponse
	}{
		{
			description: "check returns the latest version if there is no previous",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			files:        [][]string{},
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[6]),
			},
		},
		{
			description: "check returns the latest version if there is no previous w/basebranch",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				BaseBranch:  "master",
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			files:        [][]string{},
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[5]),
			},
		},
		/*		{
					description: "check returns the previous version when its still latest",
					source: resource.Source{
						Repository:  "itsdalmo/test-repository",
						AccessToken: "oauthtoken",
					},
					version:      resource.NewVersion(testPullRequests[6]),
					pullRequests: testPullRequests,
					files:        [][]string{},
					expected: resource.CheckResponse{
						resource.NewVersion(testPullRequests[6]),
					},
				},
				{
					description: "check returns all new versions since the last",
					source: resource.Source{
						Repository:  "itsdalmo/test-repository",
						AccessToken: "oauthtoken",
					},
					version:      resource.NewVersion(testPullRequests[4]),
					pullRequests: testPullRequests,
					files:        [][]string{},
					expected: resource.CheckResponse{
						resource.NewVersion(testPullRequests[5]),
						resource.NewVersion(testPullRequests[6]),
					},
				},*/
		{
			description: "check supports specifying base branch",
			source: resource.Source{
				Repository:  "itsdalmo/test-repository",
				AccessToken: "oauthtoken",
				BaseBranch:  "develop",
			},
			version:      resource.Version{},
			pullRequests: testPullRequests,
			files:        [][]string{},
			expected: resource.CheckResponse{
				resource.NewVersion(testPullRequests[6]),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			github := new(fakes.FakeGithub)

			if tc.version.UpdatedDate.IsZero() {
				github.GetLatestOpenPullRequestReturns(tc.pullRequests, nil)
			} else {
				github.ListOpenPullRequestsReturns(tc.pullRequests, nil)
			}

			for i, file := range tc.files {
				github.GetChangedFilesReturnsOnCall(i, file, nil)
			}

			input := resource.CheckRequest{Source: tc.source, Version: tc.version}
			output, err := resource.Check(input, github)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, output)
			}
			if tc.version.UpdatedDate.IsZero() {
				assert.Equal(t, 1, github.GetLatestOpenPullRequestCallCount())
			} else {
				assert.Equal(t, 1, github.ListOpenPullRequestsCallCount())
			}
		})
	}
}
