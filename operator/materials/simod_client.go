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

package materials

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	materialsv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/materials/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

const (
	phasePending = "Pending"
	phaseReady   = "Ready"
	phaseFailed  = "Failed"
)

// credentialsRecheckInterval is how soon a reconcile is retried after a
// *simod.CredentialsError (missing/invalid Secret) - short and fixed, not
// the default escalating backoff a returned error would otherwise trigger,
// since this is an expected, human-actionable state ("create the Secret"),
// not an operator bug worth escalating/log-spamming about.
const credentialsRecheckInterval = 30 * time.Second

// loginResult turns a loginSIMOD error into the right reconcile outcome: a
// credentials-class error (simod.NeedsCredentialsFix - the Secret is
// missing/invalid, or the backend rejected it) is already reflected in
// status by the caller and recovers on its own once an admin fixes the
// Secret, so it's requeued directly at a short fixed interval instead of
// being returned as a reconcile error - that keeps it out of the default
// exponential backoff and the "reconciler error" log level reserved for
// unexpected failures (like a genuine backend/network problem, which still
// returns err below).
func loginResult(err error) (ctrl.Result, error) {
	if simod.NeedsCredentialsFix(err) {
		return reconcile.Result{RequeueAfter: credentialsRecheckInterval}, nil
	}
	return reconcile.Result{}, err
}

// defaultSimodBaseURL is the SIMOD backend base URL used when a resource's
// credentials Secret doesn't set its own "baseUrl" key, set once at startup
// via SetSIMODBaseURL (matching cmd/materials's --simod-base-url flag). See
// simod.LoginFromSecret for the per-Secret override.
var defaultSimodBaseURL = "https://backend.dev.simod.de"

// SetSIMODBaseURL overrides the default SIMOD backend base URL used by the
// reconcilers when a Secret doesn't configure its own. Call before
// SetupWithManager.
func SetSIMODBaseURL(url string) {
	defaultSimodBaseURL = url
}

// loginSIMOD resolves the first SecretRef's `email`/`password` keys and logs
// in against the SIMOD backend the Secret configures (or defaultSimodBaseURL
// if it doesn't set its own).
func loginSIMOD(ctx context.Context, cl client.Client, namespace string, secretRefs []materialsv1alpha1.SecretReference) (*simod.Client, string, error) {
	if len(secretRefs) == 0 {
		return nil, "", fmt.Errorf("no secretRefs configured")
	}
	return simod.LoginFromSecret(ctx, cl, namespace, secretRefs[0].Name, defaultSimodBaseURL)
}

// materialURL derives the public SIMOD web UI link for a single material
// from baseURL, e.g. https://backend.dev.simod.de ->
// https://dev.simod.de/material/<uuid> (same "strip backend. prefix"
// pattern as operator/injectionmolding's simulationURL). id is the Platform
// API's Material.ID, already the backend's material uuid (ianus-simod's
// MaterialResource serializes the uuid column as "id" - see
// pkg/simod/client.go's MaterialResponse).
func materialURL(baseURL, id string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid simod base url %q: %w", baseURL, err)
	}
	u.Host = strings.TrimPrefix(u.Host, "backend.")
	u.Path = "/material/" + id
	return u.String(), nil
}
