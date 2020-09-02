package resource

import (
	"encoding/json"
	"fmt"
	"strconv"

	meta "github.com/digitalocean/concourse-resource-library/metadata"
	"github.com/telia-oss/github-pr-resource/pullrequest"
)

func metadataFactory(pull pullrequest.PullRequest) meta.Metadata {
	var m meta.Metadata

	m.Add("pr", strconv.Itoa(pull.Number))
	m.Add("url", pull.URL)
	m.Add("head_name", pull.HeadRefName)
	m.Add("head_sha", pull.HeadRef.OID)
	m.Add("head_short_sha", pull.HeadRef.AbbreviatedOID)
	m.Add("base_name", pull.BaseRefName)
	m.Add("base_sha", pull.BaseRefOID)
	m.Add("message", pull.HeadRef.Message)
	m.Add("author", pull.HeadRef.Author)
	m.Add("events", fmt.Sprintf("%v", pull.Events))

	labels, _ := json.Marshal(pull.Labels)
	m.AddJSON("labels", string(labels))

	return m
}
