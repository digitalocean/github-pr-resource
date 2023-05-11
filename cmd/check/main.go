package main

import (
	"encoding/json"
	"log"
	"os"

	resource "github.com/telia-oss/github-pr-resource"
	rlog "github.com/telia-oss/github-pr-resource/log"
)

func main() {
	var request resource.CheckRequest

	// default DisableForks to true to prevent forks from being used as an attack
	// vector unless `disable_forks` is explicitly set to false in the JSON that
	// is unmarshaled.
	request.Source.DisableForks = true

	input := rlog.WriteStdin()
	defer rlog.Close()

	if err := json.Unmarshal(input, &request); err != nil {
		log.Fatalf("failed to unmarshal request: %s", err)
	}

	if err := request.Source.Validate(); err != nil {
		log.Fatalf("invalid source configuration: %s", err)
	}
	github, err := resource.NewGithubClient(&request.Source)
	if err != nil {
		log.Fatalf("failed to create github manager: %s", err)
	}
	response, err := resource.Check(request, github)
	if err != nil {
		log.Fatalf("check failed: %s", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		log.Fatalf("failed to marshal response: %s", err)
	}

	log.Println("check complete")
}
