package assets

import "embed"

// JiraWorkItemSchema is the normalized work-item contract embedded in the CLI.
//
//go:embed schemas/jira-work-item.schema.json
var JiraWorkItemSchema []byte

// Templates contains the shared files written by the init command.
//
//go:embed templates/*
var Templates embed.FS
