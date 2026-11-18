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

package injectionmolding

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	injectionmoldingv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/injectionmolding/v1alpha1"
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
// via SetSIMODBaseURL (matching cmd/injectionmolding's --simod-base-url
// flag). See simod.LoginFromSecret/simod.ResolveBaseURL for the per-Secret
// override.
var defaultSimodBaseURL = "https://backend.dev.simod.de"

// SetSIMODBaseURL overrides the default SIMOD backend base URL used by the
// reconciler when a Secret doesn't configure its own. Call before
// SetupWithManager.
func SetSIMODBaseURL(url string) {
	defaultSimodBaseURL = url
}

// loginSIMOD resolves the first SecretRef's `email`/`password` keys and logs
// in against the SIMOD backend the Secret configures (or defaultSimodBaseURL
// if it doesn't set its own).
func loginSIMOD(ctx context.Context, cl client.Client, namespace string, secretRefs []injectionmoldingv1alpha1.SecretReference) (*simod.Client, string, error) {
	if len(secretRefs) == 0 {
		return nil, "", fmt.Errorf("no secretRefs configured")
	}
	return simod.LoginFromSecret(ctx, cl, namespace, secretRefs[0].Name, defaultSimodBaseURL)
}

// resolveBaseURL peeks at the first SecretRef's "baseUrl" key (see
// simod.ResolveBaseURL) without requiring a successful login - used to build
// an informational link before credentials are necessarily configured yet.
func resolveBaseURL(ctx context.Context, cl client.Client, namespace string, secretRefs []injectionmoldingv1alpha1.SecretReference) string {
	secretName := ""
	if len(secretRefs) > 0 {
		secretName = secretRefs[0].Name
	}
	return simod.ResolveBaseURL(ctx, cl, namespace, secretName, defaultSimodBaseURL)
}

// modelsURL derives the public SIMOD web UI link from baseURL by stripping a
// leading "backend." host prefix, e.g. https://backend.dev.simod.de ->
// https://dev.simod.de/models3D and https://backend.simod.de (prod has no
// subdomain) -> https://simod.de/models3D.
func modelsURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid simod base url %q: %w", baseURL, err)
	}
	u.Host = strings.TrimPrefix(u.Host, "backend.")
	u.Path = "/models3D"
	return u.String(), nil
}

// simulationURL derives the public SIMOD web UI link for a single
// simulation's results page from baseURL, e.g.
// https://backend.dev.simod.de -> https://dev.simod.de/simulation/<id>.
// id is the Platform API's Simulation.ID, which is already the backend's
// simulation uuid (ianus-simod's SimulationResource serializes the uuid
// column as "id" - see pkg/simod/client.go's SimulationResponse).
func simulationURL(baseURL, id string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid simod base url %q: %w", baseURL, err)
	}
	u.Host = strings.TrimPrefix(u.Host, "backend.")
	u.Path = "/simulation/" + id
	return u.String(), nil
}

// displayStatus prefers a simod.SimulationResponse's ungrouped StatusDetail
// (e.g. "pending", "enqueued", "paused") over its coarse Status (which
// collapses every non-terminal state to "processing" - see
// SimulationResponse's doc), falling back to Status if StatusDetail is empty
// (an older backend that predates the status_detail field).
func displayStatus(sim simod.SimulationResponse) string {
	if sim.StatusDetail != "" {
		return sim.StatusDetail
	}
	return sim.Status
}
