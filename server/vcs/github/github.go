package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/romain325/doc-thor/server/vcs"
	"gopkg.in/yaml.v3"
)

// GitHubProvider implements the vcs.Provider interface for GitHub.
type GitHubProvider struct{}

// Name returns the provider identifier.
func (p *GitHubProvider) Name() string {
	return "github"
}

// ValidateWebhook verifies the webhook signature and parses the payload.
func (p *GitHubProvider) ValidateWebhook(r *http.Request, secret string) (*vcs.Event, error) {
	// Validate X-Hub-Signature-256 header
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return nil, fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	// Read payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Verify signature
	if !verifySignature(body, signature, secret) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	// Parse event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "push" && eventType != "create" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	event := &vcs.Event{}

	if eventType == "push" {
		var payload struct {
			Ref        string `json:"ref"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			HeadCommit *struct {
				ID      string `json:"id"`
				Message string `json:"message"`
				Author  struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"head_commit"`
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
		}

		event.Repository = payload.Repository.FullName

		// Check if it's a tag push (refs/tags/*) or branch push (refs/heads/*)
		if strings.HasPrefix(payload.Ref, "refs/tags/") {
			event.Type = vcs.EventTag
			event.Tag = extractTag(payload.Ref)
		} else {
			event.Type = vcs.EventPush
			event.Branch = extractBranch(payload.Ref)
		}

		if payload.HeadCommit != nil {
			event.Commit = payload.HeadCommit.ID
			event.CommitMessage = payload.HeadCommit.Message
			event.Author = payload.HeadCommit.Author.Name
		}
	} else if eventType == "create" {
		// Handle tag creation events
		var payload struct {
			RefType    string `json:"ref_type"`
			Ref        string `json:"ref"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Sender struct {
				Login string `json:"login"`
			} `json:"sender"`
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
		}

		if payload.RefType != "tag" {
			return nil, fmt.Errorf("unsupported create event ref_type: %s", payload.RefType)
		}

		event.Type = vcs.EventTag
		event.Repository = payload.Repository.FullName
		event.Tag = payload.Ref
		event.Author = payload.Sender.Login
	}

	return event, nil
}

// DiscoverProjects scans the given scope and returns projects with .doc-thor.project.yaml.
func (p *GitHubProvider) DiscoverProjects(ctx context.Context, config vcs.IntegrationConfig, scope string) ([]vcs.DiscoveredProject, error) {
	slog.Debug("starting GitHub project discovery", slog.String("scope", scope))

	client := github.NewClient(nil).WithAuthToken(config.AccessToken)
	if config.InstanceURL != "" && config.InstanceURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(config.InstanceURL, config.InstanceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure GitHub Enterprise URLs: %w", err)
		}
	}

	var discovered []vcs.DiscoveredProject
	checkedCount := 0

	// First, try to fetch it as a single repository (for --repo use case)
	// This works when scope is a full repo path like "owner/repo"
	singleRepo, singleErr := p.discoverSingleRepository(ctx, client, scope, &checkedCount)
	if singleErr == nil && len(singleRepo) > 0 {
		discovered = append(discovered, singleRepo...)
		slog.Debug("discovery complete", slog.Int("checked", checkedCount), slog.Int("found", len(discovered)))
		return discovered, nil
	}

	slog.Warn("single repository discovery failed, trying organization discovery", slog.Any("err", singleErr))

	// Try organization discovery
	orgRepos, orgErr := p.discoverOrganizationRepositories(ctx, client, scope, &checkedCount)
	if orgErr == nil {
		discovered = append(discovered, orgRepos...)
		slog.Debug("discovery complete", slog.Int("checked", checkedCount), slog.Int("found", len(discovered)))
		return discovered, nil
	}

	slog.Warn("organization discovery failed, trying user discovery", slog.Any("err", orgErr))

	// If organization discovery fails, try user repository discovery
	userRepos, userErr := p.discoverUserRepositories(ctx, client, scope, &checkedCount)
	if userErr != nil {
		return nil, fmt.Errorf("failed to discover projects in scope %s (tried repository, organization, and user): repository error: %v, organization error: %v, user error: %v", scope, singleErr, orgErr, userErr)
	}

	discovered = append(discovered, userRepos...)
	slog.Debug("discovery complete", slog.Int("checked", checkedCount), slog.Int("found", len(discovered)))
	return discovered, nil
}

// discoverSingleRepository tries to fetch a single repository by its full path.
func (p *GitHubProvider) discoverSingleRepository(ctx context.Context, client *github.Client, repoPath string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	*checkedCount++
	slog.Debug("checking single repository", slog.String("repo", repoPath))

	// Split owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository path format (expected owner/repo): %s", repoPath)
	}
	owner, repo := parts[0], parts[1]

	// Try to get the repository
	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s: %w", repoPath, err)
	}

	// Check if repository has .doc-thor.project.yaml
	hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, repository)

	if !hasDocThor {
		slog.Debug("no .doc-thor.project.yaml found", slog.String("repo", repoPath))
		return nil, fmt.Errorf("repository %s does not have .doc-thor.project.yaml", repoPath)
	}

	slog.Debug("found .doc-thor.project.yaml", slog.String("repo", repoPath), slog.String("slug", docThorConfig.Slug))
	return []vcs.DiscoveredProject{
		{
			Name:          repository.GetName(),
			Path:          repository.GetFullName(),
			CloneURL:      repository.GetCloneURL(),
			DefaultBranch: repository.GetDefaultBranch(),
			HasDocThor:    true,
			DocThorConfig: docThorConfig,
		},
	}, nil
}

// discoverOrganizationRepositories lists repositories in a GitHub organization.
func (p *GitHubProvider) discoverOrganizationRepositories(ctx context.Context, client *github.Client, orgName string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	var discovered []vcs.DiscoveredProject

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, orgName, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories in organization %s: %w", orgName, err)
		}

		for _, repo := range repos {
			*checkedCount++
			slog.Debug("checking repository", slog.Int("index", *checkedCount), slog.String("repo", repo.GetFullName()))

			// Check if repository has .doc-thor.project.yaml
			hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, repo)

			if hasDocThor {
				slog.Debug("found .doc-thor.project.yaml", slog.String("repo", repo.GetFullName()), slog.String("slug", docThorConfig.Slug))
				discovered = append(discovered, vcs.DiscoveredProject{
					Name:          repo.GetName(),
					Path:          repo.GetFullName(),
					CloneURL:      repo.GetCloneURL(),
					DefaultBranch: repo.GetDefaultBranch(),
					HasDocThor:    true,
					DocThorConfig: docThorConfig,
				})
			} else {
				slog.Debug("no .doc-thor.project.yaml found", slog.String("repo", repo.GetFullName()))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return discovered, nil
}

// discoverUserRepositories lists repositories owned by a GitHub user.
func (p *GitHubProvider) discoverUserRepositories(ctx context.Context, client *github.Client, username string, checkedCount *int) ([]vcs.DiscoveredProject, error) {
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 50},
		Type:        "owner",
	}

	var discovered []vcs.DiscoveredProject

	for {
		repos, resp, err := client.Repositories.ListByUser(ctx, username, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories for user %s: %w", username, err)
		}

		for _, repo := range repos {
			*checkedCount++
			slog.Debug("checking repository", slog.Int("index", *checkedCount), slog.String("repo", repo.GetFullName()))

			// Check if repository has .doc-thor.project.yaml
			hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, repo)

			if hasDocThor {
				slog.Debug("found .doc-thor.project.yaml", slog.String("repo", repo.GetFullName()), slog.String("slug", docThorConfig.Slug))
				discovered = append(discovered, vcs.DiscoveredProject{
					Name:          repo.GetName(),
					Path:          repo.GetFullName(),
					CloneURL:      repo.GetCloneURL(),
					DefaultBranch: repo.GetDefaultBranch(),
					HasDocThor:    true,
					DocThorConfig: docThorConfig,
				})
			} else {
				slog.Debug("no .doc-thor.project.yaml found", slog.String("repo", repo.GetFullName()))
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return discovered, nil
}

// checkForDocThor checks if a repository has .doc-thor.project.yaml and parses it.
func (p *GitHubProvider) checkForDocThor(ctx context.Context, client *github.Client, repo *github.Repository) (bool, *vcs.DocThorConfig) {
	// Split owner/repo
	parts := strings.Split(repo.GetFullName(), "/")
	if len(parts) != 2 {
		slog.Debug("invalid repository path", slog.String("repo", repo.GetFullName()))
		return false, nil
	}
	owner, repoName := parts[0], parts[1]

	// Check for .doc-thor.project.yaml in root
	fileContent, _, _, err := client.Repositories.GetContents(
		ctx,
		owner,
		repoName,
		".doc-thor.project.yaml",
		&github.RepositoryContentGetOptions{Ref: repo.GetDefaultBranch()},
	)

	if err != nil {
		// File doesn't exist or other error
		slog.Debug("file read error", slog.String("repo", repo.GetFullName()), slog.Any("err", err))
		return false, nil
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		slog.Debug("content decode error", slog.String("repo", repo.GetFullName()), slog.Any("err", err))
		return false, nil
	}

	slog.Debug("file content fetched", slog.String("repo", repo.GetFullName()), slog.String("content", content))

	// Parse YAML
	var config vcs.DocThorConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		// Invalid YAML - log but don't fail discovery
		slog.Debug("YAML parse error", slog.String("repo", repo.GetFullName()), slog.Any("err", err))
		return false, nil
	}

	slog.Debug("parsed config", slog.String("repo", repo.GetFullName()),
		slog.String("slug", config.Slug), slog.String("name", config.Name),
		slog.String("docker_image", config.DockerImage), slog.Int("branch_mappings", len(config.BranchMappings)))

	// Validate required fields
	if config.Slug == "" || config.Name == "" || config.DockerImage == "" {
		slog.Debug("validation failed: missing required fields", slog.String("repo", repo.GetFullName()))
		return false, nil
	}

	return true, &config
}

// GetRepositoryInfo fetches metadata about a repository.
func (p *GitHubProvider) GetRepositoryInfo(ctx context.Context, config vcs.IntegrationConfig, repoPath string) (*vcs.RepositoryInfo, error) {
	client := github.NewClient(nil).WithAuthToken(config.AccessToken)
	if config.InstanceURL != "" && config.InstanceURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(config.InstanceURL, config.InstanceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure GitHub Enterprise URLs: %w", err)
		}
	}

	// Split owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository path format (expected owner/repo): %s", repoPath)
	}
	owner, repo := parts[0], parts[1]

	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s: %w", repoPath, err)
	}

	return &vcs.RepositoryInfo{
		FullPath:      repository.GetFullName(),
		DefaultBranch: repository.GetDefaultBranch(),
		CloneURL:      repository.GetCloneURL(),
		Description:   repository.GetDescription(),
	}, nil
}

// RegisterWebhook creates a webhook on the VCS platform for the given project.
func (p *GitHubProvider) RegisterWebhook(ctx context.Context, config vcs.IntegrationConfig, repoPath string, events []vcs.EventType, callbackURL string) (string, error) {
	client := github.NewClient(nil).WithAuthToken(config.AccessToken)
	if config.InstanceURL != "" && config.InstanceURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(config.InstanceURL, config.InstanceURL)
		if err != nil {
			return "", fmt.Errorf("failed to configure GitHub Enterprise URLs: %w", err)
		}
	}

	// Split owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository path format (expected owner/repo): %s", repoPath)
	}
	owner, repo := parts[0], parts[1]

	// Map vcs.EventType to GitHub event names
	eventNames := []string{}
	for _, e := range events {
		if e == vcs.EventPush {
			eventNames = append(eventNames, "push")
		}
		if e == vcs.EventTag {
			eventNames = append(eventNames, "create")
		}
	}

	hookConfig := map[string]any{
		"url":          callbackURL,
		"content_type": "json",
		"secret":       config.WebhookSecret,
	}

	hook := &github.Hook{
		Events: eventNames,
		Config: hookConfig,
		Active: github.Bool(true),
	}

	createdHook, _, err := client.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	return strconv.FormatInt(createdHook.GetID(), 10), nil
}

// UnregisterWebhook deletes a webhook by its provider-specific ID.
func (p *GitHubProvider) UnregisterWebhook(ctx context.Context, config vcs.IntegrationConfig, repoPath, webhookID string) error {
	client := github.NewClient(nil).WithAuthToken(config.AccessToken)
	if config.InstanceURL != "" && config.InstanceURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(config.InstanceURL, config.InstanceURL)
		if err != nil {
			return fmt.Errorf("failed to configure GitHub Enterprise URLs: %w", err)
		}
	}

	// Split owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository path format (expected owner/repo): %s", repoPath)
	}
	owner, repo := parts[0], parts[1]

	hookID, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %s", webhookID)
	}

	_, err = client.Repositories.DeleteHook(ctx, owner, repo, hookID)
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

func verifySignature(payload []byte, signature, secret string) bool {
	// GitHub signature format: sha256=<hex>
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	signatureHex := signature[len(prefix):]

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedHex := hex.EncodeToString(expectedMAC)

	// Constant-time comparison
	return hmac.Equal([]byte(signatureHex), []byte(expectedHex))
}
