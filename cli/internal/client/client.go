package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is the single HTTP entry-point for every server API call.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL += "/api/v1"
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{},
	}
}

// APIError is returned whenever the server responds with HTTP >= 400.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Error (HTTP %d): %s", e.StatusCode, e.Message)
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil || errResp.Error == "" {
			return nil, &APIError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}

	return resp, nil
}

// decode performs a request and JSON-decodes the response body into v.
// 204 No Content is a successful no-op; v is left untouched.
func (c *Client) decode(method, path string, body, v any) error {
	resp, err := c.do(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// ---------------------------------------------------------------------------
// Health & Backends
// ---------------------------------------------------------------------------

func (c *Client) Health() (map[string]string, error) {
	var v map[string]string
	err := c.decode("GET", "/health", nil, &v)
	return v, err
}

type BackendStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Healthy   bool   `json:"healthy"`
	LastCheck string `json:"last_check"`
}

func (c *Client) Backends() ([]BackendStatus, error) {
	var v []BackendStatus
	err := c.decode("GET", "/backends", nil, &v)
	return v, err
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (c *Client) Login(req LoginRequest) (LoginResponse, error) {
	var v LoginResponse
	err := c.decode("POST", "/auth/login", req, &v)
	return v, err
}

type APIKeyCreateRequest struct {
	Label string `json:"label,omitempty"`
}

type APIKeyResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func (c *Client) CreateAPIKey(req APIKeyCreateRequest) (APIKeyResponse, error) {
	var v APIKeyResponse
	err := c.decode("POST", "/auth/apikey", req, &v)
	return v, err
}

type User struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (c *Client) Me() (User, error) {
	var v User
	err := c.decode("GET", "/auth/me", nil, &v)
	return v, err
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

type Project struct {
	ID          uint   `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	SourceURL   string `json:"source_url"`
	DockerImage string `json:"docker_image"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProjectCreate struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	SourceURL   string `json:"source_url"`
	DockerImage string `json:"docker_image"`
}

type ProjectUpdate struct {
	Name        string `json:"name,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	DockerImage string `json:"docker_image,omitempty"`
}

func (c *Client) ListProjects() ([]Project, error) {
	var v []Project
	err := c.decode("GET", "/projects", nil, &v)
	return v, err
}

func (c *Client) CreateProject(req ProjectCreate) (Project, error) {
	var v Project
	err := c.decode("POST", "/projects", req, &v)
	return v, err
}

func (c *Client) GetProject(slug string) (Project, error) {
	var v Project
	err := c.decode("GET", "/projects/"+slug, nil, &v)
	return v, err
}

func (c *Client) UpdateProject(slug string, req ProjectUpdate) (Project, error) {
	var v Project
	err := c.decode("PUT", "/projects/"+slug, req, &v)
	return v, err
}

func (c *Client) DeleteProject(slug string) error {
	return c.decode("DELETE", "/projects/"+slug, nil, nil)
}

// ---------------------------------------------------------------------------
// Builds
// ---------------------------------------------------------------------------

type Build struct {
	ID         uint   `json:"id"`
	ProjectID  uint   `json:"project_id"`
	Ref        string `json:"ref"`
	Tag        string `json:"tag"`
	Status     string `json:"status"`
	Logs       string `json:"logs,omitempty"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type BuildCreate struct {
	Ref string `json:"ref,omitempty"`
	Tag string `json:"tag,omitempty"`
}

func (c *Client) TriggerBuild(slug string, req BuildCreate) (Build, error) {
	var v Build
	err := c.decode("POST", "/projects/"+slug+"/builds", req, &v)
	return v, err
}

func (c *Client) ListBuilds(slug string, limit, offset int) ([]Build, error) {
	path := fmt.Sprintf("/projects/%s/builds?limit=%d&offset=%d", slug, limit, offset)
	var v []Build
	err := c.decode("GET", path, nil, &v)
	return v, err
}

func (c *Client) GetBuild(slug string, id uint) (Build, error) {
	var v Build
	err := c.decode("GET", fmt.Sprintf("/projects/%s/builds/%d", slug, id), nil, &v)
	return v, err
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

type Version struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	BuildID   uint   `json:"build_id"`
	Version   string `json:"version"`
	Published bool   `json:"published"`
	IsLatest  bool   `json:"is_latest"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type VersionUpdate struct {
	Published *bool `json:"published,omitempty"`
	IsLatest  *bool `json:"is_latest,omitempty"`
}

func (c *Client) ListVersions(slug string) ([]Version, error) {
	var v []Version
	err := c.decode("GET", "/projects/"+slug+"/versions", nil, &v)
	return v, err
}

func (c *Client) UpdateVersion(slug, ver string, req VersionUpdate) (Version, error) {
	var v Version
	err := c.decode("PUT", "/projects/"+slug+"/versions/"+ver, req, &v)
	return v, err
}
