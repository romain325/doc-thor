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

	config.ConfigureLogger()
	cfg := config.Load()
	db := config.PrepareDatabase(cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		// --- public ---
		r.Get("/health", routes.Health())
		r.Post("/auth/login", routes.Login(db, cfg.SessionTTLHours))

		// Webhooks (public - called by VCS platforms)
		routes.RegisterWebhookRoutes(r, db)

		// --- authenticated ---
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(db))

			routes.RegisterProjectRoutes(r, db)
			// Builds
			r.Post("/projects/{slug}/builds", routes.CreateBuild(db))
			r.Get("/projects/{slug}/builds", routes.ListBuilds(db))
			r.Get("/projects/{slug}/builds/{id}", routes.GetBuild(db))

			// Builder job endpoints
			r.Get("/builds/pending", routes.ClaimPendingBuild(db))
			r.Post("/builds/{id}/result", routes.ReportBuildResult(db))

			// Versions
			r.Get("/projects/{slug}/versions", routes.ListVersions(db))
			r.Put("/projects/{slug}/versions/{ver}", routes.UpdateVersion(db, cfg.NginxConfigDir, cfg.StorageConfig))

			// Auth (key management + introspection)
			r.Post("/auth/apikey", routes.CreateAPIKey(db))
			r.Get("/auth/me", routes.GetMe(db))

			// System
			r.Get("/backends", routes.Backends(cfg.StorageConfig))

			// VCS Integrations
			routes.RegisterVCSIntegrationRoutes(r, db)

			// Project Discovery
			routes.RegisterDiscoveryRoutes(r, db)
		})

	})

	slog.Info("doc-thor server listening", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("Server Error", slog.Any("err", err))
		os.Exit(1)
	}
}
