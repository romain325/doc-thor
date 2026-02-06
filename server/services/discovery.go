package services

import (
	"context"
	"fmt"

	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/vcs"
	"gorm.io/gorm"
)

// DiscoverProjectsRequest is the request to discover projects in a VCS scope.
type DiscoverProjectsRequest struct {
	IntegrationName string `json:"integration_name"`
	Scope           string `json:"scope"`
}

// DiscoverProjects scans a VCS scope for projects with .doc-thor.project.yaml.
func DiscoverProjects(ctx context.Context, db *gorm.DB, req DiscoverProjectsRequest) ([]vcs.DiscoveredProject, error) {
	// Get VCS integration
	integration, err := GetVCSIntegration(db, req.IntegrationName)
	if err != nil {
		return nil, err
	}

	if !integration.Enabled {
		return nil, fmt.Errorf("integration %s is disabled", req.IntegrationName)
	}

	// Get provider
	provider, err := vcs.GetProvider(integration.Provider)
	if err != nil {
		return nil, err
	}

	// Create config
	config := vcs.IntegrationConfig{
		InstanceURL:   integration.InstanceURL,
		AccessToken:   integration.AccessToken,
		WebhookSecret: integration.WebhookSecret,
	}

	// Discover projects
	return provider.DiscoverProjects(ctx, config, req.Scope)
}

// ImportProjectRequest is the request to import a discovered project.
type ImportProjectRequest struct {
	IntegrationName string                 `json:"integration_name"`
	DiscoveredProject vcs.DiscoveredProject `json:"discovered_project"`
	BranchMappings  []models.BranchMapping `json:"branch_mappings,omitempty"` // Optional override
	AutoPublish     bool                   `json:"auto_publish"`              // Apply to all mappings
	RegisterWebhook bool                   `json:"register_webhook"`          // Whether to register webhook
	CallbackURL     string                 `json:"callback_url"`              // Webhook callback URL
}

// ImportProject creates a project from a discovered project and optionally registers a webhook.
func ImportProject(ctx context.Context, db *gorm.DB, req ImportProjectRequest) (*models.Project, error) {
	if req.DiscoveredProject.DocThorConfig == nil {
		return nil, fmt.Errorf("discovered project has no doc-thor config")
	}

	config := req.DiscoveredProject.DocThorConfig

	// Create project
	project := &models.Project{
		Slug:        config.Slug,
		Name:        config.Name,
		SourceURL:   req.DiscoveredProject.CloneURL,
		DockerImage: config.DockerImage,
	}

	// Use branch mappings from request, or fall back to config file
	branchMappings := req.BranchMappings
	if len(branchMappings) == 0 && len(config.BranchMappings) > 0 {
		branchMappings = config.BranchMappings
	}

	// Apply auto_publish override if requested
	if req.AutoPublish {
		for i := range branchMappings {
			branchMappings[i].AutoPublish = true
		}
	}

	// Create VCS config if webhook registration is requested
	if req.RegisterWebhook {
		// Get VCS integration
		integration, err := GetVCSIntegration(db, req.IntegrationName)
		if err != nil {
			return nil, err
		}

		// Get provider
		provider, err := vcs.GetProvider(integration.Provider)
		if err != nil {
			return nil, err
		}

		// Prepare integration config
		integrationConfig := vcs.IntegrationConfig{
			InstanceURL:   integration.InstanceURL,
			AccessToken:   integration.AccessToken,
			WebhookSecret: integration.WebhookSecret,
		}

		// Determine which events to listen for based on branch mappings
		events := []vcs.EventType{vcs.EventPush}
		for _, mapping := range branchMappings {
			if mapping.Branch == "v*" || mapping.VersionTag == "${tag}" {
				events = append(events, vcs.EventTag)
				break
			}
		}

		// Register webhook
		webhookID, err := provider.RegisterWebhook(
			ctx,
			integrationConfig,
			req.DiscoveredProject.Path,
			events,
			req.CallbackURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to register webhook: %w", err)
		}

		// Set VCS config
		project.VCSConfig = &models.VCSConfig{
			IntegrationName: req.IntegrationName,
			WebhookID:       webhookID,
			BranchMappings:  branchMappings,
			AutoRegister:    true,
		}
	}

	// Create project in database
	if err := CreateProject(db, project); err != nil {
		// If project creation fails and webhook was registered, try to clean up
		if req.RegisterWebhook && project.VCSConfig != nil {
			integration, _ := GetVCSIntegration(db, req.IntegrationName)
			provider, _ := vcs.GetProvider(integration.Provider)
			integrationConfig := vcs.IntegrationConfig{
				InstanceURL:   integration.InstanceURL,
				AccessToken:   integration.AccessToken,
				WebhookSecret: integration.WebhookSecret,
			}
			_ = provider.UnregisterWebhook(ctx, integrationConfig, req.DiscoveredProject.Path, project.VCSConfig.WebhookID)
		}
		return nil, err
	}

	return project, nil
}
