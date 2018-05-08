package models

import (
	"errors"
	"time"
)

// Source represents the configuration for the resource.
type Source struct {
	Context     string `json:"context"`
	Repository  string `json:"repository"`
	AccessToken string `json:"access_token"`
}

// Validate the source configuration.
func (s *Source) Validate() error {
	if s.Context == "" {
		return errors.New("context must be set")
	}
	// TODO: Regexp this one?
	if s.Repository == "" {
		return errors.New("repository must be set")
	}
	return nil
}

// Metadata for the resource.
type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Version for the resource.
type Version struct {
	PR         string    `json:"pr"`
	Ref        string    `json:"ref"`
	PushedDate time.Time `json:"pushed_date"`
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (p CheckResponse) Len() int {
	return len(p)
}

func (p CheckResponse) Less(i, j int) bool {
	return p[j].PushedDate.After(p[i].PushedDate)
}

func (p CheckResponse) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// PutParameters for the resource.
type PutParameters struct {
	Template  string            `json:"template"`
	VarFile   string            `json:"var_file"`
	Variables map[string]string `json:"variables"`
}

// Validate the put parameters.
func (p *PutParameters) Validate() error {
	if p.Template == "" {
		return errors.New("template must be set")
	}
	return nil
}

// PutRequest ...
type PutRequest struct {
	Source Source        `json:"source"`
	Params PutParameters `json:"params"`
}

// PutResponse ...
type PutResponse struct {
	Version  Version    `json:"version"`
	Metadata []Metadata `json:"metadata"`
}
