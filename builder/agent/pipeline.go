package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/romain325/doc-thor/builder/agent/stages"
)

func runPipeline(cfg Config, job Job) error {
	start := time.Now()

	repoDir, err := os.MkdirTemp(cfg.WorkspaceDir, "builder-repo-"+job.ID)
	if err != nil {
		reportResult(cfg, job.ID, "failed", time.Since(start), fmt.Sprintf("create repo dir: %v", err), "")
		return err
	}
	defer os.RemoveAll(repoDir)

	outputDir, err := os.MkdirTemp(cfg.WorkspaceDir, "builder-output-"+job.ID)
	if err != nil {
		reportResult(cfg, job.ID, "failed", time.Since(start), fmt.Sprintf("create output dir: %v", err), "")
		return err
	}
	defer os.RemoveAll(outputDir)

	s3Cfg := stages.S3Config{
		Endpoint:  cfg.StorageEndpoint,
		Region:    cfg.StorageRegion,
		AccessKey: cfg.StorageAccessKey,
		SecretKey: cfg.StorageSecretKey,
		Bucket:    cfg.StorageBucket,
	}

	// containerLogs holds stdout+stderr captured during the run stage.
	// It is populated before any later stage executes so that even a failure
	// in collect/upload still includes the build output in the report.
	var containerLogs string

	type stage struct {
		name string
		fn   func() error
	}

	pipeline := []stage{
		{"pull", func() error {
			resolved, err := stages.Pull(job.SourceURL, job.Ref, repoDir)
			if err != nil {
				return err
			}
			job.Ref = resolved
			return nil
		}},
		{"run", func() error {
			var err error
			containerLogs, err = stages.Run(job.DockerImage, repoDir, outputDir, cfg.ContainerTimeout)
			return err
		}},
		{"collect", func() error { return stages.Collect(outputDir) }},
		{"upload", func() error { return stages.Upload(s3Cfg, job.ProjectSlug, job.Version, outputDir) }},
	}

	for _, s := range pipeline {
		log.Printf("[%s] job %s: starting", s.name, job.ID)
		if err := s.fn(); err != nil {
			errMsg := fmt.Sprintf("%s: %v", s.name, err)
			reportResult(cfg, job.ID, "failed", time.Since(start), errMsg, containerLogs)
			return fmt.Errorf("%s: %w", s.name, err)
		}
		log.Printf("[%s] job %s: done", s.name, job.ID)
	}

	reportResult(cfg, job.ID, "success", time.Since(start), "", containerLogs)
	log.Printf("job %s completed successfully in %s", job.ID, time.Since(start))
	return nil
}
