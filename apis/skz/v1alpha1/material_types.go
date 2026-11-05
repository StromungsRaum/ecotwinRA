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

// MaterialData is the Material's general/physical data section - mirrors
// EastmanPlastics1000.json's top-level "data" object.
type MaterialData struct {
	Type             string  `json:"type"`
	Norm             string  `json:"norm"`
	MaterialNumber   string  `json:"material_number"`
	Density          float64 `json:"density"`
	HeatConductivity float64 `json:"heatconductivity"`
	HeatCapacity     float64 `json:"heatcapacity"`
}

// ViscoModel is the Material's viscosity model - mirrors
// EastmanPlastics1000.json's "viscoModel" object (e.g. a "cross" model).
type ViscoModel struct {
	Model               string  `json:"model"`
	Exponent            float64 `json:"exponent"`
	ZeroViscosity       float64 `json:"zero_viscosity"`
	CriticalStressLevel float64 `json:"critical_stress_level"`
}

// ThermalDataPoint is one row of HeatModel.Thermaldata - temperature-indexed
// heat/density data, mirrors EastmanPlastics1000.json's
// "heatModel.thermaldata[]" entries.
type ThermalDataPoint struct {
	Temperature      float64 `json:"temperature"`
	HeatCapacity     float64 `json:"heatcapacity"`
	HeatConductivity float64 `json:"heatconductivity"`
	Density          float64 `json:"density"`
}

// HeatModel is the Material's heat model - mirrors
// EastmanPlastics1000.json's "heatModel" object (e.g. a "Tabulated" model).
type HeatModel struct {
	Model       string             `json:"model"`
	Thermaldata []ThermalDataPoint `json:"thermaldata"`
}

// TempModel is the Material's temperature-shift model - mirrors
// EastmanPlastics1000.json's "tempModel" object (e.g. a "c1c2" model).
type TempModel struct {
	Model                string  `json:"model"`
	C1                   float64 `json:"c1"`
	C2                   float64 `json:"c2"`
	D3                   float64 `json:"D3"`
	ReferenceTemperature float64 `json:"referenceTemperature"`
}

// PVTModel is the Material's pressure-volume-temperature model - mirrors
// EastmanPlastics1000.json's "pvtmodel" object (e.g. a "tait" model).
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

// MechProperties are the Material's mechanical properties - mirrors
// EastmanPlastics1000.json's "mech_properties" object.
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

// ShrinkageProperties are the Material's shrinkage properties - mirrors
// EastmanPlastics1000.json's "shrinkage_properties" object.
type ShrinkageProperties struct {
	Parallel      float64 `json:"parallel"`
	Perpendicular float64 `json:"perpendicular"`
	AvgNominal    float64 `json:"avg_nominal"`
}

// MaterialSpec defines the desired state of Material.
//
// SKZ is a dummy/demo provider: there is no real material-testing backend
// behind this CRD, no operator, and no status. Unlike the provider's
// original flat Parameters map[string]string bag, this shape is a 1:1
// mirror of a real material characterization file
// (docs/assets/EastmanPlastics1000.json) and of the sections
// ianus-simod's real Platform API accepts at `POST /api/platform/materials`
// (`sections.{temperature,viscosity,heat,injection_molding,pvt}`, each a
// `{method, parameters}` pair) - this CRD groups the same data by model
// instead of by generic method/parameters, since every field here has a
// known, fixed shape. The "IANUS Simulation - Materials" provider is what
// translates an instance of this CRD into that backend's sections payload.
//
// NOTE: the Kind is "Material", not "MaterialParameters" - a Kind name
// ending in "s" breaks the Platform Mesh CRD Gateway's pluralization (it
// generates no distinct list/collection field, only a singular
// MaterialParameters(name, namespace) getter), confirmed live against the
// real gateway before renaming.
type MaterialSpec struct {
	// Name is the material's name, e.g. "Eastman Plastics Foo 1000".
	Name string `json:"name"`

	// Data is the material's general/physical data.
	Data MaterialData `json:"data"`

	// ViscoModel is the material's viscosity model.
	ViscoModel ViscoModel `json:"viscoModel"`

	// HeatModel is the material's heat model.
	HeatModel HeatModel `json:"heatModel"`

	// TempModel is the material's temperature-shift model.
	TempModel TempModel `json:"tempModel"`

	// PVTModel is the material's pressure-volume-temperature model.
	PVTModel PVTModel `json:"pvtmodel"`

	// Structure describes the material's morphology, e.g. "amorphous".
	Structure string `json:"structure"`

	// MinMeltTemp is the minimum recommended melt temperature.
	MinMeltTemp float64 `json:"min_melt_temp"`

	// MaxMeltTemp is the maximum recommended melt temperature.
	MaxMeltTemp float64 `json:"max_melt_temp"`

	// AbsMaxMeltTemp is the absolute maximum melt temperature.
	AbsMaxMeltTemp float64 `json:"abs_max_melt_temp"`

	// EjectTemp is the part ejection temperature.
	EjectTemp float64 `json:"eject_temp"`

	// FreezeTemp is the freeze temperature.
	FreezeTemp float64 `json:"freeze_temp"`

	// MeltDensity is the material's density in the melt state.
	MeltDensity float64 `json:"melt_density"`

	// SolidDensity is the material's density in the solid state.
	SolidDensity float64 `json:"solid_density"`

	// MechProperties are the material's mechanical properties.
	MechProperties MechProperties `json:"mech_properties"`

	// ShrinkageProperties are the material's shrinkage properties.
	ShrinkageProperties ShrinkageProperties `json:"shrinkage_properties"`

	// MaxShearStress is the material's maximum allowable shear stress.
	MaxShearStress float64 `json:"max_shear_stress"`

	// MaxShearRate is the material's maximum allowable shear rate.
	MaxShearRate float64 `json:"max_shear_rate"`

	// RawJSON is the exact material characterization JSON the user
	// pasted/uploaded, verbatim - preserved alongside the parsed fields above
	// so it can be forwarded as-is to StrömungsRaum's "Temporary Material
	// Data" field on submission (see apis/materials/v1alpha1's
	// MaterialSubmissionData.RawJSON and pkg/simod.MaterialRequest.Temporary).
	// +optional
	RawJSON string `json:"rawJson,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.data.type`

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
