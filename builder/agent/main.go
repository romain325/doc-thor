package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Job is the payload the server sends when a build is pending.
type Job struct {
	ID          string `json:"id"`
	ProjectSlug string `json:"project_slug"`
	Version     string `json:"version"`
	SourceURL   string `json:"source_url"`
	Ref         string `json:"ref"`
	DockerImage string `json:"docker_image"`
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func pollForJob(ctx context.Context, cfg Config) (*Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ServerURL+"/api/v1/builds/pending", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.ServerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll: status %d", resp.StatusCode)
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}
	return &job, nil
}

func main() {
	cfg := loadConfig()
	log.Printf("builder started, polling %s every %s", cfg.ServerURL, cfg.PollInterval)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		<-ticker.C

		job, err := pollForJob(context.Background(), cfg)
		if err != nil {
			log.Printf("poll error: %v", err)
			continue
		}
		if job == nil {
			continue
		}

		log.Printf("picked up job %s for project %s", job.ID, job.ProjectSlug)
		go func(j Job) {
			if err := runPipeline(cfg, j); err != nil {
				log.Printf("pipeline error for job %s: %v", j.ID, err)
			}
		}(*job)
	}
}
