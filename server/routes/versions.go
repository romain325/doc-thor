package routes

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/services"
	"gorm.io/gorm"
)

func ListVersions(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		project, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		versions, err := services.ListVersions(db, project.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, versions)
	}
}

func UpdateVersion(db *gorm.DB, nginxDir, storageEndpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		ver := chi.URLParam(r, "ver")

		project, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var req struct {
			Published *bool `json:"published"`
			IsLatest  *bool `json:"is_latest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		updates := map[string]any{}
		if req.Published != nil {
			updates["published"] = *req.Published
		}
		if req.IsLatest != nil {
			updates["is_latest"] = *req.IsLatest
		}
		if len(updates) == 0 {
			writeError(w, http.StatusBadRequest, "nothing to update")
			return
		}

		version, err := services.UpdateVersion(db, project.ID, ver, updates)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "update failed")
			return
		}

		// best-effort nginx sync; non-fatal if it fails
		services.SyncNginxConfig(db, project, nginxDir, storageEndpoint) //nolint:errcheck

		writeJSON(w, http.StatusOK, version)
	}
}
