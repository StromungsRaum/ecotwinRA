/*
Copyright 2025 The Platform Mesh Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package simod is a native Go client for the SIMOD backend's token-based
// "Platform API" (POST /api/login, /api/platform/...). It is the Go
// counterpart to ecotwin/common/api_connector.py's JobHandler, used by the
// IANUS operator's SIMOD reconcilers instead of shelling out to Python.
package simod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client talks to a SIMOD backend (e.g. https://backend.simod.de).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Client for the given SIMOD backend base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: http.DefaultClient,
	}
}

type loginResponse struct {
	Success struct {
		Token string `json:"token"`
	} `json:"success"`
}

// Login authenticates against POST /api/login and returns a bearer token.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	form := url.Values{"email": {email}, "password": {password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/login", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login: unexpected status %d: %s", resp.StatusCode, body)
	}

	var out loginResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("login: decode response: %w", err)
	}
	if out.Success.Token == "" {
		return "", fmt.Errorf("login: response did not contain a token: %s", body)
	}
	return out.Success.Token, nil
}

// post sends an authenticated JSON POST and decodes the JSON response into
// respBody (skipped if respBody is nil). Every /api/platform/... create call
// shares this shape.
func (c *Client) post(ctx context.Context, token, path string, reqBody, respBody any) error {
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("%s: encode request: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%s: unexpected status %d: %s", path, resp.StatusCode, body)
	}

	if respBody != nil {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("%s: decode response: %w", path, err)
		}
	}
	return nil
}

// ComponentRequest is the payload for POST /api/platform/components.
type ComponentRequest struct {
	Name          string            `json:"name"`
	ComponentType string            `json:"component_type"`
	Parameters    map[string]string `json:"parameters"`
	Components    map[string]string `json:"components"`
}

// ComponentResponse is the shape returned by the components endpoints.
type ComponentResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	ComponentType string            `json:"component_type"`
	Parameters    map[string]string `json:"parameters"`
	Components    map[string]string `json:"components"`
	CreatedAt     string            `json:"created_at"`
}

// CreateComponent calls POST /api/platform/components.
func (c *Client) CreateComponent(ctx context.Context, token string, req ComponentRequest) (*ComponentResponse, error) {
	var out ComponentResponse
	if err := c.post(ctx, token, "/api/platform/components", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DigitalTwinRequest is the payload for POST /api/platform/digital_twins.
type DigitalTwinRequest struct {
	Name        string `json:"name"`
	ComponentID string `json:"component_id"`
}

// DigitalTwinResponse is the shape returned by the digital_twins endpoints.
type DigitalTwinResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ComponentID string `json:"component_id"`
	CreatedAt   string `json:"created_at"`
}

// CreateDigitalTwin calls POST /api/platform/digital_twins.
func (c *Client) CreateDigitalTwin(ctx context.Context, token string, req DigitalTwinRequest) (*DigitalTwinResponse, error) {
	var out DigitalTwinResponse
	if err := c.post(ctx, token, "/api/platform/digital_twins", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecutionParameterRequest is the payload for POST /api/platform/execution_parameters.
type ExecutionParameterRequest struct {
	FormID     string            `json:"form_id"`
	Parameters map[string]string `json:"parameters"`
}

// ExecutionParameterResponse is the shape returned by the execution_parameters
// endpoint. There is no GET for this resource - it is minted fresh per call.
type ExecutionParameterResponse struct {
	ID string `json:"id"`
}

// CreateExecutionParameter calls POST /api/platform/execution_parameters.
func (c *Client) CreateExecutionParameter(ctx context.Context, token string, req ExecutionParameterRequest) (*ExecutionParameterResponse, error) {
	var out ExecutionParameterResponse
	if err := c.post(ctx, token, "/api/platform/execution_parameters", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SimulationRequest is the payload for POST /api/platform/simulations.
type SimulationRequest struct {
	DigitalTwinID        string `json:"digital_twin_id"`
	ExecutionParameterID string `json:"execution_parameter_id"`
}

// SimulationResponse is the shape returned by the simulations endpoints.
type SimulationResponse struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	DigitalTwinID     string   `json:"digital_twin_id"`
	Status            string   `json:"status"`
	Notification      string   `json:"notification"`
	SimulationFileIDs []string `json:"simulation_file_ids"`
	CreatedAt         string   `json:"created_at"`
}

// CreateSimulation calls POST /api/platform/simulations.
func (c *Client) CreateSimulation(ctx context.Context, token string, req SimulationRequest) (*SimulationResponse, error) {
	var out SimulationResponse
	if err := c.post(ctx, token, "/api/platform/simulations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
