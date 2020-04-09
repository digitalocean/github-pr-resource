package resource

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/telia-oss/github-pr-resource/pullrequest"
)

func findPulls(since time.Time, gh Github) ([]pullrequest.PullRequest, error) {
	if since.IsZero() {
		since = time.Now().AddDate(-3, 0, 0)
	}
	return gh.ListOpenPullRequests(since)
}

// Check (business logic)
func Check(request CheckRequest, manager Github) (CheckResponse, error) {
	var response CheckResponse

	pulls, err := findPulls(request.Version.UpdatedDate, manager)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

	paths := request.Source.Paths
	iPaths := request.Source.IgnorePaths

	log.Println("total pulls found:", len(pulls))

	for _, p := range pulls {
		log.Printf("evaluate pull: %+v\n", p)
		if !newVersion(request, p) {
			log.Println("no new version found")
			continue
		}

		if len(paths)+len(iPaths) > 0 {
			log.Println("pattern/s configured")
			p.Files, err = pullRequestFiles(p.Number, manager)
			if err != nil {
				return nil, err
			}

			log.Println("paths configured:", paths)
			log.Println("ignore paths configured:", iPaths)
			log.Println("changed files found:", p.Files)

			// if `paths` is configured && NONE of the changed files match `paths` pattern/s
			if pullrequest.Patterns(paths)(p) {
				matches, err := pullrequest.Files(paths)(p.Files)
				if err != nil {
					log.Println("error identifying matching paths")
					continue
				}

				if len(matches) == 0 {
					log.Println("paths excluded pull")
					continue
				}
			}

			// if `ignore_paths` is configured && ALL of the changed files match `ignore_paths` pattern/s
			if pullrequest.Patterns(iPaths)(p) {
				matches, err := pullrequest.Files(iPaths)(p.Files)
				if err != nil {
					log.Println("error identifying matching ignore_paths")
					continue
				}

				if len(matches) == len(p.Files) {
					log.Println("ignore paths excluded pull")
					continue
				}
			}

			// Both `paths` and `ignore_paths` are defined, it is possible for the pull request
			// to contain files outside of `paths`
			if pullrequest.Patterns(paths)(p) && pullrequest.Patterns(iPaths)(p) {
				log.Println("paths and ignore_paths both defined")
				matches, err := pullrequest.Files(paths)(p.Files)
				if err != nil {
					log.Println("error identifying matching paths when both paths and ignore_paths are defined")
					continue
				}

				if len(matches) > 0 {
					matches, err = pullrequest.Files(iPaths)(matches)
					if err != nil {
						log.Println("error identifying matching ignore_paths when both paths and ignore_paths are defined")
						continue
					}
				}

				if len(matches) == 0 {
					continue
				}
			}
		}

		response = append(response, NewVersion(p))
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != 0 {
		log.Println("no new versions, use old")
		response = append(response, request.Version)
	}

	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == 0 {
		response = CheckResponse{response[len(response)-1]}
	}

	log.Println("version count in response:", len(response))
	log.Println("versions:", response)

	return response, nil
}

func newVersion(r CheckRequest, p pullrequest.PullRequest) bool {
	switch {
	// negative filters
	case pullrequest.SkipCI(r.Source.DisableCISkip)(p),
		pullrequest.BaseBranch(r.Source.BaseBranch)(p),
		pullrequest.ApprovedReviewCount(r.Source.RequiredReviewApprovals)(p),
		pullrequest.Labels(r.Source.Labels)(p),
		pullrequest.Fork(r.Source.DisableForks)(p):
		return false
	// positive filters
	case pullrequest.Created(r.Version.UpdatedDate)(p),
		pullrequest.BaseRefChanged()(p),
		pullrequest.BaseRefForcePushed()(p),
		pullrequest.HeadRefForcePushed()(p),
		pullrequest.Reopened()(p),
		pullrequest.BuildCI()(p),
		pullrequest.NewCommits(r.Version.UpdatedDate)(p):
		return true
	}

	return false
}

func pullRequestFiles(n int, manager Github) ([]string, error) {
	files, err := manager.GetChangedFiles(n)
	if err != nil {
		return nil, fmt.Errorf("failed to list modified files: %s", err)
	}

	return files, nil
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
	return r[j].UpdatedDate.After(r[i].UpdatedDate)
}

func (r CheckResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
