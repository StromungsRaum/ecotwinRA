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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoginFromSecret resolves the named Secret's `email`/`password` keys and
// logs in against the SIMOD backend at baseURL, returning a client plus
// bearer token ready to use for subsequent calls. Shared by every IANUS
// provider operator that needs to authenticate against SIMOD from a
// namespace-local credentials Secret.
func LoginFromSecret(ctx context.Context, cl client.Client, namespace, secretName, baseURL string) (*Client, string, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: secretName}
	if err := cl.Get(ctx, key, secret); err != nil {
		return nil, "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	email := string(secret.Data["email"])
	password := string(secret.Data["password"])
	if email == "" || password == "" {
		return nil, "", fmt.Errorf("secret %s must contain non-empty email and password keys", key)
	}

	simodClient := NewClient(baseURL)
	token, err := simodClient.Login(ctx, email, password)
	if err != nil {
		return nil, "", fmt.Errorf("simod login failed: %w", err)
	}
	return simodClient, token, nil
}
