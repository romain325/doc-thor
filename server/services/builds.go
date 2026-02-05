package services

import (
	"errors"
	"time"

	"github.com/romain325/doc-thor/server/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func CreateBuild(db *gorm.DB, projectID uint, ref, tag string) (*models.Build, error) {
	b := &models.Build{
		ProjectID: projectID,
		Ref:       ref,
		Tag:       tag,
		Status:    "pending",
	}
	if err := db.Create(b).Error; err != nil {
		return nil, err
	}
	return b, nil
}

// ClaimPendingBuild atomically finds the oldest pending build, transitions it to
// running, and returns it together with its project.  Safe under SQLite's
// single-writer constraint without explicit row locking.  Returns ErrNotFound
// when the queue is empty.
func ClaimPendingBuild(db *gorm.DB) (*models.Build, *models.Project, error) {
	var b models.Build
	err := db.Transaction(func(tx *gorm.DB) error {
		// Silent logger: an empty queue is the normal idle state; letting GORM
		// log ErrRecordNotFound every poll cycle is just noise.
		quiet := tx.Session(&gorm.Session{Logger: tx.Logger.LogMode(logger.Silent)})
		if err := quiet.Where("status = ?", "pending").Order("created_at ASC").First(&b).Error; err != nil {
			return err
		}
		now := time.Now()
		b.Status = "running"
		b.StartedAt = &now
		return tx.Save(&b).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}

	var p models.Project
	if err := db.First(&p, b.ProjectID).Error; err != nil {
		return nil, nil, err
	}
	return &b, &p, nil
}

// ReportBuildResult records the outcome reported by a builder.  Only builds
// currently in "running" state may be finalised; any other status returns
// ErrBuildNotRunning.
func ReportBuildResult(db *gorm.DB, buildID uint, status, logs, errMsg string) (*models.Build, error) {
	var b models.Build
	if err := db.First(&b, buildID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if b.Status != "running" {
		return nil, ErrBuildNotRunning
	}

	now := time.Now()
	b.Status = status
	b.Logs = logs
	b.Error = errMsg
	b.FinishedAt = &now
	if err := db.Save(&b).Error; err != nil {
		return nil, err
	}

	// Successful build with a tag → publish the version immediately.
	if status == "success" && b.Tag != "" {
		if _, err := CreateVersion(db, b.ProjectID, b.ID, b.Tag); err != nil {
			return nil, err
		}
	}

	return &b, nil
}

func ListBuilds(db *gorm.DB, projectID uint, limit, offset int) ([]models.Build, error) {
	var builds []models.Build
	err := db.Where("project_id = ?", projectID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&builds).Error
	return builds, err
}

func GetBuild(db *gorm.DB, projectID, buildID uint) (*models.Build, error) {
	var b models.Build
	if err := db.Where("id = ? AND project_id = ?", buildID, projectID).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

