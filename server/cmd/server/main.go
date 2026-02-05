package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/romain325/doc-thor/server/auth"
	"github.com/romain325/doc-thor/server/config"
	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/routes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
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
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	seedUser(db, cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- public ---
	r.Get("/api/v1/health", routes.Health())
	r.Post("/api/v1/auth/login", routes.Login(db, cfg.SessionTTLHours))

	// --- authenticated ---
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(db))

		// Projects
		r.Post("/api/v1/projects", routes.CreateProject(db))
		r.Get("/api/v1/projects", routes.ListProjects(db))
		r.Get("/api/v1/projects/{slug}", routes.GetProject(db))
		r.Put("/api/v1/projects/{slug}", routes.UpdateProject(db))
		r.Delete("/api/v1/projects/{slug}", routes.DeleteProject(db))

		// Builds
		r.Post("/api/v1/projects/{slug}/builds", routes.CreateBuild(db))
		r.Get("/api/v1/projects/{slug}/builds", routes.ListBuilds(db))
		r.Get("/api/v1/projects/{slug}/builds/{id}", routes.GetBuild(db))

		// Builder job endpoints
		r.Get("/api/v1/builds/pending", routes.ClaimPendingBuild(db))
		r.Post("/api/v1/builds/{id}/result", routes.ReportBuildResult(db))

		// Versions
		r.Get("/api/v1/projects/{slug}/versions", routes.ListVersions(db))
		r.Put("/api/v1/projects/{slug}/versions/{ver}", routes.UpdateVersion(db, cfg.NginxConfigDir, cfg.StorageEndpoint))

		// Auth (key management + introspection)
		r.Post("/api/v1/auth/apikey", routes.CreateAPIKey(db))
		r.Get("/api/v1/auth/me", routes.GetMe(db))

		// System
		r.Get("/api/v1/backends", routes.Backends(cfg.BuilderEndpoints, cfg.StorageEndpoint, cfg.StorageUseSSL))
	})

	log.Printf("doc-thor server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// seedUser creates the initial admin account when INITIAL_USER/INITIAL_PASSWORD
// are set and no users exist yet.  Idempotent — does nothing after first run.
func seedUser(db *gorm.DB, cfg config.Config) {
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
		log.Fatalf("failed to hash initial password: %v", err)
	}
	user := models.User{Username: cfg.InitialUser, PasswordHash: hash}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("failed to create initial user: %v", err)
	}
	log.Printf("created initial user: %s", cfg.InitialUser)
}
