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
func Run(request models.PutRequest, inputDir string) (*models.PutResponse, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}
	if err := request.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %s", err)
	}

	// Version available after a GET step.
	var version models.Version
	path := filepath.Join(inputDir, request.Params.Path, ".git", "resource", "version.json")

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read version from path: %s", err)
	}
	if err := json.Unmarshal(content, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version from file: %s", err)
	}

	manager, err := manager.New(request.Source.Repository, request.Source.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %s", err)
	}

	// Set status if specified
	if status := request.Params.Status; status != "" {
		if err := manager.SetCommitStatus(version.Commit, request.Params.Context, status); err != nil {
			return nil, fmt.Errorf("failed to set status: %s", err)
		}
	}

	// Set comment if specified
	if comment := request.Params.Comment; comment != "" {
		err = manager.AddComment(version.PR, comment)
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
			err = manager.AddComment(version.PR, comment)
			if err != nil {
				return nil, fmt.Errorf("failed to post comment: %s", err)
			}
		}
	}

	return &models.PutResponse{
		Version: version,
	}, nil
}
