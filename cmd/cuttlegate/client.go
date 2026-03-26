package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// apiClient wraps an HTTP client with auth and base URL.
type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// newClient builds an apiClient from config and global flag overrides.
func newClient(g *globalFlags) (*apiClient, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	server := g.Server
	if server == "" {
		server = cfg.Server
	}
	if server == "" {
		return nil, fmt.Errorf("server not configured — run 'cuttlegate config set server <URL>'")
	}
	server = strings.TrimRight(server, "/")

	tok, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("loading token: %w", err)
	}
	if tok == nil {
		return nil, fmt.Errorf("not logged in — run 'cuttlegate login'")
	}
	if tok.Expired() {
		return nil, fmt.Errorf("token expired — run 'cuttlegate login' to re-authenticate")
	}

	return &apiClient{
		baseURL: server,
		token:   tok.AccessToken,
		http:    &http.Client{},
	}, nil
}

// do sends a request and returns the response body. It handles 401 specially.
func (c *apiClient) do(method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized — run 'cuttlegate login' to re-authenticate")
	}
	if resp.StatusCode >= 400 {
		// Try to extract error message from JSON.
		var errBody struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &errBody) == nil {
			msg := errBody.Error
			if msg == "" {
				msg = errBody.Message
			}
			if msg != "" {
				return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// resolveProjectEnv returns the project and environment slugs from flags or config.
func resolveProjectEnv(g *globalFlags) (project, environment string, err error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", "", err
	}
	project = g.Project
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		return "", "", fmt.Errorf("project not specified — use --project or 'cuttlegate config set project <SLUG>'")
	}

	environment = g.Environment
	if environment == "" {
		environment = cfg.Environment
	}
	if environment == "" {
		return "", "", fmt.Errorf("environment not specified — use --environment or 'cuttlegate config set environment <SLUG>'")
	}
	return project, environment, nil
}
