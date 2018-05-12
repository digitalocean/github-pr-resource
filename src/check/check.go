package check

import (
	"fmt"
	"path"
	"regexp"
	"sort"

	"github.com/itsdalmo/github-pr-resource/src/manager"
	"github.com/itsdalmo/github-pr-resource/src/models"
)

// Run (business logic)
func Run(request models.CheckRequest) (models.CheckResponse, error) {
	var response models.CheckResponse

	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}

	manager, err := manager.New(request.Source.Repository, request.Source.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}
	pulls, err := manager.GetLastCommits()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

	for _, p := range pulls {
		// [ci skip]/[skip ci] in Pull request title
		if request.Source.DisableCISkip != "true" && ContainsSkipCI(p.Title) {
			continue
		}
		c, ok := p.GetLastCommit()
		if !ok {
			continue
		}
		// [ci skip]/[skip ci] in Commit message
		if request.Source.DisableCISkip != "true" && ContainsSkipCI(c.Message) {
			continue
		}
		// Filter out commits that are too old.
		if !c.PushedDate.Time.After(request.Version.PushedDate) {
			continue
		}

		// Filter on files if path or ignore_path is specified
		if request.Source.Path != "" || request.Source.IgnorePath != "" {
			files, err := manager.GetChangedFiles(p.Number)
			if err != nil {
				return nil, fmt.Errorf("failed to get changed files: %s", err)
			}

			// Ignore path is provided and ALL files match it.
			if glob := request.Source.IgnorePath; glob != "" {
				// If there are no files changed in a commit there is nothing to ignore
				if len(files) > 0 && AllFilesMatch(files, glob) {
					continue
				}
			}

			// Path is provided but no files match it.
			if glob := request.Source.Path; glob != "" {
				// If there are no files in a commit they cannot possibly match the glob.
				if len(files) == 0 || !AnyFilesMatch(files, glob) {
					continue
				}
			}
		}
		v := models.Version{
			PR:         p.ID,
			Commit:     c.ID,
			PushedDate: c.PushedDate.Time,
		}
		response = append(response, v)
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != "" {
		response = append(response, request.Version)
	}
	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == "" {
		response = models.CheckResponse{response[len(response)-1]}
	}
	return response, nil
}

// ContainsSkipCI returns true if a string contains [ci skip] or [skip ci].
func ContainsSkipCI(s string) bool {
	re := regexp.MustCompile("(?i)[(ci skip|skip ci)]")
	return re.MatchString(s)
}

// AllFilesMatch returns true if all files match the glob.
func AllFilesMatch(files []string, glob string) bool {
	for _, file := range files {
		match, err := path.Match(glob, file)
		if err != nil {
			panic(err)
		}
		if !match {
			return false
		}
	}
	return true
}

// AnyFilesMatch returns true if ANY files match the glob.
func AnyFilesMatch(files []string, glob string) bool {
	for _, file := range files {
		match, err := path.Match(glob, file)
		if err != nil {
			panic(err)
		}
		if match {
			return true
		}
	}
	return false
}
