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

// InjectionMoldingServiceSpec defines the desired state of InjectionMoldingService.
// Each InjectionMoldingService monitors at most one SIMOD Simulation - like
// HttpBin, the portal creates one of these per card via its "+" button, not
// one shared resource per account.
type InjectionMoldingServiceSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// RefreshRequestedAt is bumped by the UI's Refresh button (e.g. to an RFC3339
	// timestamp) to request the reconciler re-fetch the account's Simulations
	// from the SIMOD backend, to populate status.availableSimulations for the
	// portal's selection dropdown. Any change to this value triggers a refresh;
	// the value itself is otherwise opaque.
	// +optional
	RefreshRequestedAt string `json:"refreshRequestedAt,omitempty"`

	// SimulationID is the SIMOD backend uuid of the Simulation this resource
	// monitors, chosen by the user from status.availableSimulations. Once set,
	// the reconciler polls that simulation's status every 5 seconds until it
	// completes.
	// +optional
	SimulationID string `json:"simulationId,omitempty"`
}

// SimulationSummary is a condensed view of a SIMOD Simulation, enough to
// populate the portal's simulation dropdown.
type SimulationSummary struct {
	// ID is the uuid the SIMOD backend assigned to this simulation.
	ID string `json:"id"`

	// Name is the simulation's name on the SIMOD backend.
	Name string `json:"name"`

	// Status is the simulation's current status on the SIMOD backend.
	Status string `json:"status"`
}

// MonitoredSimulationStatus is the observed state of the Simulation this
// resource is monitoring (spec.simulationId), refreshed every 5 seconds
// until it completes.
type MonitoredSimulationStatus struct {
	// ID is the uuid of the monitored simulation (mirrors spec.simulationId).
	ID string `json:"id"`

	// Name is the simulation's name on the SIMOD backend.
	Name string `json:"name"`

	// Status is the simulation's current status on the SIMOD backend (e.g.
	// "processing", "complete", "canceled", "error", "contract_limit_reached").
	Status string `json:"status"`

	// Complete is true once Status == "complete".
	Complete bool `json:"complete"`

	// ResultsURL points the user at the SIMOD web UI to view/download this
	// simulation's result files once Complete is true. The Platform API's own
	// per-file endpoint requires the same Bearer token the operator uses, which
	// a browser link can't supply, so this points at the SIMOD web UI instead
	// (session-cookie authenticated, already logged in there) rather than the
	// raw API.
	// +optional
	ResultsURL string `json:"resultsUrl,omitempty"`

	// LastPolledAt is the RFC3339 timestamp of the last status poll.
	LastPolledAt string `json:"lastPolledAt,omitempty"`
}

// InjectionMoldingServiceStatus defines the observed state of InjectionMoldingService
type InjectionMoldingServiceStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// URL points the user at the SIMOD web UI to create a DigitalTwin/Simulation
	// themselves. Derived from the operator's configured SIMOD backend base URL
	// (the "backend." host prefix stripped off), not hardcoded, so it tracks
	// whichever environment (dev/int/prod) the operator is pointed at.
	// +optional
	URL string `json:"url,omitempty"`

	// AvailableSimulations lists the SIMOD backend's existing Simulations for
	// this account, as of the last refresh - populates the portal's selection
	// dropdown.
	// +optional
	AvailableSimulations []SimulationSummary `json:"availableSimulations,omitempty"`

	// LastRefreshedAt is the RFC3339 timestamp of the last successful
	// available-simulations fetch from the SIMOD backend.
	// +optional
	LastRefreshedAt string `json:"lastRefreshedAt,omitempty"`

	// LastProcessedRefreshRequestedAt mirrors spec.refreshRequestedAt as of the
	// last successful fetch. The reconciler compares this against the current
	// spec value to decide whether a new refresh was requested, rather than
	// reacting to every reconcile (a status-only update doesn't change
	// metadata.generation, so without this marker the reconciler would re-fetch
	// on every reconcile it itself triggers).
	// +optional
	LastProcessedRefreshRequestedAt string `json:"lastProcessedRefreshRequestedAt,omitempty"`

	// MonitoredSimulation is the observed state of spec.simulationId, once set.
	// +optional
	MonitoredSimulation *MonitoredSimulationStatus `json:"monitoredSimulation,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`

// InjectionMoldingService is the Schema for the injectionmoldingservices API
type InjectionMoldingService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InjectionMoldingServiceSpec   `json:"spec,omitempty"`
	Status InjectionMoldingServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InjectionMoldingServiceList contains a list of InjectionMoldingService
type InjectionMoldingServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InjectionMoldingService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InjectionMoldingService{}, &InjectionMoldingServiceList{})
}
