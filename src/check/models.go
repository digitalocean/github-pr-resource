package check

import "github.com/itsdalmo/github-pr-resource/src/models"

// Request ...
type Request struct {
	Source  models.Source  `json:"source"`
	Version models.Version `json:"version"`
}

// Response ...
type Response []models.Version

func (r Response) Len() int {
	return len(r)
}

func (r Response) Less(i, j int) bool {
	return r[j].PushedDate.After(r[i].PushedDate)
}

func (r Response) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
