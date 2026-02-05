package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/services"
	"gorm.io/gorm"
)

func CreateBuild(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		project, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var req struct {
			Ref string `json:"ref"`
			Tag string `json:"tag"`
		}
		// ref and tag are optional; ignore decode errors from empty bodies.
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

		build, err := services.CreateBuild(db, project.ID, req.Ref, req.Tag)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create build")
			return
		}
		writeJSON(w, http.StatusCreated, build)
	}
}

func ListBuilds(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		project, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		limit, offset := 20, 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if l, err := strconv.Atoi(v); err == nil && l > 0 {
				limit = l
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if o, err := strconv.Atoi(v); err == nil && o >= 0 {
				offset = o
			}
		}

		builds, err := services.ListBuilds(db, project.ID, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, builds)
	}
}

func GetBuild(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		idStr := chi.URLParam(r, "id")

		project, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid build id")
			return
		}

		build, err := services.GetBuild(db, project.ID, uint(id))
		if err != nil {
			writeError(w, http.StatusNotFound, "build not found")
			return
		}
		writeJSON(w, http.StatusOK, build)
	}
}

// ClaimPendingBuild is the builder-facing poll endpoint.  It atomically claims
// the oldest pending build (transitions it to running) and returns the job
// payload the builder needs to start work.  Returns 204 when the queue is empty.
func ClaimPendingBuild(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		build, project, err := services.ClaimPendingBuild(db)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"id":           strconv.FormatUint(uint64(build.ID), 10),
			"project_slug": project.Slug,
			"version":      build.Tag,
			"source_url":   project.SourceURL,
			"ref":          build.Ref,
			"docker_image": project.DockerImage,
		})
	}
}

// ReportBuildResult is the builder-facing endpoint for recording a completed
// job.  The build must currently be in "running" state; any other state yields
// 409 Conflict.
func ReportBuildResult(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid build id")
			return
		}

		var req struct {
			Status string `json:"status"`
			Error  string `json:"error"`
			Logs   string `json:"logs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Status != "success" && req.Status != "failed" {
			writeError(w, http.StatusBadRequest, "status must be success or failed")
			return
		}

		build, err := services.ReportBuildResult(db, uint(id), req.Status, req.Logs, req.Error)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
				writeError(w, http.StatusNotFound, "build not found")
				return
			}
			if errors.Is(err, services.ErrBuildNotRunning) {
				writeError(w, http.StatusConflict, "build is not in running state")
				return
			}
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, build)
	}
}
