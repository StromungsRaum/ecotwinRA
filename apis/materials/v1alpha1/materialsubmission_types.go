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

// The following types mirror apis/skz/v1alpha1's Material CRD 1:1 (and, in
// turn, docs/assets/EastmanPlastics1000.json) - the portal copies a chosen
// SKZ Material's spec verbatim into MaterialSubmission.Spec.Material at
// creation time; the operator never reads the SKZ CRD itself (composing
// across providers happens in the portal, not via a cross-APIExport
// permission claim - see apis/skz/v1alpha1/material_types.go). Duplicated
// here rather than imported across API groups, following this repo's
// existing convention of a per-group copy (see SecretReference, separately
// defined in ianus/injectionmolding/materials's own common_types.go).

// MaterialData is the Material's general/physical data section.
type MaterialData struct {
	Type             string  `json:"type"`
	Norm             string  `json:"norm"`
	MaterialNumber   string  `json:"material_number"`
	Density          float64 `json:"density"`
	HeatConductivity float64 `json:"heatconductivity"`
	HeatCapacity     float64 `json:"heatcapacity"`
}

// ViscoModel is the Material's viscosity model (e.g. a "cross" model).
type ViscoModel struct {
	Model               string  `json:"model"`
	Exponent            float64 `json:"exponent"`
	ZeroViscosity       float64 `json:"zero_viscosity"`
	CriticalStressLevel float64 `json:"critical_stress_level"`
}

// ThermalDataPoint is one row of HeatModel.Thermaldata - temperature-indexed
// heat/density data.
type ThermalDataPoint struct {
	Temperature      float64 `json:"temperature"`
	HeatCapacity     float64 `json:"heatcapacity"`
	HeatConductivity float64 `json:"heatconductivity"`
	Density          float64 `json:"density"`
}

// HeatModel is the Material's heat model (e.g. a "Tabulated" model).
type HeatModel struct {
	Model       string             `json:"model"`
	Thermaldata []ThermalDataPoint `json:"thermaldata"`
}

// TempModel is the Material's temperature-shift model (e.g. a "c1c2" model).
type TempModel struct {
	Model                string  `json:"model"`
	C1                   float64 `json:"c1"`
	C2                   float64 `json:"c2"`
	D3                   float64 `json:"D3"`
	ReferenceTemperature float64 `json:"referenceTemperature"`
}

// PVTModel is the Material's pressure-volume-temperature model (e.g. a
// "tait" model).
type PVTModel struct {
	Model string  `json:"model"`
	B1s   float64 `json:"b1s"`
	B2s   float64 `json:"b2s"`
	B3s   float64 `json:"b3s"`
	B4s   float64 `json:"b4s"`
	B1m   float64 `json:"b1m"`
	B2m   float64 `json:"b2m"`
	B3m   float64 `json:"b3m"`
	B4m   float64 `json:"b4m"`
	B5    float64 `json:"b5"`
	B6    float64 `json:"b6"`
	B7    float64 `json:"b7"`
	B8    float64 `json:"b8"`
	B9    float64 `json:"b9"`
}

// MechProperties are the Material's mechanical properties.
type MechProperties struct {
	E1     float64 `json:"E1"`
	E2     float64 `json:"E2"`
	EAvg   float64 `json:"E_avg"`
	V12    float64 `json:"v12"`
	V23    float64 `json:"v23"`
	Alpha1 float64 `json:"alpha_1"`
	Alpha2 float64 `json:"alpha_2"`
	G12    float64 `json:"G12"`
}

// ShrinkageProperties are the Material's shrinkage properties.
type ShrinkageProperties struct {
	Parallel      float64 `json:"parallel"`
	Perpendicular float64 `json:"perpendicular"`
	AvgNominal    float64 `json:"avg_nominal"`
}

// MaterialSubmissionData is a full material characterization - the portal's
// copy of a chosen apis/skz/v1alpha1.Material's spec, embedded verbatim.
type MaterialSubmissionData struct {
	Name                string              `json:"name"`
	Data                MaterialData        `json:"data"`
	ViscoModel          ViscoModel          `json:"viscoModel"`
	HeatModel           HeatModel           `json:"heatModel"`
	TempModel           TempModel           `json:"tempModel"`
	PVTModel            PVTModel            `json:"pvtmodel"`
	Structure           string              `json:"structure"`
	MinMeltTemp         float64             `json:"min_melt_temp"`
	MaxMeltTemp         float64             `json:"max_melt_temp"`
	AbsMaxMeltTemp      float64             `json:"abs_max_melt_temp"`
	EjectTemp           float64             `json:"eject_temp"`
	FreezeTemp          float64             `json:"freeze_temp"`
	MeltDensity         float64             `json:"melt_density"`
	SolidDensity        float64             `json:"solid_density"`
	MechProperties      MechProperties      `json:"mech_properties"`
	ShrinkageProperties ShrinkageProperties `json:"shrinkage_properties"`
	MaxShearStress      float64             `json:"max_shear_stress"`
	MaxShearRate        float64             `json:"max_shear_rate"`
}

// SourceMaterialRef records which SKZ Material this submission's data was
// copied from, for display/audit only - the operator never reads it back
// from the SKZ CRD (see package doc above).
type SourceMaterialRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// MaterialSubmissionSpec defines the desired state of MaterialSubmission.
// Each MaterialSubmission submits at most one material to StrömungsRaum -
// like InjectionMoldingService, the portal creates one of these per card via
// its "+" button, not one shared resource per account.
type MaterialSubmissionSpec struct {
	// SecretRefs is an array of references to Secrets containing SIMOD credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`

	// SourceMaterialRef records which SKZ Material this submission's data
	// came from, for display only.
	// +optional
	SourceMaterialRef *SourceMaterialRef `json:"sourceMaterialRef,omitempty"`

	// Material is the full material characterization to submit, copied by
	// the portal from the chosen SKZ Material at creation time. The
	// StrömungsRaum backend's materials endpoint has no update route, so the
	// reconciler only ever creates once (see status.materialId).
	// +optional
	Material *MaterialSubmissionData `json:"material,omitempty"`
}

// MaterialSubmissionStatus defines the observed state of MaterialSubmission
type MaterialSubmissionStatus struct {
	// Phase is the current reconciliation phase: Pending, Ready, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// MaterialID is the StrömungsRaum backend uuid assigned to this material
	// once submitted.
	// +optional
	MaterialID string `json:"materialId,omitempty"`

	// MaterialURL is the public SIMOD web UI link to the newly created
	// material (https://<env>.simod.de/material/<uuid>), set alongside
	// MaterialID once submitted.
	// +optional
	MaterialURL string `json:"materialUrl,omitempty"`

	// SubmittedAt is the RFC3339 timestamp this material was submitted at.
	// +optional
	SubmittedAt string `json:"submittedAt,omitempty"`

	// Message carries error details when Phase is Failed.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="MaterialID",type=string,JSONPath=`.status.materialId`

// MaterialSubmission is the Schema for the materialsubmissions API
type MaterialSubmission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaterialSubmissionSpec   `json:"spec,omitempty"`
	Status MaterialSubmissionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MaterialSubmissionList contains a list of MaterialSubmission
type MaterialSubmissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaterialSubmission `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaterialSubmission{}, &MaterialSubmissionList{})
}
