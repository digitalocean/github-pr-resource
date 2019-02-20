package resource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Put (business logic)
func Put(request PutRequest, manager Github, inputDir string) (*PutResponse, error) {
	if err := request.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %s", err)
	}
	path := filepath.Join(inputDir, request.Params.Path, ".git", "resource")

	// Version available after a GET step.
	var version Version
	content, err := ioutil.ReadFile(filepath.Join(path, "version.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read version from path: %s", err)
	}
	if err := json.Unmarshal(content, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version from file: %s", err)
	}

	// Metadata available after a GET step.
	var metadata Metadata
	content, err = ioutil.ReadFile(filepath.Join(path, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata from path: %s", err)
	}
	if err := json.Unmarshal(content, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from file: %s", err)
	}

	// Set status if specified
	if status := request.Params.Status; status != "" {
		if err := manager.UpdateCommitStatus(version.Commit, request.Params.Context, status); err != nil {
			return nil, fmt.Errorf("failed to set status: %s", err)
		}
	}

	// Set comment if specified
	if comment := request.Params.Comment; comment != "" {
		err = manager.PostComment(version.PR, os.ExpandEnv(comment))
		if err != nil {
			return nil, fmt.Errorf("failed to post comment: %s", err)
		}
	}

	// Set comment from a file
	if cf := request.Params.CommentFile; cf != "" {
		path := filepath.Join(inputDir, request.Params.CommentFile)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read comment file: %s", err)
		}
		comment := string(content)
		if comment != "" {
			err = manager.PostComment(version.PR, os.ExpandEnv(comment))
			if err != nil {
				return nil, fmt.Errorf("failed to post comment: %s", err)
			}
		}
	}

	return &PutResponse{
		Version:  version,
		Metadata: metadata,
	}, nil
}

// PutRequest ...
type PutRequest struct {
	Source Source        `json:"source"`
	Params PutParameters `json:"params"`
}

// PutResponse ...
type PutResponse struct {
	Version  Version  `json:"version"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// PutParameters for the resource.
type PutParameters struct {
	Path        string `json:"path"`
	Context     string `json:"context"`
	Status      string `json:"status"`
	CommentFile string `json:"comment_file"`
	Comment     string `json:"comment"`
}

// Validate the put parameters.
func (p *PutParameters) Validate() error {
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
