package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/services"
	"gorm.io/gorm"
)

// RegisterDiscoveryRoutes registers project discovery routes.
func RegisterDiscoveryRoutes(r chi.Router, db *gorm.DB) {
	r.Post("/api/v1/integrations/{name}/discover", discoverProjects(db))
	r.Post("/api/v1/projects/import", importProject(db))
}

func discoverProjects(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		integrationName := chi.URLParam(r, "name")

		var req struct {
			Scope string `json:"scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Scope == "" {
			writeError(w, http.StatusBadRequest, "Scope is required")
			return
		}

		discoveredProjects, err := services.DiscoverProjects(r.Context(), db, services.DiscoverProjectsRequest{
			IntegrationName: integrationName,
			Scope:           req.Scope,
		})

		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"count":    len(discoveredProjects),
			"projects": discoveredProjects,
		})
	}
}

func importProject(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req services.ImportProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		if req.IntegrationName == "" {
			writeError(w, http.StatusBadRequest, "Integration name is required")
			return
		}

		if req.DiscoveredProject.DocThorConfig == nil {
			writeError(w, http.StatusBadRequest, "Project must have doc-thor configuration")
			return
		}

		project, err := services.ImportProject(r.Context(), db, req)
		if err != nil {
			if err == services.ErrAlreadyExists {
				writeError(w, http.StatusConflict, "Project with this slug already exists")
				return
			}
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, project)
	}
}
