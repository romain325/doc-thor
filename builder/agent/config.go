package main

import (
	"strconv"
	"time"
)

// Config is populated entirely from environment variables.
type Config struct {
	ServerURL        string
	ServerToken      string
	StorageBucket    string
	StorageEndpoint  string
	StorageRegion    string
	StorageAccessKey string
	StorageSecretKey string
	PollInterval     time.Duration
	ContainerTimeout time.Duration
	// WorkspaceDir is the base directory for per-job temp dirs (repo clone +
	// build output).  It must be bind-mounted from the host at the exact same
	// path so that the paths the builder passes to the Docker API are valid on
	// the host daemon.
	WorkspaceDir string
}

func loadConfig() Config {
	pollSec, _ := strconv.Atoi(getEnv("POLL_INTERVAL", "5"))
	timeoutSec, _ := strconv.Atoi(getEnv("CONTAINER_TIMEOUT", "300"))

	return Config{
		ServerURL:        getEnv("SERVER_URL", "http://localhost:8080"),
		ServerToken:      mustEnv("BUILDER_TOKEN"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "doc-thor-docs"),
		StorageEndpoint:  mustEnv("STORAGE_ENDPOINT"),
		StorageRegion:    getEnv("STORAGE_REGION", "us-east-1"),
		StorageAccessKey: mustEnv("STORAGE_ACCESS_KEY"),
		StorageSecretKey: mustEnv("STORAGE_SECRET_KEY"),
		PollInterval:     time.Duration(pollSec) * time.Second,
		ContainerTimeout: time.Duration(timeoutSec) * time.Second,
		WorkspaceDir:     mustEnv("WORKSPACE_DIR"),
	}
}
