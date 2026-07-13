package assets

import "embed"

// JiraWorkItemSchema is the normalized work-item contract embedded in the CLI.
//
//go:embed schemas/jira-work-item.schema.json
var JiraWorkItemSchema []byte

// ImplementationRunSchema is the persisted dry-run execution contract.
//
//go:embed schemas/implementation-run.schema.json
var ImplementationRunSchema []byte

// SignedGovernanceRecordSchema is the common envelope for signed approvals
// and trusted offline exports.
//
//go:embed schemas/signed-governance-record.schema.json
var SignedGovernanceRecordSchema []byte

// Templates contains the shared files written by the init command.
//
//go:embed templates/*
var Templates embed.FS
