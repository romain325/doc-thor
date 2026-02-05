package main

import (
	"errors"
	"fmt"
	"log"
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
		log.Fatalf("failed to open database: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	var existing models.User
	if err := db.Where("username = ?", username).First(&existing).Error; err == nil {
		log.Fatalf("user %q already exists", username)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatalf("failed to query user: %v", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Username:     username,
		PasswordHash: hash,
		IsSuperuser:  true,
	}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("failed to create superuser: %v", err)
	}

	fmt.Printf("superuser %q created (id=%d)\n", username, user.ID)
}
