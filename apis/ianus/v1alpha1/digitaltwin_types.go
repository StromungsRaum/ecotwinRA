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

// DigitalTwinSpec defines the desired state of DigitalTwin
type DigitalTwinSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// Name is the digital twin's name on the SIMOD backend
	Name string `json:"name"`

	// ComponentRef references the Component this digital twin is built from.
	// It must reach phase Ready before the digital twin can be created.
	ComponentRef LocalObjectReference `json:"componentRef"`
}

// DigitalTwinStatus defines the observed state of DigitalTwin
type DigitalTwinStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// DigitalTwinID is the uuid the SIMOD backend assigned to this digital twin.
	// +optional
	DigitalTwinID string `json:"digitalTwinId,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="DigitalTwinID",type=string,JSONPath=`.status.digitalTwinId`

// DigitalTwin is the Schema for the digitaltwins API
type DigitalTwin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DigitalTwinSpec   `json:"spec,omitempty"`
	Status DigitalTwinStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DigitalTwinList contains a list of DigitalTwin
type DigitalTwinList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DigitalTwin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DigitalTwin{}, &DigitalTwinList{})
}
