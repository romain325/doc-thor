package routes

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/services"
	"gorm.io/gorm"
)

func CreateProject(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p models.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if p.Slug == "" || p.Name == "" || p.SourceURL == "" {
			writeError(w, http.StatusBadRequest, "slug, name, and source_url are required")
			return
		}
		if err := services.CreateProject(db, &p); err != nil {
			if errors.Is(err, services.ErrAlreadyExists) {
				writeError(w, http.StatusConflict, "project with this slug already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusCreated, p)
	}
}

// projectWithVersions is the wire shape returned by ListProjects.  It embeds
// the base Project and adds the published version tags that generate.py needs
// to render nginx server blocks.
type projectWithVersions struct {
	models.Project
	Versions []string `json:"versions"`
	Latest   string   `json:"latest"`
}

func ListProjects(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := services.ListProjects(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}

		resp := make([]projectWithVersions, len(projects))
		for i, p := range projects {
			versions, err := services.ListVersions(db, p.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "database error")
				return
			}
			pwv := projectWithVersions{Project: p, Versions: []string{}}
			for _, v := range versions {
				if !v.Published {
					continue
				}
				pwv.Versions = append(pwv.Versions, v.Tag)
				if v.IsLatest {
					pwv.Latest = v.Tag
				}
			}
			resp[i] = pwv
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func GetProject(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		p, err := services.GetProject(db, slug)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func UpdateProject(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		var updates models.Project
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		p, err := services.UpdateProject(db, slug, &updates)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "update failed")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func DeleteProject(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		if err := services.DeleteProject(db, slug); err != nil {
			if errors.Is(err, services.ErrNotFound) {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
