package out

import (
	"fmt"
	"strings"

	"github.com/itsdalmo/github-pr-resource/src/models"
)

// Parameters for the resource.
type Parameters struct {
	Path        string `json:"path"`
	Context     string `json:"context"`
	Status      string `json:"status"`
	CommentFile string `json:"comment_file"`
	Comment     string `json:"comment"`
}

// Validate the put parameters.
func (p *Parameters) Validate() error {
	if p.Status == "" {
		return nil
	}
	// Make sure we are setting an allowed status
	var allowedStatus bool

	status := strings.ToLower(p.Status)
	allowed := []string{"success", "pending", "failure", "error"}

	for _, a := range allowed {
		if status == a {
			allowedStatus = true
		}
	}

	if !allowedStatus {
		return fmt.Errorf("unknown status: %s", p.Status)
	}

	return nil
}

// Request ...
type Request struct {
	Source models.Source `json:"source"`
	Params Parameters    `json:"params"`
}

// Response ...
type Response struct {
	Version  models.Version  `json:"version"`
	Metadata models.Metadata `json:"metadata,omitempty"`
}
