package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/romain325/doc-thor/server/models"
	"gorm.io/gorm"
)

// SyncNginxConfig rewrites the server-block file for a project based on
// its currently-published versions.  If nothing is published the file is removed.
// Storage path contract: <slug>/<version>/<file-path> (see CLAUDE.md).
func SyncNginxConfig(db *gorm.DB, project *models.Project, nginxDir, storageEndpoint string) error {
	versions, err := ListVersions(db, project.ID)
	if err != nil {
		return err
	}

	var published []models.Version
	for _, v := range versions {
		if v.Published {
			published = append(published, v)
		}
	}

	configPath := filepath.Join(nginxDir, project.Slug+".conf")
	if len(published) == 0 {
		return os.Remove(configPath)
	}

	var blocks []string
	for _, v := range published {
		// versioned subdomain: <slug>-<tag>.docs.<domain>
		blocks = append(blocks, renderBlock(project.Slug, v.Tag, v.Tag, storageEndpoint))
		// bare subdomain served by latest
		if v.IsLatest {
			blocks = append(blocks, renderBlock(project.Slug, "", v.Tag, storageEndpoint))
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(blocks, "\n\n")+"\n"), 0644)
}

func renderBlock(slug, versionSuffix, storageVersion, storageEndpoint string) string {
	subdomain := slug
	if versionSuffix != "" {
		subdomain = slug + "-" + versionSuffix
	}
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s.docs.localhost;

    set $doc_project "%s";
    set $doc_version "%s";

    location / {
        proxy_pass %s/%s/%s/;
        proxy_set_header Host $host;
    }
}`, subdomain, slug, storageVersion, strings.TrimRight(storageEndpoint, "/"), slug, storageVersion)
}
