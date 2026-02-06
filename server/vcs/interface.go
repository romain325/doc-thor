package vcs

import (
	"context"
	"net/http"

	"github.com/romain325/doc-thor/server/models"
)

// Provider is the interface all VCS integrations must implement.
type Provider interface {
	// Name returns the provider identifier ("gitlab", "github", "gitea")
	Name() string

	// ValidateWebhook verifies the webhook signature and parses the payload.
	// Returns a normalized Event or an error if invalid/unsupported.
	ValidateWebhook(r *http.Request, secret string) (*Event, error)

	// DiscoverProjects scans the given scope (group, org, namespace) and
	// returns projects that have a .doc-thor.project.yaml file in their root.
	DiscoverProjects(ctx context.Context, config IntegrationConfig, scope string) ([]DiscoveredProject, error)

	// GetRepositoryInfo fetches metadata about a repository (default branch, clone URL, etc.)
	GetRepositoryInfo(ctx context.Context, config IntegrationConfig, repoPath string) (*RepositoryInfo, error)

	// RegisterWebhook creates a webhook on the VCS platform for the given project.
	// Returns the provider-specific webhook ID.
	RegisterWebhook(ctx context.Context, config IntegrationConfig, repoPath string, events []EventType, callbackURL string) (string, error)

	// UnregisterWebhook deletes a webhook by its provider-specific ID.
	UnregisterWebhook(ctx context.Context, config IntegrationConfig, repoPath, webhookID string) error
}

// IntegrationConfig contains the connection details for a VCS instance.
type IntegrationConfig struct {
	InstanceURL   string
	AccessToken   string
	WebhookSecret string
}

// Event is the normalized webhook event after parsing provider-specific payloads.
type Event struct {
	Type          EventType
	Repository    string // e.g., "mygroup/mydocs"
	Branch        string
	Tag           string
	Commit        string
	CommitMessage string
	Author        string
}

// EventType represents the type of VCS event.
type EventType string

const (
	EventPush EventType = "push"
	EventTag  EventType = "tag"
)

// DiscoveredProject represents a project found during discovery.
type DiscoveredProject struct {
	Name          string         `json:"name"`
	Path          string         `json:"path"` // full repo path: "group/subgroup/project"
	CloneURL      string         `json:"clone_url"`
	DefaultBranch string         `json:"default_branch"`
	HasDocThor    bool           `json:"has_doc_thor"` // true if .doc-thor.project.yaml exists
	DocThorConfig *DocThorConfig `json:"doc_thor_config,omitempty"` // parsed from .doc-thor.project.yaml
}

// DocThorConfig is parsed from .doc-thor.project.yaml in the repository root.
// This file explicitly declares a project's doc-thor configuration.
type DocThorConfig struct {
	Slug           string                 `yaml:"slug" json:"slug"`
	Name           string                 `yaml:"name" json:"name"`
	DockerImage    string                 `yaml:"docker_image" json:"docker_image"`
	BranchMappings []models.BranchMapping `yaml:"branch_mappings,omitempty" json:"branch_mappings,omitempty"`
}

// RepositoryInfo is metadata about a single repository.
type RepositoryInfo struct {
	FullPath      string
	DefaultBranch string
	CloneURL      string
	Description   string
}
