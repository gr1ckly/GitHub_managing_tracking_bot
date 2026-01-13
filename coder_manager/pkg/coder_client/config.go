package coder_client

import "time"

type Config struct {
	URL                     string
	AccessToken             string
	TemplateID              string
	TemplateVersionID       string
	TemplateVersionPresetID string
	User                    string
	EditorAppSlug           string
	AgentName               string
	WorkspaceReadyTimeout   time.Duration
}
