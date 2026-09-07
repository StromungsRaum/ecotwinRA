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

// SimulationSpec defines the desired state of Simulation
type SimulationSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// DigitalTwinRef references the DigitalTwin this simulation runs against.
	// It must reach phase Ready before the simulation can be created.
	DigitalTwinRef LocalObjectReference `json:"digitalTwinRef"`

	// ExecutionParameterRef references the ExecutionParameter whose token is
	// redeemed to create this simulation. It must reach phase Ready before
	// the simulation can be created.
	ExecutionParameterRef LocalObjectReference `json:"executionParameterRef"`
}

// SimulationStatus defines the observed state of Simulation
type SimulationStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// SimulationID is the uuid the SIMOD backend assigned to this simulation.
	// +optional
	SimulationID string `json:"simulationId,omitempty"`

	// SimulationStatus is the backend's own simulation status (e.g.
	// processing, complete, error) - distinct from Phase, which only tracks
	// whether this CR's create call has been made.
	// +optional
	SimulationStatus string `json:"simulationStatus,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.simulationStatus`

// Simulation is the Schema for the simulations API
type Simulation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimulationSpec   `json:"spec,omitempty"`
	Status SimulationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SimulationList contains a list of Simulation
type SimulationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Simulation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Simulation{}, &SimulationList{})
}
