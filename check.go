package resource

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/ryanuber/go-glob"
)

// Check (business logic)
func Check(request CheckRequest, manager Github) (CheckResponse, error) {
	var response CheckResponse

	pulls, err := manager.ListOpenPullRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

Loop:
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
		if !p.Tip.CommittedDate.Time.After(request.Version.CommittedDate) {
			continue
		}

		// Fetch files once if paths/ignore_paths are specified.
		var files []string

		if len(request.Source.Paths) > 0 || len(request.Source.IgnorePaths) > 0 {
			files, err = manager.ListModifiedFiles(p.Number)
			if err != nil {
				return nil, fmt.Errorf("failed to list modified files: %s", err)
			}
		}

		// Skip version if no files match the specified paths.
		if len(request.Source.Paths) > 0 {
			var wanted []string
			for _, pattern := range request.Source.Paths {
				wanted = append(wanted, FilterPath(files, pattern)...)
			}
			if len(wanted) == 0 {
				continue Loop
			}
		}

		// Skip version if all files are ignored.
		if len(request.Source.IgnorePaths) > 0 {
			wanted := files
			for _, pattern := range request.Source.IgnorePaths {
				wanted = FilterIgnorePath(wanted, pattern)
			}
			if len(wanted) == 0 {
				continue Loop
			}
		}
		response = append(response, NewVersion(p))
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != "" {
		response = append(response, request.Version)
	}
	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == "" {
		response = CheckResponse{response[len(response)-1]}
	}
	return response, nil
}

// ContainsSkipCI returns true if a string contains [ci skip] or [skip ci].
func ContainsSkipCI(s string) bool {
	re := regexp.MustCompile("(?i)\\[(ci skip|skip ci)\\]")
	return re.MatchString(s)
}

// FilterIgnorePath ...
func FilterIgnorePath(files []string, pattern string) []string {
	var out []string
	for _, file := range files {
		if !glob.Glob(pattern, file) {
			out = append(out, file)
		}
	}
	return out
}

// FilterPath ...
func FilterPath(files []string, pattern string) []string {
	var out []string
	for _, file := range files {
		if glob.Glob(pattern, file) {
			out = append(out, file)
		}
	}
	return out
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (r CheckResponse) Len() int {
	return len(r)
}

func (r CheckResponse) Less(i, j int) bool {
	return r[j].CommittedDate.After(r[i].CommittedDate)
}

func (r CheckResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
