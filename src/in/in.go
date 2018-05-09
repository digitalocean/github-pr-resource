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
		Version: request.Version,
	}, nil
	// manager, err := manager.New(request.Source.Repository, request.Source.AccessToken)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create manager: %s", err)
	// }
	// commit, err := manager.GetCommitByID(request.Version.ID)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get last commits: %s", err)
	// }

	// outputDir = filepath.Join(outputDir, ".git", "metadata")
	// if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
	// 	return nil, fmt.Errorf("failed to create output directory: %s", err)
	// }
	// if err := ioutil.WriteFile(filepath.Join(outputDir, "sha"), []byte(query.Node.Commit.OID), 0644); err != nil {
	// 	return nil, fmt.Errorf("failed to write commit sha: %s", err)
	// }

	// // Return the response
	// response.Version = request.Version
	// return &response, nil
}

// func imageMetadata(image *ec2.Image) []models.Metadata {
// 	var m []models.Metadata

// 	m = append(m, models.Metadata{
// 		Name:  "name",
// 		Value: aws.StringValue(image.Name),
// 	})

// 	m = append(m, models.Metadata{
// 		Name:  "owner_id",
// 		Value: aws.StringValue(image.OwnerId),
// 	})

// 	m = append(m, models.Metadata{
// 		Name:  "creation_date",
// 		Value: aws.StringValue(image.CreationDate),
// 	})

// 	m = append(m, models.Metadata{
// 		Name:  "virtualization_type",
// 		Value: aws.StringValue(image.VirtualizationType),
// 	})

// 	m = append(m, models.Metadata{
// 		Name:  "root_device_type",
// 		Value: aws.StringValue(image.RootDeviceType),
// 	})

// 	return m
// }
