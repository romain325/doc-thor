package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type buildResult struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
	Logs     string `json:"logs,omitempty"`
}

func reportResult(cfg Config, jobID, status string, duration time.Duration, errMsg, logs string) {
	body, err := json.Marshal(buildResult{
		JobID:    jobID,
		Status:   status,
		Duration: duration.String(),
		Error:    errMsg,
		Logs:     logs,
	})
	if err != nil {
		log.Printf("report marshal: %v", err)
		return
	}

	url := cfg.ServerURL + "/api/v1/builds/" + jobID + "/result"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("report request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.ServerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("report send: %v", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("report: server returned %d", resp.StatusCode)
	}
}
