package in

import "github.com/itsdalmo/github-pr-resource/src/models"

// Parameters ...
type Parameters struct{}

// Request ...
type Request struct {
	Source  models.Source  `json:"source"`
	Version models.Version `json:"version"`
	Params  Parameters     `json:"params"`
}

// Response ...
type Response struct {
	Version  models.Version  `json:"version"`
	Metadata models.Metadata `json:"metadata,omitempty"`
}
