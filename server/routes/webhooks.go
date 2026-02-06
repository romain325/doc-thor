package routes

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/romain325/doc-thor/server/models"
	"github.com/romain325/doc-thor/server/services"
	"github.com/romain325/doc-thor/server/vcs"
	"gorm.io/gorm"
)

// RegisterWebhookRoutes registers webhook routes.
func RegisterWebhookRoutes(r chi.Router, db *gorm.DB) {
	r.Post("/api/v1/webhooks/{provider}/{slug}", handleWebhook(db))
}

func handleWebhook(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := chi.URLParam(r, "provider")
		projectSlug := chi.URLParam(r, "slug")

		// 1. Load project from DB
		project, err := services.GetProject(db, projectSlug)
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "Project not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 2. Verify project has VCS config
		if project.VCSConfig == nil {
			writeError(w, http.StatusBadRequest, "Project not configured for webhooks")
			return
		}

		// 3. Load VCS integration config
		integration, err := services.GetVCSIntegration(db, project.VCSConfig.IntegrationName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "VCS integration not found")
			return
		}

		if !integration.Enabled {
			writeError(w, http.StatusBadRequest, "VCS integration is disabled")
			return
		}

		// 4. Get provider implementation
		vcsProvider, err := vcs.GetProvider(provider)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// 5. Validate and parse webhook
		event, err := vcsProvider.ValidateWebhook(r, integration.WebhookSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid webhook: "+err.Error())
			return
		}

		// 6. Match event to branch mappings
		var matchedMapping *models.BranchMapping
		for i := range project.VCSConfig.BranchMappings {
			mapping := &project.VCSConfig.BranchMappings[i]
			if matchesBranch(event.Branch, mapping.Branch) || matchesTag(event.Tag, mapping.Branch) {
				matchedMapping = mapping
				break
			}
		}

		if matchedMapping == nil {
			// No matching branch mapping - this is OK, just ignore
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "ignored",
				"message": fmt.Sprintf("No mapping for branch/tag: %s%s", event.Branch, event.Tag),
			})
			return
		}

		// 7. Resolve version tag
		versionTag := resolveVersionTag(matchedMapping.VersionTag, event)

		// 8. Create build job
		ref := event.Branch
		if event.Type == vcs.EventTag {
			ref = event.Tag
		}

		build, err := services.CreateBuild(db, project.ID, ref, versionTag)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create build: "+err.Error())
			return
		}

		// 9. TODO: Queue build for builder to pick up
		// This would typically involve pushing to a job queue

		// 10. TODO: If auto_publish is enabled, set up callback to publish after success
		// This would be handled by the build completion handler

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"status":      "accepted",
			"build_id":    build.ID,
			"version_tag": versionTag,
			"ref":         ref,
			"auto_publish": matchedMapping.AutoPublish,
		})
	}
}

// matchesBranch checks if a branch matches a pattern.
func matchesBranch(branch, pattern string) bool {
	if branch == "" {
		return false
	}

	// Exact match
	if branch == pattern {
		return true
	}

	// Glob pattern matching
	matched, _ := filepath.Match(pattern, branch)
	return matched
}

// matchesTag checks if a tag matches a pattern.
func matchesTag(tag, pattern string) bool {
	if tag == "" {
		return false
	}

	// Exact match
	if tag == pattern {
		return true
	}

	// Glob pattern matching
	matched, _ := filepath.Match(pattern, tag)
	return matched
}

// resolveVersionTag resolves template variables in version tag.
func resolveVersionTag(template string, event *vcs.Event) string {
	result := template

	// Replace ${branch}
	if event.Branch != "" {
		result = strings.ReplaceAll(result, "${branch}", event.Branch)
	}

	// Replace ${tag}
	if event.Tag != "" {
		// Strip 'v' prefix from tags if present (v1.2.3 -> 1.2.3)
		tag := event.Tag
		if strings.HasPrefix(tag, "v") && len(tag) > 1 {
			tag = tag[1:]
		}
		result = strings.ReplaceAll(result, "${tag}", tag)
	}

	return result
}
