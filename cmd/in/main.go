package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/itsdalmo/github-pr-resource/src/in"
	"github.com/itsdalmo/github-pr-resource/src/models"
)

func main() {
	var request models.GetRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		log.Fatalf("failed to unmarshal request: %s", err)
	}

	if len(os.Args) < 2 {
		log.Fatalf("missing arguments")
	}
	outputDir := os.Args[1]

	response, err := in.Run(request, outputDir)
	if err != nil {
		log.Fatalf("get failed: %s", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		log.Fatalf("failed to marshal response: %s", err)
	}
}
