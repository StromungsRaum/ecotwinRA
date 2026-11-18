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
	if !IsAuthRejected(err) {
		t.Fatalf("err = %v, want a *AuthRejectedError (backend was reached, it said no)", err)
	}
}

func TestListSimulations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/simulations" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-123" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []SimulationResponse{{ID: "sim-uuid", Name: "mold_gear", Status: "processing"}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.ListSimulations(context.Background(), "tok-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "sim-uuid" {
		t.Fatalf("out = %+v", out)
	}
}

func TestGetSimulation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/platform/simulations/sim-uuid" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": SimulationResponse{ID: "sim-uuid", Name: "mold_gear", Status: "complete"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	out, err := c.GetSimulation(context.Background(), "tok-123", "sim-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "complete" {
		t.Fatalf("status = %q, want complete", out.Status)
	}
}
