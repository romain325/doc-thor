package discovery

import (
	"context"
	"net/http"
	"time"
)

// BackendStatus is the health-check result for a single backend service.
type BackendStatus struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
}

// CheckBuilder pings a builder's /health endpoint.
func CheckBuilder(url string) BackendStatus {
	s := BackendStatus{Name: "builder", URL: url, LastCheck: time.Now()}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)
	if err != nil {
		return s
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return s
	}
	resp.Body.Close()
	s.Healthy = resp.StatusCode == http.StatusOK
	return s
}

// CheckStorage pings the S3-compatible storage endpoint.  A 403 on the
// root path without credentials still proves the service is alive.
func CheckStorage(endpoint string, useSSL bool) BackendStatus {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	url := scheme + "://" + endpoint
	s := BackendStatus{Name: "storage", URL: url, LastCheck: time.Now()}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return s
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return s
	}
	resp.Body.Close()
	s.Healthy = resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden
	return s
}
