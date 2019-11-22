package env

import "os"

// Check stores the env vars for the Check container
type Check struct {
	BuildPipelineName string
	BuildTeamName     string
	BuildTeamID       string
	ATCExternalURL    string
}

// ReadCheck parses the env vars into a Check struct
func ReadCheck() Check {
	c := Check{}
	c.BuildPipelineName = os.Getenv("BUILD_PIPELINE_NAME")
	c.BuildTeamName = os.Getenv("BUILD_TEAM_NAME")
	c.BuildTeamID = os.Getenv("BUILD_TEAM_ID")
	c.ATCExternalURL = os.Getenv("ATC_EXTERNAL_URL")

	return c
}
