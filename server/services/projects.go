package services

import (
	"errors"

	"github.com/romain325/doc-thor/server/models"
	"gorm.io/gorm"
)

func CreateProject(db *gorm.DB, p *models.Project) error {
	var count int64
	db.Model(&models.Project{}).Where("slug = ?", p.Slug).Count(&count)
	if count > 0 {
		return ErrAlreadyExists
	}
	return db.Create(p).Error
}

func ListProjects(db *gorm.DB) ([]models.Project, error) {
	var out []models.Project
	err := db.Find(&out).Error
	return out, err
}

func GetProject(db *gorm.DB, slug string) (*models.Project, error) {
	var p models.Project
	if err := db.Where("slug = ?", slug).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func UpdateProject(db *gorm.DB, slug string, updates *models.Project) (*models.Project, error) {
	p, err := GetProject(db, slug)
	if err != nil {
		return nil, err
	}
	if updates.Name != "" {
		p.Name = updates.Name
	}
	if updates.SourceURL != "" {
		p.SourceURL = updates.SourceURL
	}
	if updates.DockerImage != "" {
		p.DockerImage = updates.DockerImage
	}
	// VCSConfig is updated via separate VCS integration endpoints
	if err := db.Save(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func DeleteProject(db *gorm.DB, slug string) error {
	p, err := GetProject(db, slug)
	if err != nil {
		return err
	}
	db.Where("project_id = ?", p.ID).Delete(&models.Build{})
	db.Where("project_id = ?", p.ID).Delete(&models.Version{})
	return db.Delete(p).Error
}
