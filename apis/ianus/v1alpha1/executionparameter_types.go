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

// ExecutionParameterSpec defines the desired state of ExecutionParameter
type ExecutionParameterSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// FormID is the uuid of the SIMOD backend Form this execution parameter
	// set is validated against (see `GET /api/platform/forms`).
	FormID string `json:"formId"`

	// Parameters are the form's input parameter values.
	Parameters map[string]string `json:"parameters"`
}

// ExecutionParameterStatus defines the observed state of ExecutionParameter
type ExecutionParameterStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Token is the opaque, encrypted execution-parameter reference the SIMOD
	// backend issued. It is not persisted backend-side (there is no GET for
	// this resource) and is redeemable for a Simulation - handle it like a
	// credential, not a plain id.
	// +optional
	Token string `json:"token,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// ExecutionParameter is the Schema for the executionparameters API
type ExecutionParameter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutionParameterSpec   `json:"spec,omitempty"`
	Status ExecutionParameterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExecutionParameterList contains a list of ExecutionParameter
type ExecutionParameterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExecutionParameter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExecutionParameter{}, &ExecutionParameterList{})
}
