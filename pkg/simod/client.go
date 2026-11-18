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
// injectionmolding and materials providers' SIMOD reconcilers instead of
// shelling out to Python.
package simod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// requestTimeout bounds every call to the SIMOD backend. http.DefaultClient
// has no timeout, so a backend that's unreachable in a way that blocks the
// TCP/TLS handshake (rather than failing fast with connection refused) would
// otherwise hang the calling reconciler - and therefore the CRD's status -
// forever, leaving the portal spinning with nothing to show.
const requestTimeout = 15 * time.Second

// Client talks to a SIMOD backend (e.g. https://backend.simod.de).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Client for the given SIMOD backend base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: requestTimeout},
	}
}

type loginResponse struct {
	Success struct {
		Token string `json:"token"`
	} `json:"success"`
}

// AuthRejectedError means the SIMOD backend was reached and responded to
// /api/login, but not with 200 OK - the backend itself is up, the problem is
// with the request (almost always wrong email/password, occasionally a
// backend-side error). Distinct from a network/timeout failure (no response
// received at all), which stays a plain wrapped error - see IsAuthRejected.
type AuthRejectedError struct {
	StatusCode int
	Body       string
}

func (e *AuthRejectedError) Error() string {
	return fmt.Sprintf("SIMOD login rejected by backend: status %d: %s", e.StatusCode, e.Body)
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
		return "", &AuthRejectedError{StatusCode: resp.StatusCode, Body: string(body)}
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

// get sends an authenticated GET and decodes the JSON response into
// respBody. Every /api/platform/... list endpoint wraps its array in a
// Laravel AnonymousResourceCollection, which serializes as {"data": [...]}
// (confirmed: no withoutWrapping() anywhere in ianus-simod's feat/platform-api
// branch, so the default wrapping applies) - callers decode into a
// `struct{ Data []T }`.
func (c *Client) get(ctx context.Context, token, path string, respBody any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, respBody); err != nil {
		return fmt.Errorf("%s: decode response: %w", path, err)
	}
	return nil
}

// SimulationResponse is the shape returned by the simulations endpoints.
type SimulationResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DigitalTwinID string `json:"digital_twin_id"`

	// Status is a coarse bucket - every non-terminal state (pending, enqueued,
	// paused, etc.) collapses to "processing" (ianus-simod's
	// Simulation::getSimpleStatus()). Kept as-is for completion checks and
	// Notification, which are keyed off this coarse value.
	Status string `json:"status"`

	// StatusDetail is the ungrouped status (e.g. "pending", "enqueued",
	// "paused"), added to ianus-simod's SimulationResource alongside `status`
	// (not replacing it) specifically so callers that want to *display* the
	// real state - rather than just check for completion - have something to
	// show. Empty on an older backend that predates this field.
	StatusDetail string `json:"status_detail"`

	Notification      string   `json:"notification"`
	SimulationFileIDs []string `json:"simulation_file_ids"`
	CreatedAt         string   `json:"created_at"`
}

// ListSimulations calls GET /api/platform/simulations, returning every
// Simulation belonging to the authenticated user's company.
func (c *Client) ListSimulations(ctx context.Context, token string) ([]SimulationResponse, error) {
	var out struct {
		Data []SimulationResponse `json:"data"`
	}
	if err := c.get(ctx, token, "/api/platform/simulations", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetSimulation calls GET /api/platform/simulations/{id}, returning the
// current state of a single Simulation. Used to poll a monitored
// simulation's status.
func (c *Client) GetSimulation(ctx context.Context, token, id string) (*SimulationResponse, error) {
	var out struct {
		Data SimulationResponse `json:"data"`
	}
	if err := c.get(ctx, token, "/api/platform/simulations/"+id, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// MaterialSection is one named section of a material characterization as
// accepted by POST /api/platform/materials - every section but "data" is a
// {method, parameters} pair (ianus-simod's MaterialController validates each
// as `sometimes|string`/`sometimes|array` - Parameters is deliberately `any`,
// not map[string]string like ComponentRequest.Parameters, because a
// section's shape varies: e.g. heat's parameters is a list of
// temperature-indexed rows, not a flat map).
type MaterialSection struct {
	Method     string `json:"method,omitempty"`
	Parameters any    `json:"parameters,omitempty"`
}

// MaterialSections groups a material's characterization data into the
// sections /api/platform/materials expects. Data is `any` for the same
// reason as MaterialSection.Parameters - the backend validates it only as
// `required|array`.
type MaterialSections struct {
	Data             any              `json:"data"`
	Temperature      *MaterialSection `json:"temperature,omitempty"`
	Viscosity        *MaterialSection `json:"viscosity,omitempty"`
	Heat             *MaterialSection `json:"heat,omitempty"`
	InjectionMolding *MaterialSection `json:"injection_molding,omitempty"`
	PVT              *MaterialSection `json:"pvt,omitempty"`
}

// MaterialRequest is the payload for POST /api/platform/materials.
type MaterialRequest struct {
	Name     string           `json:"name"`
	Sections MaterialSections `json:"sections"`

	// Temporary is the source material's raw characterization JSON, stored
	// server-side as StrömungsRaum's "Temporary Material Data" field
	// (Material::$temporary) - sent as-is, not escaped as a string.
	Temporary json.RawMessage `json:"temporary,omitempty"`
}

// MaterialResponse is the shape returned by the materials endpoints.
type MaterialResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Sections  MaterialSections `json:"sections"`
	CreatedAt string           `json:"created_at"`
}

// CreateMaterial calls POST /api/platform/materials. Like the list endpoint,
// the single-resource response is wrapped as {"data": {...}} - Laravel's
// default JsonResource wrapping, which MaterialResource (no withoutWrapping()
// or $wrap=null override) doesn't opt out of - so the response is decoded
// into a Data wrapper, not MaterialResponse directly (confirmed against
// ianus-simod's MaterialController::store()/MaterialResource, 2026-11-04).
func (c *Client) CreateMaterial(ctx context.Context, token string, req MaterialRequest) (*MaterialResponse, error) {
	var out struct {
		Data MaterialResponse `json:"data"`
	}
	if err := c.post(ctx, token, "/api/platform/materials", req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// ListMaterials calls GET /api/platform/materials, returning every Material
// belonging to the authenticated user's company.
func (c *Client) ListMaterials(ctx context.Context, token string) ([]MaterialResponse, error) {
	var out struct {
		Data []MaterialResponse `json:"data"`
	}
	if err := c.get(ctx, token, "/api/platform/materials", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
