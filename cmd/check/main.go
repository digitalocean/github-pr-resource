package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/itsdalmo/github-pr-resource/src/check"
)

func main() {
	var request check.Request
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
