package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/spf13/cobra"
)

var (
	importIntegration   string
	importRepoPath      string
	importScope         string
	importRegisterHook  bool
	importCallbackURL   string
	importAutoPublish   bool
	importBranchMapping []string
)

var projectImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import project(s) from VCS integration",
	Long: `Import one or more projects by discovering their .doc-thor.project.yaml configuration.
The webhook callback URL is auto-generated from your server config if not provided.

Examples:
  # Import all projects from a scope
  doc-thor project import --integration company-gitlab --scope myteam/docs \
    --register-webhook --auto-publish

  # Import a single repository
  doc-thor project import --integration company-gitlab --repo myteam/docs/api-docs \
    --register-webhook

  # Import with custom callback URL
  doc-thor project import --integration company-gitlab --repo myteam/docs/api-docs \
    --register-webhook --callback-url https://docs.example.com/api/v1/webhooks/gitlab/api-docs

  # Import with custom branch mappings
  doc-thor project import --integration company-gitlab --repo myteam/docs/user-guide \
    --register-webhook --auto-publish \
    --branch-mapping main:latest:true --branch-mapping "v*:\${tag}:true"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine the discovery scope
		scope := importScope
		if scope == "" && importRepoPath != "" {
			scope = importRepoPath
		}
		if scope == "" {
			return fmt.Errorf("either --scope or --repo must be provided")
		}

		// Discover projects in the scope
		discoveryReq := client.DiscoveryRequest{
			Scope: scope,
		}

		result, err := c.DiscoverProjects(importIntegration, discoveryReq)
		if err != nil {
			return err
		}

		if result.Count == 0 {
			return fmt.Errorf("no projects with .doc-thor.project.yaml found in scope: %s", scope)
		}

		// Debug: show what we received
		fmt.Printf("DEBUG: Received %d projects from discovery\n", result.Count)
		for i, proj := range result.Projects {
			fmt.Printf("DEBUG: Project %d: path=%s, HasDocThor=%v, DocThorConfig=%v\n",
				i, proj.Path, proj.HasDocThor, proj.DocThorConfig != nil)
			if proj.DocThorConfig != nil {
				fmt.Printf("DEBUG:   Config: slug=%s, name=%s, docker_image=%s\n",
					proj.DocThorConfig.Slug, proj.DocThorConfig.Name, proj.DocThorConfig.DockerImage)
			}
		}

		// Filter projects to only those with valid doc-thor config
		var validProjects []client.DiscoveredProject
		skippedCount := 0
		for _, proj := range result.Projects {
			if proj.HasDocThor && proj.DocThorConfig != nil {
				validProjects = append(validProjects, proj)
			} else {
				skippedCount++
				fmt.Printf("DEBUG: Skipping project %s: HasDocThor=%v, DocThorConfig!=nil=%v\n",
					proj.Path, proj.HasDocThor, proj.DocThorConfig != nil)
			}
		}

		if skippedCount > 0 {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
			fmt.Printf("%s Skipped %d project(s) without valid .doc-thor.project.yaml\n\n",
				warningStyle.Render("⚠"), skippedCount)
		}

		if len(validProjects) == 0 {
			return fmt.Errorf("no projects with valid .doc-thor.project.yaml found in scope: %s", scope)
		}

		// Determine which projects to import
		projectsToImport := validProjects
		if importRepoPath != "" && importScope == "" {
			// If --repo was specified (not --scope), import only that one
			found := false
			for _, proj := range validProjects {
				if proj.Path == importRepoPath {
					projectsToImport = []client.DiscoveredProject{proj}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("repository %s not found in discovered projects or missing .doc-thor.project.yaml", importRepoPath)
			}
		}

		// Parse branch mappings if provided
		var branchMappings []client.BranchMapping
		for _, mapping := range importBranchMapping {
			// Format: branch:version_tag:auto_publish
			var bm client.BranchMapping
			fmt.Sscanf(mapping, "%s:%s:%t", &bm.Branch, &bm.VersionTag, &bm.AutoPublish)
			branchMappings = append(branchMappings, bm)
		}

		// Get integration details once (for webhook URL generation)
		var integration client.VCSIntegration
		var serverURL string
		if importRegisterHook && importCallbackURL == "" {
			var err error
			integration, err = c.GetVCSIntegration(importIntegration)
			if err != nil {
				return fmt.Errorf("failed to get integration details: %w", err)
			}

			serverURL = c.BaseURL()
			if len(serverURL) > 7 && serverURL[len(serverURL)-7:] == "/api/v1" {
				serverURL = serverURL[:len(serverURL)-7]
			}
		}

		// Import each project
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

		importedCount := 0
		failedCount := 0

		for _, discovered := range projectsToImport {
			fmt.Printf("\nImporting %s (%s)...\n", labelStyle.Render(discovered.Path), discovered.DocThorConfig.Slug)

			// Auto-generate callback URL if needed
			callbackURL := importCallbackURL
			if importRegisterHook && callbackURL == "" {
				callbackURL = fmt.Sprintf("%s/api/v1/webhooks/%s/%s",
					serverURL, integration.Provider, discovered.DocThorConfig.Slug)
				fmt.Printf("  Webhook URL: %s\n", callbackURL)
			}

			importReq := client.ImportProjectRequest{
				IntegrationName:   importIntegration,
				DiscoveredProject: discovered,
				BranchMappings:    branchMappings,
				AutoPublish:       importAutoPublish,
				RegisterWebhook:   importRegisterHook,
				CallbackURL:       callbackURL,
			}

			project, err := c.ImportProject(importReq)
			if err != nil {
				fmt.Println(errorStyle.Render(fmt.Sprintf("  ✗ Failed: %v", err)))
				failedCount++
				continue
			}

			fmt.Println(successStyle.Render("  ✓ Imported successfully"))
			fmt.Printf("    Slug: %s\n", project.Slug)
			fmt.Printf("    Name: %s\n", project.Name)
			fmt.Printf("    Image: %s\n", project.DockerImage)
			if importRegisterHook {
				fmt.Printf("    Webhook: %s\n", successStyle.Render("registered"))
			}
			importedCount++
		}

		// Summary
		fmt.Println()
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Import complete: %d succeeded, %d failed", importedCount, failedCount)))

		if failedCount > 0 {
			return fmt.Errorf("%d project(s) failed to import", failedCount)
		}

		return nil
	},
}

func init() {
	projectImportCmd.Flags().StringVar(&importIntegration, "integration", "", "VCS integration name (required)")
	projectImportCmd.Flags().StringVar(&importScope, "scope", "", "VCS scope to discover and import all projects (group/namespace/user)")
	projectImportCmd.Flags().StringVar(&importRepoPath, "repo", "", "Single repository path to import (mutually exclusive with --scope)")
	projectImportCmd.Flags().BoolVar(&importRegisterHook, "register-webhook", false, "Register webhook for automatic builds")
	projectImportCmd.Flags().StringVar(&importCallbackURL, "callback-url", "", "Webhook callback URL (optional, auto-generated if not provided)")
	projectImportCmd.Flags().BoolVar(&importAutoPublish, "auto-publish", false, "Override auto-publish for all branch mappings")
	projectImportCmd.Flags().StringArrayVar(&importBranchMapping, "branch-mapping", nil, "Branch mapping: branch:version_tag:auto_publish")

	projectImportCmd.MarkFlagRequired("integration")
	projectImportCmd.MarkFlagsOneRequired("scope", "repo")
	projectImportCmd.MarkFlagsMutuallyExclusive("scope", "repo")

	projectCmd.AddCommand(projectImportCmd)
}
