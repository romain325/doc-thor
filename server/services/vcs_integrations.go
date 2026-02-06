package services

import (
	"context"
	"errors"

	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/vcs"
	"gorm.io/gorm"
)

// CreateVCSIntegration creates a new VCS integration.
func CreateVCSIntegration(db *gorm.DB, integration *models.VCSIntegration) error {
	var count int64
	db.Model(&models.VCSIntegration{}).Where("name = ?", integration.Name).Count(&count)
	if count > 0 {
		return ErrAlreadyExists
	}

	// Validate provider exists
	if _, err := vcs.GetProvider(integration.Provider); err != nil {
		return err
	}

	return db.Create(integration).Error
}

// ListVCSIntegrations returns all VCS integrations.
func ListVCSIntegrations(db *gorm.DB) ([]models.VCSIntegration, error) {
	var integrations []models.VCSIntegration
	err := db.Find(&integrations).Error
	return integrations, err
}

// GetVCSIntegration returns a VCS integration by name.
func GetVCSIntegration(db *gorm.DB, name string) (*models.VCSIntegration, error) {
	var integration models.VCSIntegration
	if err := db.Where("name = ?", name).First(&integration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &integration, nil
}

// UpdateVCSIntegration updates an existing VCS integration.
func UpdateVCSIntegration(db *gorm.DB, name string, updates *models.VCSIntegration) (*models.VCSIntegration, error) {
	integration, err := GetVCSIntegration(db, name)
	if err != nil {
		return nil, err
	}

	if updates.Provider != "" && updates.Provider != integration.Provider {
		// Validate new provider exists
		if _, err := vcs.GetProvider(updates.Provider); err != nil {
			return nil, err
		}
		integration.Provider = updates.Provider
	}

	if updates.InstanceURL != "" {
		integration.InstanceURL = updates.InstanceURL
	}

	if updates.AccessToken != "" {
		integration.AccessToken = updates.AccessToken
	}

	if updates.WebhookSecret != "" {
		integration.WebhookSecret = updates.WebhookSecret
	}

	// Enabled can be explicitly set to false
	integration.Enabled = updates.Enabled

	if err := db.Save(integration).Error; err != nil {
		return nil, err
	}

	return integration, nil
}

// DeleteVCSIntegration deletes a VCS integration by name.
func DeleteVCSIntegration(db *gorm.DB, name string) error {
	integration, err := GetVCSIntegration(db, name)
	if err != nil {
		return err
	}

	// Check if any projects are using this integration
	var count int64
	db.Model(&models.Project{}).Where("vcs_config->>'integration_name' = ?", name).Count(&count)
	if count > 0 {
		return errors.New("cannot delete integration: projects are using it")
	}

	return db.Delete(integration).Error
}

// TestVCSIntegration tests the connection to a VCS integration.
func TestVCSIntegration(ctx context.Context, integration *models.VCSIntegration) error {
	provider, err := vcs.GetProvider(integration.Provider)
	if err != nil {
		return err
	}

	config := vcs.IntegrationConfig{
		InstanceURL:   integration.InstanceURL,
		AccessToken:   integration.AccessToken,
		WebhookSecret: integration.WebhookSecret,
	}

	// Test by fetching a dummy repository info (use a known test path or just validate credentials)
	// For now, we'll just verify we can create a client - actual providers should implement a health check
	_, err = provider.GetRepositoryInfo(ctx, config, "__test__")
	// Expect "not found" error, which means credentials work
	if err != nil && err.Error() != "failed to get project __test__: 404 {message: 404 Project Not Found}" {
		return err
	}

	return nil
}
