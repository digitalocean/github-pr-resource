package check

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/itsdalmo/github-pr-resource/src/manager"
	"github.com/itsdalmo/github-pr-resource/src/models"
	glob "github.com/ryanuber/go-glob"
)

// Run (business logic)
func Run(request Request) (Response, error) {
	var response Response

	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}

	manager, err := manager.NewGithubManager(&request.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}
	pulls, err := manager.ListOpenPullRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

	for _, p := range pulls {
		// [ci skip]/[skip ci] in Pull request title
		if request.Source.DisableCISkip != "true" && ContainsSkipCI(p.Title) {
			continue
		}
		// [ci skip]/[skip ci] in Commit message
		if request.Source.DisableCISkip != "true" && ContainsSkipCI(p.Tip.Message) {
			continue
		}
		// Filter out commits that are too old.
		if !p.Tip.PushedDate.Time.After(request.Version.PushedDate) {
			continue
		}

		// Filter on files if path or ignore_path is specified
		if request.Source.Path != "" || request.Source.IgnorePath != "" {
			files, err := manager.ListModifiedFiles(p.Number)
			if err != nil {
				return nil, fmt.Errorf("failed to list modified files: %s", err)
			}

			// Ignore path is provided and ALL files match it.
			if glob := request.Source.IgnorePath; glob != "" {
				if AllFilesMatch(files, glob) {
					continue
				}
			}

			// Path is provided but no files match it.
			if glob := request.Source.Path; glob != "" {
				// If there are no files in a commit they cannot possibly match the glob.
				if AnyFilesMatch(files, glob) {
					continue
				}
			}
		}
		response = append(response, models.NewVersion(p))
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != "" {
		response = append(response, request.Version)
	}
	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == "" {
		response = Response{response[len(response)-1]}
	}
	return response, nil
}

// ContainsSkipCI returns true if a string contains [ci skip] or [skip ci].
func ContainsSkipCI(s string) bool {
	re := regexp.MustCompile("(?i)\\[(ci skip|skip ci)\\]")
	return re.MatchString(s)
}

// AllFilesMatch returns true if all files match the glob.
func AllFilesMatch(files []string, pattern string) bool {
	// If there are no files changed in a commit there is nothing to ignore
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if !glob.Glob(pattern, file) {
			return false
		}
	}
	return true
}

// AnyFilesMatch returns true if ANY files match the glob.
func AnyFilesMatch(files []string, pattern string) bool {
	// If there are no files in a commit they cannot possibly match the glob.
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if glob.Glob(pattern, file) {
			return true
		}
	}
	return false
}
