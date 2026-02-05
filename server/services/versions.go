package services

import (
	"errors"

	"github.com/romain325/doc-thor/server/models"
	"gorm.io/gorm"
)

// CreateVersion registers a new published version for a project.  It does not
// touch is_latest — promotion is an explicit step via UpdateVersion.
func CreateVersion(db *gorm.DB, projectID, buildID uint, tag string) (*models.Version, error) {
	v := &models.Version{
		ProjectID: projectID,
		BuildID:   buildID,
		Tag:       tag,
		Published: true,
	}
	if err := db.Create(v).Error; err != nil {
		return nil, err
	}
	return v, nil
}

func ListVersions(db *gorm.DB, projectID uint) ([]models.Version, error) {
	var versions []models.Version
	err := db.Where("project_id = ?", projectID).Find(&versions).Error
	return versions, err
}

func UpdateVersion(db *gorm.DB, projectID uint, tag string, updates map[string]any) (*models.Version, error) {
	var v models.Version
	if err := db.Where("tag = ? AND project_id = ?", tag, projectID).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Ensure only one version carries is_latest per project.
	if val, ok := updates["is_latest"]; ok {
		if latest, ok := val.(bool); ok && latest {
			db.Model(&models.Version{}).
				Where("project_id = ? AND id != ?", projectID, v.ID).
				Update("is_latest", false)
		}
	}

	if err := db.Model(&v).Updates(updates).Error; err != nil {
		return nil, err
	}
	db.First(&v, v.ID) // reload after update
	return &v, nil
}
