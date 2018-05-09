package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/itsdalmo/github-pr-resource/src/check"
	"github.com/itsdalmo/github-pr-resource/src/models"
)

func main() {
	var request models.CheckRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		log.Fatalf("failed to unmarshal request: %s", err)
	}

	response, err := check.Run(request)
	if err != nil {
		log.Fatalf("check failed: %s", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		log.Fatalf("failed to marshal response: %s", err)
	}
}
