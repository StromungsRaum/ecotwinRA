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
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoginFromSecretMissingSecretIsCredentialsError(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, _, err := LoginFromSecret(context.Background(), cl, "default", "simod-credentials", "https://backend.simod.de")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsCredentialsError(err) {
		t.Fatalf("err = %v, want a *CredentialsError", err)
	}
}

func TestLoginFromSecretEmptyKeysIsCredentialsError(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "simod-credentials", Namespace: "default"},
		Data:       map[string][]byte{"email": []byte("user@example.com")},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	_, _, err := LoginFromSecret(context.Background(), cl, "default", "simod-credentials", "https://backend.simod.de")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsCredentialsError(err) {
		t.Fatalf("err = %v, want a *CredentialsError", err)
	}
}

func TestLoginFromSecretBackendErrorResponseIsAuthRejectedNotCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "simod-credentials", Namespace: "default"},
		Data:       map[string][]byte{"email": []byte("user@example.com"), "password": []byte("hunter2")},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	_, _, err := LoginFromSecret(context.Background(), cl, "default", "simod-credentials", srv.URL)
	if err == nil {
		t.Fatal("expected an error")
	}
	if IsCredentialsError(err) {
		t.Fatalf("err = %v, want NOT a *CredentialsError (the Secret is fine)", err)
	}
	if !IsAuthRejected(err) || !NeedsCredentialsFix(err) {
		t.Fatalf("err = %v, want IsAuthRejected and NeedsCredentialsFix (backend was reached, it said no)", err)
	}
}

func TestLoginFromSecretUnreachableIsNeitherCredentialsNorAuthRejected(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "simod-credentials", Namespace: "default"},
		Data:       map[string][]byte{"email": []byte("user@example.com"), "password": []byte("hunter2")},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	// Port 0 on loopback: nothing listens there, so Do() fails before any
	// HTTP response is ever received - a real network failure, not a
	// response the backend sent us.
	_, _, err := LoginFromSecret(context.Background(), cl, "default", "simod-credentials", "http://127.0.0.1:0")
	if err == nil {
		t.Fatal("expected an error")
	}
	if NeedsCredentialsFix(err) {
		t.Fatalf("err = %v, want NOT credentials-class (no response was ever received)", err)
	}
}

func TestLoginFromSecretUsesBaseURLFromSecret(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"success":{"token":"tok-123"}}`))
	}))
	defer srv.Close()

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "simod-credentials", Namespace: "default"},
		Data: map[string][]byte{
			"email":    []byte("user@example.com"),
			"password": []byte("hunter2"),
			"baseUrl":  []byte(srv.URL),
		},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	// The default points nowhere real - if the Secret's baseUrl weren't
	// honored, this would fail to connect instead of reaching srv.
	c, token, err := LoginFromSecret(context.Background(), cl, "default", "simod-credentials", "http://127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok-123" {
		t.Fatalf("token = %q, want tok-123", token)
	}
	if c.BaseURL != srv.URL {
		t.Fatalf("c.BaseURL = %q, want %q", c.BaseURL, srv.URL)
	}
	if gotPath != "/api/login" {
		t.Fatalf("server saw path %q, want /api/login", gotPath)
	}
}

func TestResolveBaseURL(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	secretWithURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "has-url", Namespace: "default"},
		Data:       map[string][]byte{"baseUrl": []byte("https://backend.int.simod.de")},
	}
	secretWithoutURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "no-url", Namespace: "default"},
		Data:       map[string][]byte{"email": []byte("user@example.com")},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secretWithURL, secretWithoutURL).Build()
	ctx := context.Background()
	const fallback = "https://backend.dev.simod.de"

	if got := ResolveBaseURL(ctx, cl, "default", "has-url", fallback); got != "https://backend.int.simod.de" {
		t.Errorf("with baseUrl set: got %q, want https://backend.int.simod.de", got)
	}
	if got := ResolveBaseURL(ctx, cl, "default", "no-url", fallback); got != fallback {
		t.Errorf("without baseUrl set: got %q, want fallback %q", got, fallback)
	}
	if got := ResolveBaseURL(ctx, cl, "default", "does-not-exist", fallback); got != fallback {
		t.Errorf("missing secret: got %q, want fallback %q", got, fallback)
	}
	if got := ResolveBaseURL(ctx, cl, "default", "", fallback); got != fallback {
		t.Errorf("empty secretName: got %q, want fallback %q", got, fallback)
	}
}
