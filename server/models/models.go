package models

import "time"

// Base provides common fields for all models. Omits DeletedAt
// intentionally — deletes in this project are hard.
type Base struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Project is a registered documentation source.
type Project struct {
	Base
	Slug        string      `gorm:"uniqueIndex;not null" json:"slug"`
	Name        string      `gorm:"not null" json:"name"`
	SourceURL   string      `gorm:"column:source_url;not null" json:"source_url"`
	DockerImage string      `gorm:"column:docker_image;not null" json:"docker_image"`
	BuildConfig BuildConfig `gorm:"serializer:json" json:"build_config"`
}

// BuildConfig is stored as a single JSON column on Project.
type BuildConfig struct {
	Plugins         []string       `json:"plugins,omitempty"`
	PreBuildHooks   []string       `json:"pre_build_hooks,omitempty"`
	MkdocsOverrides map[string]any `json:"mkdocs_overrides,omitempty"`
}

// Build tracks a single doc-build job. Status lifecycle: pending → running → success | failed.
type Build struct {
	Base
	ProjectID  uint       `gorm:"not null;index" json:"project_id"`
	Ref        string     `json:"ref"`
	Tag        string     `json:"tag"`
	Status     string     `gorm:"default:pending" json:"status"`
	Logs       string     `gorm:"type:text" json:"logs,omitempty"`
	Error      string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// Version is a published build output addressable by tag.
type Version struct {
	Base
	ProjectID uint   `gorm:"not null;uniqueIndex:idx_project_version" json:"project_id"`
	BuildID   uint   `gorm:"not null" json:"build_id"`
	Tag       string `gorm:"not null;uniqueIndex:idx_project_version" json:"version"`
	Published bool   `gorm:"default:false" json:"published"`
	IsLatest  bool   `gorm:"default:false;column:is_latest" json:"is_latest"`
}

// User is a local account.
type User struct {
	Base
	Username     string `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"not null" json:"-"`
	IsSuperuser  bool   `gorm:"default:false" json:"is_superuser"`
}

// Token covers both session tokens and API keys.
// The raw token is never stored; only its SHA-256 hash.
type Token struct {
	Base
	UserID    uint       `gorm:"not null;index"`
	TokenHash string     `gorm:"uniqueIndex;not null"`
	Type      string     `gorm:"not null"` // "session" | "apikey"
	Label     string     `json:"label,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
