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

package simod

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/login" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("email"); got != "user@example.com" {
			t.Fatalf("email = %q", got)
		}
		if got := r.FormValue("password"); got != "hunter2" {
			t.Fatalf("password = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": map[string]string{"token": "tok-123"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	token, err := c.Login(context.Background(), "user@example.com", "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok-123" {
		t.Fatalf("token = %q, want tok-123", token)
	}
}

func TestLoginError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Login(context.Background(), "user@example.com", "wrong")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v, want it to mention status 401", err)
	}
}

func TestCreateComponent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/components" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-123" {
			t.Fatalf("authorization = %q", got)
		}
		var body ComponentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "pump-01" || body.ComponentType != "ct-uuid" {
			t.Fatalf("body = %+v", body)
		}
		_ = json.NewEncoder(w).Encode(ComponentResponse{ID: "comp-uuid", Name: body.Name, ComponentType: body.ComponentType})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.CreateComponent(context.Background(), "tok-123", ComponentRequest{
		Name:          "pump-01",
		ComponentType: "ct-uuid",
		Parameters:    map[string]string{"length": "10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "comp-uuid" {
		t.Fatalf("id = %q, want comp-uuid", out.ID)
	}
}

func TestCreateComponentError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"validation failed"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.CreateComponent(context.Background(), "tok-123", ComponentRequest{Name: "x", ComponentType: "y"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Fatalf("error = %v, want it to mention status 422", err)
	}
}

func TestCreateDigitalTwin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/digital_twins" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body DigitalTwinRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.ComponentID != "comp-uuid" {
			t.Fatalf("body = %+v", body)
		}
		_ = json.NewEncoder(w).Encode(DigitalTwinResponse{ID: "twin-uuid", Name: body.Name, ComponentID: body.ComponentID})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.CreateDigitalTwin(context.Background(), "tok-123", DigitalTwinRequest{Name: "pump-01-twin", ComponentID: "comp-uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "twin-uuid" {
		t.Fatalf("id = %q, want twin-uuid", out.ID)
	}
}

func TestCreateExecutionParameter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/execution_parameters" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		// The real endpoint responds 201, not 200 - the client must accept both.
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ExecutionParameterResponse{ID: "encrypted-token"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.CreateExecutionParameter(context.Background(), "tok-123", ExecutionParameterRequest{
		FormID:     "form-uuid",
		Parameters: map[string]string{"pressure": "10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "encrypted-token" {
		t.Fatalf("id = %q, want encrypted-token", out.ID)
	}
}

func TestCreateSimulation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/simulations" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.DigitalTwinID != "twin-uuid" || body.ExecutionParameterID != "encrypted-token" {
			t.Fatalf("body = %+v", body)
		}
		_ = json.NewEncoder(w).Encode(SimulationResponse{ID: "sim-uuid", DigitalTwinID: body.DigitalTwinID, Status: "processing"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.CreateSimulation(context.Background(), "tok-123", SimulationRequest{
		DigitalTwinID:        "twin-uuid",
		ExecutionParameterID: "encrypted-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "sim-uuid" || out.Status != "processing" {
		t.Fatalf("out = %+v", out)
	}
}
