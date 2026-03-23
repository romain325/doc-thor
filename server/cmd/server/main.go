package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/romain325/doc-thor/server/auth"
	"github.com/romain325/doc-thor/server/config"
	"github.com/romain325/doc-thor/server/routes"
	"github.com/romain325/doc-thor/server/vcs"
	"github.com/romain325/doc-thor/server/vcs/github"
	"github.com/romain325/doc-thor/server/vcs/gitlab"
)

func main() {
	// Register VCS providers
	vcs.RegisterProvider(&gitlab.GitLabProvider{})
	vcs.RegisterProvider(&github.GitHubProvider{})

	cfg := config.Load()
	db := config.PrepareDatabase(cfg)
	config.ConfigureLogger()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- public ---
	r.Get("/api/v1/health", routes.Health())
	r.Post("/api/v1/auth/login", routes.Login(db, cfg.SessionTTLHours))

	// Webhooks (public - called by VCS platforms)
	routes.RegisterWebhookRoutes(r, db)

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
		r.Put("/api/v1/projects/{slug}/versions/{ver}", routes.UpdateVersion(db, cfg.NginxConfigDir, cfg.StorageConfig))

		// Auth (key management + introspection)
		r.Post("/api/v1/auth/apikey", routes.CreateAPIKey(db))
		r.Get("/api/v1/auth/me", routes.GetMe(db))

		// System
		r.Get("/api/v1/backends", routes.Backends(cfg.StorageConfig))

		// VCS Integrations
		routes.RegisterVCSIntegrationRoutes(r, db)

		// Project Discovery
		routes.RegisterDiscoveryRoutes(r, db)
	})

	slog.Info("doc-thor server listening", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("Server Error", slog.Any("err", err))
		os.Exit(1)
	}
}
