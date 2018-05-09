package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itsdalmo/github-pr-resource/src/models"
)

// Run (business logic)
func Run(request models.GetRequest, outputDir string) (*models.GetResponse, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}
	if err := request.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %s", err)
	}

	// Write version so we can reuse it in PUT.
	path := filepath.Join(outputDir, ".git", "resource")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %s", err)
	}
	b, err := json.Marshal(request.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version: %s", err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "version.json"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write version: %s", err)
	}

	return &models.GetResponse{
		Version:  request.Version,
		Metadata: nil,
	}, nil
}

func commitMetadata(commit *models.Commit) []models.Metadata {
	// var m []models.Metadata
	// m = append(m, models.Metadata{
	// 	Name:  "name",
	// 	Value: "",
	// })
	return nil
}
