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

	"sigs.k8s.io/controller-runtime/pkg/client"

	injectionmoldingv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/injectionmolding/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

const (
	phasePending = "Pending"
	phaseReady   = "Ready"
	phaseFailed  = "Failed"
)

// simodBaseURL is the SIMOD backend base URL used by the reconciler, set once
// at startup via SetSIMODBaseURL (matching cmd/injectionmolding's
// --simod-base-url flag). Also drives the status.url link - see modelsURL.
var simodBaseURL = "https://backend.dev.simod.de"

// SetSIMODBaseURL overrides the SIMOD backend base URL used by the
// reconciler. Call before SetupWithManager.
func SetSIMODBaseURL(url string) {
	simodBaseURL = url
}

// loginSIMOD resolves the first SecretRef's `email`/`password` keys and logs
// in against the SIMOD backend.
func loginSIMOD(ctx context.Context, cl client.Client, namespace string, secretRefs []injectionmoldingv1alpha1.SecretReference) (*simod.Client, string, error) {
	if len(secretRefs) == 0 {
		return nil, "", fmt.Errorf("no secretRefs configured")
	}
	return simod.LoginFromSecret(ctx, cl, namespace, secretRefs[0].Name, simodBaseURL)
}

// modelsURL derives the public SIMOD web UI link from the configured SIMOD
// backend base URL by stripping a leading "backend." host prefix, e.g.
// https://backend.dev.simod.de -> https://dev.simod.de/models3D and
// https://backend.simod.de (prod has no subdomain) -> https://simod.de/models3D.
// This keeps the link in lockstep with --simod-base-url across dev/int/prod
// instead of hardcoding one environment.
func modelsURL() (string, error) {
	u, err := url.Parse(simodBaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid simod base url %q: %w", simodBaseURL, err)
	}
	u.Host = strings.TrimPrefix(u.Host, "backend.")
	u.Path = "/models3D"
	return u.String(), nil
}
