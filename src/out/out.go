package out

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/itsdalmo/github-pr-resource/src/manager"
	"github.com/itsdalmo/github-pr-resource/src/models"
)

// Run (business logic)
func Run(request Request, inputDir string) (*Response, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}
	if err := request.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %s", err)
	}
	path := filepath.Join(inputDir, request.Params.Path, ".git", "resource")

	// Version available after a GET step.
	var version models.Version
	content, err := ioutil.ReadFile(filepath.Join(path, "version.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read version from path: %s", err)
	}
	if err := json.Unmarshal(content, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version from file: %s", err)
	}

	// Metadata available after a GET step.
	var metadata models.Metadata
	content, err = ioutil.ReadFile(filepath.Join(path, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata from path: %s", err)
	}
	if err := json.Unmarshal(content, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from file: %s", err)
	}

	m, err := manager.NewGithubManager(&request.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}

	// Set status if specified
	if status := request.Params.Status; status != "" {
		if err := m.UpdateCommitStatus(version.Commit, request.Params.Context, status); err != nil {
			return nil, fmt.Errorf("failed to set status: %s", err)
		}
	}

	// Set comment if specified
	if comment := request.Params.Comment; comment != "" {
		err = m.PostComment(version.PR, comment)
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
			err = m.PostComment(version.PR, comment)
			if err != nil {
				return nil, fmt.Errorf("failed to post comment: %s", err)
			}
		}
	}

	return &Response{
		Version:  version,
		Metadata: metadata,
	}, nil
}
