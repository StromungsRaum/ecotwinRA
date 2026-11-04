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

	"sigs.k8s.io/controller-runtime/pkg/client"

	materialsv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/materials/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

const (
	phasePending = "Pending"
	phaseReady   = "Ready"
	phaseFailed  = "Failed"
)

// simodBaseURL is the SIMOD backend base URL used by both reconcilers, set
// once at startup via SetSIMODBaseURL (matching cmd/materials's
// --simod-base-url flag).
var simodBaseURL = "https://backend.dev.simod.de"

// SetSIMODBaseURL overrides the SIMOD backend base URL used by the
// reconcilers. Call before SetupWithManager.
func SetSIMODBaseURL(url string) {
	simodBaseURL = url
}

// loginSIMOD resolves the first SecretRef's `email`/`password` keys and logs
// in against the SIMOD backend.
func loginSIMOD(ctx context.Context, cl client.Client, namespace string, secretRefs []materialsv1alpha1.SecretReference) (*simod.Client, string, error) {
	if len(secretRefs) == 0 {
		return nil, "", fmt.Errorf("no secretRefs configured")
	}
	return simod.LoginFromSecret(ctx, cl, namespace, secretRefs[0].Name, simodBaseURL)
}

// materialURL derives the public SIMOD web UI link for a single material from
// the configured SIMOD backend base URL, e.g.
// https://backend.dev.simod.de -> https://dev.simod.de/material/<uuid> (same
// "strip backend. prefix" pattern as operator/injectionmolding's
// simulationURL). id is the Platform API's Material.ID, already the
// backend's material uuid (ianus-simod's MaterialResource serializes the
// uuid column as "id" - see pkg/simod/client.go's MaterialResponse).
func materialURL(id string) (string, error) {
	u, err := url.Parse(simodBaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid simod base url %q: %w", simodBaseURL, err)
	}
	u.Host = strings.TrimPrefix(u.Host, "backend.")
	u.Path = "/material/" + id
	return u.String(), nil
}
