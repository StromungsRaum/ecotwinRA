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

package ianus

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ianusv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/ianus/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

// Reconciliation phases shared by every SIMOD-backed resource (Component,
// DigitalTwin, ExecutionParameter, Simulation).
const (
	phasePending = "Pending"
	phaseReady   = "Ready"
	phaseFailed  = "Failed"
)

// dependencyRequeueAfter is how long a reconciler waits before re-checking a
// referenced resource that hasn't reached phaseReady yet.
const dependencyRequeueAfter = 5 * time.Second

// simodBaseURL is the SIMOD backend base URL used by every SIMOD reconciler,
// set once at startup via SetSIMODBaseURL (matching cmd/ianus's
// --simod-base-url flag).
var simodBaseURL = "https://backend.simod.de"

// SetSIMODBaseURL overrides the SIMOD backend base URL used by all SIMOD
// reconcilers. Call before SetupWithManager.
func SetSIMODBaseURL(url string) {
	simodBaseURL = url
}

// loginSIMOD resolves the first SecretRef's `email`/`password` keys and logs
// in against the SIMOD backend, returning a client plus bearer token ready
// to use for a Create call.
func loginSIMOD(ctx context.Context, cl client.Client, namespace string, secretRefs []ianusv1alpha1.SecretReference) (*simod.Client, string, error) {
	if len(secretRefs) == 0 {
		return nil, "", fmt.Errorf("no secretRefs configured")
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: secretRefs[0].Name}
	if err := cl.Get(ctx, key, secret); err != nil {
		return nil, "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	email := string(secret.Data["email"])
	password := string(secret.Data["password"])
	if email == "" || password == "" {
		return nil, "", fmt.Errorf("secret %s must contain non-empty email and password keys", key)
	}

	simodClient := simod.NewClient(simodBaseURL)
	token, err := simodClient.Login(ctx, email, password)
	if err != nil {
		return nil, "", fmt.Errorf("simod login failed: %w", err)
	}
	return simodClient, token, nil
}
