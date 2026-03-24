package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/romain325/doc-thor/server/auth"
	"github.com/romain325/doc-thor/server/config"
	"github.com/romain325/doc-thor/server/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <username> <password>\n", os.Args[0])
		os.Exit(1)
	}

	username := os.Args[1]
	password := os.Args[2]

	cfg := config.Load()

	db, err := gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("failed to open database", slog.Any("err", err))
		os.Exit(1)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.User{}); err != nil {
		slog.Error("failed to migrate", slog.Any("err", err))
		os.Exit(1)
	}

	var existing models.User
	if err := db.Where("username = ?", username).First(&existing).Error; err == nil {
		slog.Error("user already exists", slog.String("username", username))
		os.Exit(1)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("failed to query user", slog.Any("err", err))
		os.Exit(1)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", slog.Any("err", err))
		os.Exit(1)
	}

	user := models.User{
		Username:     username,
		PasswordHash: hash,
		IsSuperuser:  true,
	}
	if err := db.Create(&user).Error; err != nil {
		slog.Error("failed to create superuser", slog.Any("err", err))
		os.Exit(1)
	}

	fmt.Printf("superuser %q created (id=%d)\n", username, user.ID)
}
