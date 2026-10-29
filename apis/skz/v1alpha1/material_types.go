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

// MaterialSpec defines the desired state of Material.
//
// SKZ is a dummy/demo provider: there is no real material-testing backend
// behind this CRD, no operator, and no status - the portal fills Parameters
// with canned placeholder values at creation time. The field is a flat
// key/value bag rather than the real ianus-simod Platform API's sectioned
// `POST /api/platform/materials` shape (sections.{temperature,viscosity,heat,
// injection_molding,pvt}, each {method, parameters}) deliberately - this is
// illustrative, not a real material characterization.
//
// NOTE: the Kind is "Material", not "MaterialParameters" - a Kind name
// ending in "s" breaks the Platform Mesh CRD Gateway's pluralization (it
// generates no distinct list/collection field, only a singular
// MaterialParameters(name, namespace) getter), confirmed live against the
// real gateway before renaming.
type MaterialSpec struct {
	// Name is the material's name, e.g. "Luran 368R", chosen by the user when
	// provisioning this material.
	Name string `json:"name"`

	// Parameters are dummy material parameter values keyed by name (e.g.
	// "density", "meltTemperature"). Filled in by the portal with placeholder
	// values, not measured data.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`

// Material is the Schema for the materials API
type Material struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MaterialSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// MaterialList contains a list of Material
type MaterialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Material `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Material{}, &MaterialList{})
}
