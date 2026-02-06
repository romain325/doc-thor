package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/services"
	"gorm.io/gorm"
)

// RegisterVCSIntegrationRoutes registers VCS integration routes.
func RegisterVCSIntegrationRoutes(r chi.Router, db *gorm.DB) {
	r.Route("/api/v1/integrations", func(r chi.Router) {
		r.Post("/", createVCSIntegration(db))
		r.Get("/", listVCSIntegrations(db))
		r.Route("/{name}", func(r chi.Router) {
			r.Get("/", getVCSIntegration(db))
			r.Put("/", updateVCSIntegration(db))
			r.Delete("/", deleteVCSIntegration(db))
			r.Post("/test", testVCSIntegration(db))
		})
	})
}

type vcsIntegrationCreateRequest struct {
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	InstanceURL   string `json:"instance_url"`
	AccessToken   string `json:"access_token"`
	WebhookSecret string `json:"webhook_secret"`
	Enabled       bool   `json:"enabled"`
}

func createVCSIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req vcsIntegrationCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate required fields
		var missing []string
		if req.Name == "" {
			missing = append(missing, "name")
		}
		if req.Provider == "" {
			missing = append(missing, "provider")
		}
		if req.InstanceURL == "" {
			missing = append(missing, "instance_url")
		}
		if req.AccessToken == "" {
			missing = append(missing, "access_token")
		}
		if req.WebhookSecret == "" {
			missing = append(missing, "webhook_secret")
		}
		if len(missing) > 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Missing required fields: %v", missing))
			return
		}

		// Convert to model
		integration := &models.VCSIntegration{
			Name:          req.Name,
			Provider:      req.Provider,
			InstanceURL:   req.InstanceURL,
			AccessToken:   req.AccessToken,
			WebhookSecret: req.WebhookSecret,
			Enabled:       req.Enabled,
		}

		if err := services.CreateVCSIntegration(db, integration); err != nil {
			if err == services.ErrAlreadyExists {
				writeError(w, http.StatusConflict, "Integration with this name already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, integration)
	}
}

func listVCSIntegrations(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		integrations, err := services.ListVCSIntegrations(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, integrations)
	}
}

func getVCSIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		integration, err := services.GetVCSIntegration(db, name)
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, integration)
	}
}

type vcsIntegrationUpdateRequest struct {
	Provider      string `json:"provider,omitempty"`
	InstanceURL   string `json:"instance_url,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

func updateVCSIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		var req vcsIntegrationUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Convert to model update struct
		updates := &models.VCSIntegration{
			Provider:      req.Provider,
			InstanceURL:   req.InstanceURL,
			AccessToken:   req.AccessToken,
			WebhookSecret: req.WebhookSecret,
		}
		if req.Enabled != nil {
			updates.Enabled = *req.Enabled
		}

		integration, err := services.UpdateVCSIntegration(db, name, updates)
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, integration)
	}
}

func deleteVCSIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		if err := services.DeleteVCSIntegration(db, name); err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func testVCSIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		integration, err := services.GetVCSIntegration(db, name)
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Integration not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := services.TestVCSIntegration(r.Context(), integration); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Connection successful",
		})
	}
}
