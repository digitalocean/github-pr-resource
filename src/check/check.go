package check

import (
	"fmt"
	"path"
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
		// We loop, but there should only be 0 or 1.
		for _, c := range p.GetCommits() {
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
					if allFilesMatch(files, glob) {
						continue
					}
				}

				// Path is provided but no files match it.
				if glob := request.Source.Path; glob != "" {
					if !anyFilesMatch(files, glob) {
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

type commits []models.Commit

// True if all files match the glob pattern.
func allFilesMatch(files []string, glob string) bool {
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

// True if one file matches the glob pattern.
func anyFilesMatch(files []string, glob string) bool {
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
