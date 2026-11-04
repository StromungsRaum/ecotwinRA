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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MaterialSummary is a condensed view of a StrömungsRaum Material, enough to
// populate the portal's catalog browse view.
type MaterialSummary struct {
	// ID is the uuid the StrömungsRaum backend assigned to this material.
	ID string `json:"id"`

	// Name is the material's name on the StrömungsRaum backend.
	Name string `json:"name"`

	// CreatedAt is when the material was created on the StrömungsRaum backend.
	CreatedAt string `json:"createdAt"`
}

// MaterialCatalogSpec defines the desired state of MaterialCatalog.
//
// Unlike MaterialSubmission, this is a singleton per account - it exists
// purely to browse StrömungsRaum's existing materials into status, not to
// create new ones.
type MaterialCatalogSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// RefreshRequestedAt is bumped by the UI's Refresh button to request the
	// reconciler re-fetch the account's Materials from the StrömungsRaum
	// backend, populating status.availableMaterials. Any change to this
	// value triggers a refresh; the value itself is otherwise opaque.
	// +optional
	RefreshRequestedAt string `json:"refreshRequestedAt,omitempty"`
}

// MaterialCatalogStatus defines the observed state of MaterialCatalog
type MaterialCatalogStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// AvailableMaterials lists the StrömungsRaum backend's existing
	// Materials for this account, as of the last refresh.
	// +optional
	AvailableMaterials []MaterialSummary `json:"availableMaterials,omitempty"`

	// LastRefreshedAt is the RFC3339 timestamp of the last successful
	// available-materials fetch from the StrömungsRaum backend.
	// +optional
	LastRefreshedAt string `json:"lastRefreshedAt,omitempty"`

	// LastProcessedRefreshRequestedAt mirrors spec.refreshRequestedAt as of
	// the last successful fetch. The reconciler compares this against the
	// current spec value to decide whether a new refresh was requested,
	// rather than reacting to every reconcile (a status-only update doesn't
	// change metadata.generation, so without this marker the reconciler
	// would re-fetch on every reconcile it itself triggers).
	// +optional
	LastProcessedRefreshRequestedAt string `json:"lastProcessedRefreshRequestedAt,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// MaterialCatalog is the Schema for the materialcatalogs API
type MaterialCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaterialCatalogSpec   `json:"spec,omitempty"`
	Status MaterialCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MaterialCatalogList contains a list of MaterialCatalog
type MaterialCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaterialCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaterialCatalog{}, &MaterialCatalogList{})
}
