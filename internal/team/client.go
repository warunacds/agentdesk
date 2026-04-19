package team

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the HTTP client for the Forge Team API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// ManifestEntry represents a single file in the team manifest.
type ManifestEntry struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Version   int    `json:"version"`
	Tool      string `json:"tool"`
	Category  string `json:"category"`
	UpdatedAt string `json:"updated_at"`
}

// FileContent represents the content of a pulled file.
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	SHA256   string `json:"sha256"`
	Version  int    `json:"version"`
	Tool     string `json:"tool"`
	Category string `json:"category"`
	Name     string `json:"name"`
}

// ChangeEntry represents a file that has changed since a given time.
type ChangeEntry struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Version   int    `json:"version"`
	Tool      string `json:"tool"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
	Deleted   bool   `json:"deleted"`
}

// UserInfo represents the authenticated user.
type UserInfo struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     *string `json:"name"`
	Role     string  `json:"role"`
	OrgID    string  `json:"org_id"`
	Provider *string `json:"auth_provider"`
}

// NewClient creates a new team API client.
func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest executes an authenticated request and returns the response body.
func (c *Client) doRequest(method, path string) ([]byte, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed: invalid or expired token")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetManifest retrieves the full file manifest from the team server.
func (c *Client) GetManifest() ([]ManifestEntry, error) {
	body, err := c.doRequest("GET", "/api/v1/sync/manifest")
	if err != nil {
		return nil, err
	}
	var entries []ManifestEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return entries, nil
}

// PullFile retrieves the content of a single file by path.
func (c *Client) PullFile(path string) (*FileContent, error) {
	apiPath := "/api/v1/sync/pull/" + strings.TrimPrefix(path, "/")
	body, err := c.doRequest("GET", apiPath)
	if err != nil {
		return nil, err
	}
	var fc FileContent
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("parse file content: %w", err)
	}
	return &fc, nil
}

// GetChanges retrieves files that have changed since the given time.
func (c *Client) GetChanges(since time.Time) ([]ChangeEntry, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	body, err := c.doRequest("GET", "/api/v1/sync/changes?since="+sinceStr)
	if err != nil {
		return nil, err
	}
	var entries []ChangeEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse changes: %w", err)
	}
	return entries, nil
}

// GetMe retrieves information about the authenticated user.
func (c *Client) GetMe() (*UserInfo, error) {
	body, err := c.doRequest("GET", "/api/v1/auth/me")
	if err != nil {
		return nil, err
	}
	var user UserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}
	return &user, nil
}
