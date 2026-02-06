# OpenAPI & CLI Update Summary

Summary of updates to OpenAPI specification and CLI for VCS integration support.

---

## OpenAPI Specification Updates

### New Schemas Added

**`server/api/openapi.yaml`**

1. **VCSIntegration** - VCS integration configuration object
   - `id`, `name`, `provider`, `instance_url`, `enabled`, timestamps

2. **VCSIntegrationCreate** - Create VCS integration request
   - `name`, `provider`, `instance_url`, `access_token`, `webhook_secret`, `enabled`

3. **VCSIntegrationUpdate** - Update VCS integration request
   - All fields optional for partial updates

4. **DiscoveryRequest** - Project discovery request
   - `scope` - VCS scope to scan (group/org path)

5. **DiscoveredProject** - Discovered project with config
   - `name`, `path`, `clone_url`, `default_branch`, `has_doc_thor`, `doc_thor_config`

6. **DocThorConfig** - Parsed .doc-thor.project.yaml
   - `slug`, `name`, `docker_image`, `branch_mappings`

7. **BranchMapping** - Branch to version mapping
   - `branch` (pattern), `version_tag` (with template vars), `auto_publish`

8. **ImportProjectRequest** - Import discovered project
   - `integration_name`, `discovered_project`, optional overrides

### New Endpoints Added

#### VCS Integrations (Authenticated)
```
POST   /integrations                      # Create VCS integration
GET    /integrations                      # List all integrations
GET    /integrations/{name}               # Get integration details
PUT    /integrations/{name}               # Update integration
DELETE /integrations/{name}               # Delete integration
POST   /integrations/{name}/test          # Test connection
```

#### Project Discovery (Authenticated)
```
POST   /integrations/{name}/discover      # Discover projects in scope
POST   /projects/import                   # Import discovered project
```

#### Webhooks (Public - No Auth)
```
POST   /webhooks/{provider}/{slug}        # Webhook receiver
```

---

## CLI Updates

### New Client Methods

**`cli/internal/client/client.go`**

Added complete client support for VCS integration:

```go
// VCS Integration
ListVCSIntegrations() ([]VCSIntegration, error)
CreateVCSIntegration(req VCSIntegrationCreate) (VCSIntegration, error)
GetVCSIntegration(name string) (VCSIntegration, error)
UpdateVCSIntegration(name string, req VCSIntegrationUpdate) (VCSIntegration, error)
DeleteVCSIntegration(name string) error
TestVCSIntegration(name string) (TestResult, error)

// Discovery
DiscoverProjects(integrationName string, req DiscoveryRequest) (DiscoveryResponse, error)
ImportProject(req ImportProjectRequest) (Project, error)
```

### New CLI Commands

#### Integration Management

**`doc-thor integration list`**
- Lists all VCS integrations
- Shows name, provider, URL, enabled status
- Styled output with badges

**`doc-thor integration create`**
```bash
doc-thor integration create \
  --name company-gitlab \
  --provider gitlab \
  --url https://gitlab.company.com \
  --token glpat-xxx \
  --webhook-secret my-secret
```

**`doc-thor integration get [name]`**
- Shows detailed integration information
- Provider, URL, status, timestamps

**`doc-thor integration delete [name]`**
- Interactive confirmation prompt (huh)
- Deletes VCS integration

**`doc-thor integration test [name]`**
- Tests VCS connection
- Shows success/failure with details

#### Project Discovery

**`doc-thor discover [integration-name] [scope]`**
```bash
doc-thor discover company-gitlab myteam/docs
```
- Scans VCS scope for projects with `.doc-thor.project.yaml`
- Shows discovered projects with config details
- Lists: path, name, slug, docker image, branch mappings

**`doc-thor project import`**
```bash
# Basic import
doc-thor project import \
  --integration company-gitlab \
  --repo myteam/docs/api-docs

# With webhook registration
doc-thor project import \
  --integration company-gitlab \
  --repo myteam/docs/api-docs \
  --register-webhook \
  --callback-url https://docs.example.com/api/v1/webhooks/gitlab/api-docs

# With custom branch mappings
doc-thor project import \
  --integration company-gitlab \
  --repo myteam/docs/user-guide \
  --branch-mapping main:latest:true \
  --branch-mapping "v*:\${tag}:true" \
  --auto-publish
```

### Command Files Created

```
cli/cmd/
├── integration.go                # Integration group command
├── integration_list.go           # List integrations
├── integration_create.go         # Create integration
├── integration_get.go            # Get integration details
├── integration_delete.go         # Delete integration
├── integration_test.go           # Test connection
├── discover.go                   # Discover projects
└── project_import.go             # Import project
```

---

## CLI Command Reference

### Complete Integration Workflow

```bash
# 1. Create VCS integration
doc-thor integration create \
  --name company-gitlab \
  --provider gitlab \
  --url https://gitlab.company.com \
  --token glpat-xxx \
  --webhook-secret my-secret-123

# 2. Test connection
doc-thor integration test company-gitlab

# 3. Discover projects
doc-thor discover company-gitlab myteam/docs

# 4. Import project with webhook
doc-thor project import \
  --integration company-gitlab \
  --repo myteam/docs/api-docs \
  --register-webhook \
  --callback-url https://docs.example.com/api/v1/webhooks/gitlab/api-docs

# 5. List integrations
doc-thor integration list

# 6. Get integration details
doc-thor integration get company-gitlab

# 7. Delete integration
doc-thor integration delete company-gitlab
```

---

## Updated Endpoint Mapping

| CLI Command                    | Method | Path                                  |
|--------------------------------|--------|---------------------------------------|
| `integration list`             | GET    | `/integrations`                       |
| `integration create`           | POST   | `/integrations`                       |
| `integration get`              | GET    | `/integrations/{name}`                |
| `integration delete`           | DELETE | `/integrations/{name}`                |
| `integration test`             | POST   | `/integrations/{name}/test`           |
| `discover`                     | POST   | `/integrations/{name}/discover`       |
| `project import`               | POST   | `/projects/import`                    |

---

## CLI Output Examples

### List Integrations
```
VCS INTEGRATIONS

company-gitlab  gitlab  https://gitlab.company.com  ✓ enabled
github-cloud    github  https://github.com          ✓ enabled
local-gitea     gitea   https://git.local.com       ✗ disabled
```

### Discover Projects
```
PROJECT DISCOVERY

✓ Found 3 projects

1. myteam/docs/api-docs
   Name: API Documentation
   Slug: api-docs
   Image: doc-thor/mkdocs-material:latest
   Branches: 2 mappings

2. myteam/docs/user-guide
   Name: User Guide
   Slug: user-guide
   Image: doc-thor/mkdocs:latest
   Branches: 3 mappings

3. myteam/platform/admin-docs
   Name: Admin Documentation
   Slug: admin-docs
   Image: custom/sphinx:latest
   Branches: 1 mappings

Use 'doc-thor import' to import these projects.
```

### Import Project
```
✓ Project imported
  Slug: api-docs
  Name: API Documentation
  Image: doc-thor/mkdocs-material:latest

✓ Webhook registered
  Builds will be triggered automatically on push
```

### Test Integration
```
✓ Connection successful
  Successfully connected to GitLab instance
```

---

## JSON Output

All commands support `--json` flag for programmatic usage:

```bash
# List integrations as JSON
doc-thor integration list --json

# Discover projects as JSON
doc-thor discover company-gitlab myteam/docs --json

# Import project with JSON output
doc-thor project import --integration company-gitlab --repo myteam/docs/api-docs --json
```

---

## Build Status

✅ **Server OpenAPI spec updated** - All VCS integration endpoints documented
✅ **CLI client updated** - All HTTP methods implemented
✅ **CLI commands created** - 8 new commands added
✅ **CLI compiles successfully** - No errors

---

## Files Modified/Created

### OpenAPI
- **Modified**: `server/api/openapi.yaml` - Added 8 schemas, 11 endpoints

### CLI Client
- **Modified**: `cli/internal/client/client.go` - Added VCS integration methods

### CLI Commands
- **Created**: `cli/cmd/integration.go` - Integration group
- **Created**: `cli/cmd/integration_list.go` - List command
- **Created**: `cli/cmd/integration_create.go` - Create command
- **Created**: `cli/cmd/integration_get.go` - Get command
- **Created**: `cli/cmd/integration_delete.go` - Delete command
- **Created**: `cli/cmd/integration_test.go` - Test command
- **Created**: `cli/cmd/discover.go` - Discover command
- **Created**: `cli/cmd/project_import.go` - Import command

---

## Next Steps

1. ✅ OpenAPI specification complete
2. ✅ CLI implementation complete
3. ⏭️ Update CLI AGENTS.md with new commands
4. ⏭️ Test end-to-end workflow
5. ⏭️ Create user documentation

The CLI now provides full support for managing VCS integrations, discovering projects, and importing them with automatic webhook registration!
