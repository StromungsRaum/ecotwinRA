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

// ComponentSpec defines the desired state of Component
type ComponentSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// Name is the component's name on the SIMOD backend
	Name string `json:"name"`

	// ComponentType is the uuid of the SIMOD component type this component is
	// an instance of (see the backend's `GET /api/platform/component_types`
	// for the available types and the parameter/child-slot schema each one
	// expects).
	ComponentType string `json:"componentType"`

	// Parameters are the type-dependent parameter values ComponentType expects.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`

	// Components references child components by slot name, pointing at other
	// Component resources in this namespace. Each referenced Component must
	// reach phase Ready before this component can be created on the backend.
	// +optional
	Components map[string]LocalObjectReference `json:"components,omitempty"`
}

// ComponentStatus defines the observed state of Component
type ComponentStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// ComponentID is the uuid the SIMOD backend assigned to this component.
	// +optional
	ComponentID string `json:"componentId,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="ComponentID",type=string,JSONPath=`.status.componentId`

// Component is the Schema for the components API
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec   `json:"spec,omitempty"`
	Status ComponentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComponentList contains a list of Component
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Component `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
}
