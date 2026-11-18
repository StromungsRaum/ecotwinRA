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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CredentialsError means the SIMOD credentials Secret itself is the problem
// (missing, or present but missing its email/password keys) - a static
// config issue a human has to fix, not a transient backend/network failure.
// Callers use errors.As to tell the two apart and react differently (see
// IsCredentialsError).
type CredentialsError struct {
	Secret client.ObjectKey
	Reason string
}

func (e *CredentialsError) Error() string {
	return fmt.Sprintf("SIMOD credentials secret %s %s", e.Secret, e.Reason)
}

// IsCredentialsError reports whether err is a *CredentialsError (possibly
// wrapped) - true for "the credentials Secret is missing/invalid", false for
// a login/network failure against an otherwise-valid Secret.
func IsCredentialsError(err error) bool {
	var credErr *CredentialsError
	return errors.As(err, &credErr)
}

// IsAuthRejected reports whether err is a *AuthRejectedError (possibly
// wrapped) - true for "the backend was reached and rejected the login",
// false for a network/timeout failure where no response was ever received.
func IsAuthRejected(err error) bool {
	var authErr *AuthRejectedError
	return errors.As(err, &authErr)
}

// NeedsCredentialsFix reports whether err means a human has to go fix the
// credentials themselves (missing Secret, or a Secret the backend rejects) -
// as opposed to a transient backend/network problem that might resolve on
// its own. Callers that want "is this credentials-class" without caring
// which of the two it is (e.g. to decide the reconcile requeue strategy)
// should use this instead of checking IsCredentialsError/IsAuthRejected
// separately.
func NeedsCredentialsFix(err error) bool {
	return IsCredentialsError(err) || IsAuthRejected(err)
}

// LoginFromSecret resolves the named Secret's `email`/`password` keys and
// logs in against the SIMOD backend, returning a client plus bearer token
// ready to use for subsequent calls. Shared by every provider operator that
// needs to authenticate against SIMOD from a namespace-local credentials
// Secret. Returns a *CredentialsError (see IsCredentialsError) when the
// Secret itself is missing or invalid, so callers can propagate that
// distinctly from - and faster than - a backend connection failure.
//
// The backend base URL isn't fixed: if the Secret has a non-empty "baseUrl"
// key, that's used (the returned Client.BaseURL reflects whichever one was
// actually used); otherwise defaultBaseURL (the operator's --simod-base-url
// flag) applies. This lets each account point its own credentials at its
// own SIMOD instance (e.g. a different dev/int/prod backend) without a
// per-provider flag or CRD field.
func LoginFromSecret(ctx context.Context, cl client.Client, namespace, secretName, defaultBaseURL string) (*Client, string, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: secretName}
	if err := cl.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, "", &CredentialsError{Secret: key, Reason: "not found - create it with non-empty email and password keys"}
		}
		return nil, "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	email := string(secret.Data["email"])
	password := string(secret.Data["password"])
	if email == "" || password == "" {
		return nil, "", &CredentialsError{Secret: key, Reason: "exists but is missing a non-empty email or password key"}
	}

	baseURL := defaultBaseURL
	if v := string(secret.Data["baseUrl"]); v != "" {
		baseURL = v
	}

	simodClient := NewClient(baseURL)
	token, err := simodClient.Login(ctx, email, password)
	if err != nil {
		return nil, "", fmt.Errorf("simod login failed: %w", err)
	}
	return simodClient, token, nil
}

// ResolveBaseURL returns the SIMOD backend base URL a credentials Secret
// configures via its optional "baseUrl" key, or defaultBaseURL if
// secretName is empty or the Secret doesn't exist or doesn't set one.
// Unlike LoginFromSecret, a missing/invalid Secret is not an error here -
// this is a best-effort peek for building an informational link (e.g. "open
// the SIMOD web UI") before any login is necessarily attempted yet.
func ResolveBaseURL(ctx context.Context, cl client.Client, namespace, secretName, defaultBaseURL string) string {
	if secretName == "" {
		return defaultBaseURL
	}
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, secret); err != nil {
		return defaultBaseURL
	}
	if v := string(secret.Data["baseUrl"]); v != "" {
		return v
	}
	return defaultBaseURL
}
