package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/romain325/doc-thor/server/vcs"
	"gitlab.com/gitlab-org/api/client-go"
	"gopkg.in/yaml.v3"
)

// GitLabProvider implements the vcs.Provider interface for GitLab.
type GitLabProvider struct{}

// Name returns the provider identifier.
func (p *GitLabProvider) Name() string {
	return "gitlab"
}

// ValidateWebhook verifies the webhook signature and parses the payload.
func (p *GitLabProvider) ValidateWebhook(r *http.Request, secret string) (*vcs.Event, error) {
	// Validate X-Gitlab-Token header
	token := r.Header.Get("X-Gitlab-Token")
	if token != secret {
		return nil, fmt.Errorf("invalid webhook token")
	}

	// Parse event type
	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType != "Push Hook" && eventType != "Tag Push Hook" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	// Parse payload
	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"repository"`
		Commits []struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commits"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event := &vcs.Event{
		Repository: payload.Repository.PathWithNamespace,
	}

	if eventType == "Tag Push Hook" {
		event.Type = vcs.EventTag
		event.Tag = extractTag(payload.Ref) // refs/tags/v1.0.0 -> v1.0.0
	} else {
		event.Type = vcs.EventPush
		event.Branch = extractBranch(payload.Ref) // refs/heads/main -> main
	}

	if len(payload.Commits) > 0 {
		event.Commit = payload.Commits[0].ID
		event.CommitMessage = payload.Commits[0].Message
		event.Author = payload.Commits[0].Author.Name
	}

	return event, nil
}

// DiscoverProjects scans the given scope and returns projects with .doc-thor.project.yaml.
func (p *GitLabProvider) DiscoverProjects(ctx context.Context, config vcs.IntegrationConfig, scope string) ([]vcs.DiscoveredProject, error) {
	log.Printf("[discovery] Starting GitLab project discovery in scope: %s", scope)

	client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	var discovered []vcs.DiscoveredProject
	checkedCount := 0

	// First, try to fetch it as a single project (for --repo use case)
	// This works when scope is a full project path like "group/subgroup/project"
	singleProject, singleErr := p.discoverSingleProject(ctx, client, scope, &checkedCount)
	if singleErr == nil && len(singleProject) > 0 {
		discovered = append(discovered, singleProject...)
		log.Printf("[discovery] Discovery complete: checked %d repositories, found %d with .doc-thor.project.yaml", checkedCount, len(discovered))
		return discovered, nil
	}

	log.Printf("[discovery] Single project discovery failed: %v. Trying group discovery...", singleErr)

	// Try group discovery
	groupProjects, groupErr := p.discoverGroupProjects(ctx, client, scope, &checkedCount)
	if groupErr == nil {
		discovered = append(discovered, groupProjects...)
		log.Printf("[discovery] Discovery complete: checked %d repositories, found %d with .doc-thor.project.yaml", checkedCount, len(discovered))
		return discovered, nil
	}

	log.Printf("[discovery] Group discovery failed: %v. Trying user namespace discovery...", groupErr)

	// If group discovery fails, try user namespace discovery
	userProjects, userErr := p.discoverUserProjects(ctx, client, scope, &checkedCount)
	if userErr != nil {
		return nil, fmt.Errorf("failed to discover projects in scope %s (tried project, group, and user): project error: %v, group error: %v, user error: %v", scope, singleErr, groupErr, userErr)
	}

	discovered = append(discovered, userProjects...)
	log.Printf("[discovery] Discovery complete: checked %d repositories, found %d with .doc-thor.project.yaml", checkedCount, len(discovered))
	return discovered, nil
}

// discoverSingleProject tries to fetch a single project by its full path.
func (p *GitLabProvider) discoverSingleProject(ctx context.Context, client *gitlab.Client, projectPath string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	*checkedCount++
	log.Printf("[discovery] Checking single repository: %s", projectPath)

	// Try to get the project by its path
	proj, _, err := client.Projects.GetProject(projectPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectPath, err)
	}

	// Check if project has .doc-thor.project.yaml
	hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, proj)

	if !hasDocThor {
		log.Printf("[discovery]   ✗ No .doc-thor.project.yaml found")
		return nil, fmt.Errorf("project %s does not have .doc-thor.project.yaml", projectPath)
	}

	log.Printf("[discovery]   ✓ Found .doc-thor.project.yaml (slug: %s)", docThorConfig.Slug)
	return []vcs.DiscoveredProject{
		{
			Name:          proj.Name,
			Path:          proj.PathWithNamespace,
			CloneURL:      proj.HTTPURLToRepo,
			DefaultBranch: proj.DefaultBranch,
			HasDocThor:    true,
			DocThorConfig: docThorConfig,
		},
	}, nil
}

// discoverGroupProjects lists projects in a GitLab group.
func (p *GitLabProvider) discoverGroupProjects(ctx context.Context, client *gitlab.Client, groupPath string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		IncludeSubGroups: gitlab.Ptr(true),
		ListOptions:      gitlab.ListOptions{PerPage: 50},
	}

	var discovered []vcs.DiscoveredProject

	for {
		projects, resp, err := client.Groups.ListGroupProjects(groupPath, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list projects in group %s: %w", groupPath, err)
		}

		for _, proj := range projects {
			*checkedCount++
			log.Printf("[discovery] Checking repository [%d]: %s", *checkedCount, proj.PathWithNamespace)

			// Check if project has .doc-thor.project.yaml
			hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, proj)

			if hasDocThor {
				log.Printf("[discovery]   ✓ Found .doc-thor.project.yaml (slug: %s)", docThorConfig.Slug)
				discovered = append(discovered, vcs.DiscoveredProject{
					Name:          proj.Name,
					Path:          proj.PathWithNamespace,
					CloneURL:      proj.HTTPURLToRepo,
					DefaultBranch: proj.DefaultBranch,
					HasDocThor:    true,
					DocThorConfig: docThorConfig,
				})
			} else {
				log.Printf("[discovery]   ✗ No .doc-thor.project.yaml found")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return discovered, nil
}

// discoverUserProjects lists projects owned by a GitLab user.
func (p *GitLabProvider) discoverUserProjects(ctx context.Context, client *gitlab.Client, username string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	// First, get the user ID from the username
	users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to find user %s: %w", username, err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user %s not found", username)
	}

	userID := users[0].ID

	// List user's projects
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 50},
		Owned:       gitlab.Ptr(true),
	}

	var discovered []vcs.DiscoveredProject

	for {
		projects, resp, err := client.Projects.ListUserProjects(userID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list projects for user %s: %w", username, err)
		}

		for _, proj := range projects {
			*checkedCount++
			log.Printf("[discovery] Checking repository [%d]: %s", *checkedCount, proj.PathWithNamespace)

			// Check if project has .doc-thor.project.yaml
			hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, proj)

			if hasDocThor {
				log.Printf("[discovery]   ✓ Found .doc-thor.project.yaml (slug: %s)", docThorConfig.Slug)
				discovered = append(discovered, vcs.DiscoveredProject{
					Name:          proj.Name,
					Path:          proj.PathWithNamespace,
					CloneURL:      proj.HTTPURLToRepo,
					DefaultBranch: proj.DefaultBranch,
					HasDocThor:    true,
					DocThorConfig: docThorConfig,
				})
			} else {
				log.Printf("[discovery]   ✗ No .doc-thor.project.yaml found")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return discovered, nil
}

// checkForDocThor checks if a project has .doc-thor.project.yaml and parses it.
func (p *GitLabProvider) checkForDocThor(ctx context.Context, client *gitlab.Client, proj *gitlab.Project) (bool, *vcs.DocThorConfig) {
	// Check for .doc-thor.project.yaml in root
	file, _, err := client.RepositoryFiles.GetFile(
		proj.ID,
		".doc-thor.project.yaml",
		&gitlab.GetFileOptions{Ref: gitlab.Ptr(proj.DefaultBranch)},
		gitlab.WithContext(ctx),
	)

	if err != nil {
		// File doesn't exist or other error
		log.Printf("[discovery]   File read error: %v", err)
		return false, nil
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		log.Printf("[discovery]   Base64 decode error: %v", err)
		return false, nil
	}

	log.Printf("[discovery]   File content:\n%s", string(content))

	// Parse YAML
	var config vcs.DocThorConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		// Invalid YAML - log but don't fail discovery
		log.Printf("[discovery]   YAML parse error: %v", err)
		return false, nil
	}

	log.Printf("[discovery]   Parsed config: slug=%q, name=%q, docker_image=%q, branch_mappings=%d",
		config.Slug, config.Name, config.DockerImage, len(config.BranchMappings))

	// Validate required fields
	if config.Slug == "" || config.Name == "" || config.DockerImage == "" {
		log.Printf("[discovery]   Validation failed: missing required fields")
		return false, nil
	}

	return true, &config
}

// GetRepositoryInfo fetches metadata about a repository.
func (p *GitLabProvider) GetRepositoryInfo(ctx context.Context, config vcs.IntegrationConfig, repoPath string) (*vcs.RepositoryInfo, error) {
	client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	proj, _, err := client.Projects.GetProject(repoPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", repoPath, err)
	}

	return &vcs.RepositoryInfo{
		FullPath:      proj.PathWithNamespace,
		DefaultBranch: proj.DefaultBranch,
		CloneURL:      proj.HTTPURLToRepo,
		Description:   proj.Description,
	}, nil
}

// RegisterWebhook creates a webhook on the VCS platform for the given project.
func (p *GitLabProvider) RegisterWebhook(ctx context.Context, config vcs.IntegrationConfig, repoPath string, events []vcs.EventType, callbackURL string) (string, error) {
	client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
	if err != nil {
		return "", fmt.Errorf("failed to create GitLab client: %w", err)
	}

	proj, _, err := client.Projects.GetProject(repoPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get project %s: %w", repoPath, err)
	}

	// Map vcs.EventType to GitLab event flags
	pushEvents := false
	tagEvents := false
	for _, e := range events {
		if e == vcs.EventPush {
			pushEvents = true
		}
		if e == vcs.EventTag {
			tagEvents = true
		}
	}

	hook, _, err := client.Projects.AddProjectHook(proj.ID, &gitlab.AddProjectHookOptions{
		URL:                   gitlab.Ptr(callbackURL),
		PushEvents:            gitlab.Ptr(pushEvents),
		TagPushEvents:         gitlab.Ptr(tagEvents),
		Token:                 gitlab.Ptr(config.WebhookSecret),
		EnableSSLVerification: gitlab.Ptr(true),
	}, gitlab.WithContext(ctx))

	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	return strconv.FormatInt(hook.ID, 10), nil
}

// UnregisterWebhook deletes a webhook by its provider-specific ID.
func (p *GitLabProvider) UnregisterWebhook(ctx context.Context, config vcs.IntegrationConfig, repoPath, webhookID string) error {
	client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	proj, _, err := client.Projects.GetProject(repoPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get project %s: %w", repoPath, err)
	}

	hookID, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %s", webhookID)
	}

	_, err = client.Projects.DeleteProjectHook(proj.ID, hookID, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

// Helper functions

func extractBranch(ref string) string {
	// refs/heads/main -> main
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ref
}

func extractTag(ref string) string {
	// refs/tags/v1.0.0 -> v1.0.0
	const prefix = "refs/tags/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ref
}
