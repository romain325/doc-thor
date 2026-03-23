package config

import (
	"log/slog"
	"os"

	"github.com/romain325/doc-thor/server/auth"
	"github.com/romain325/doc-thor/server/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func PrepareDatabase(cfg Config) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to open database", slog.Any("err", err))
		os.Exit(1)
	}

	// WAL mode + single-writer cap keeps SQLite safe under goroutine concurrency.
	db.Exec("PRAGMA journal_mode=WAL")
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&models.Project{},
		&models.Build{},
		&models.Version{},
		&models.User{},
		&models.Token{},
		&models.VCSIntegration{},
	); err != nil {
		slog.Error("Failed to migrate", slog.Any("err", err))
		os.Exit(1)
	}

	seedUser(db, cfg)
	return db

}

// seedUser creates the initial admin account when INITIAL_USER/INITIAL_PASSWORD
// are set and no users exist yet.  Idempotent — does nothing after first run.
func seedUser(db *gorm.DB, cfg Config) {
	if cfg.InitialUser == "" || cfg.InitialPassword == "" {
		return
	}
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}
	hash, err := auth.HashPassword(cfg.InitialPassword)
	if err != nil {
		slog.Error("Failed to hash initial password", slog.Any("err", err))
		os.Exit(1)
	}
	user := models.User{Username: cfg.InitialUser, PasswordHash: hash}
	if err := db.Create(&user).Error; err != nil {
		slog.Error("failed to create initial user", slog.Any("err", err))
	}
	slog.Info("Created initial user", slog.Any("user", cfg.InitialUser))
}
