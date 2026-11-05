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

package materials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	materialsv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/materials/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

// MaterialSubmissionReconciler submits one material to StrömungsRaum per
// resource, create-once (the backend's materials endpoint has no
// update/delete route, same as Component/DigitalTwin/Simulation - see
// CLAUDE.md's "SIMOD CRDs" section): the portal creates one of these per
// card via its "+" button with spec.material already populated (copied from
// a chosen SKZ Material), and the reconciler no-ops once status.materialId
// is set.
type MaterialSubmissionReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *MaterialSubmissionReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("materialsubmission-controller").
		For(&materialsv1alpha1.MaterialSubmission{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of MaterialSubmission resources across
// clusters.
func (r *MaterialSubmissionReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	sub := &materialsv1alpha1.MaterialSubmission{}
	if err := client.Get(ctx, req.NamespacedName, sub); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get materialsubmission: %w", err)
	}

	// Create-once: once submitted, never resubmit.
	if sub.Status.MaterialID != "" {
		return reconcile.Result{}, nil
	}

	if sub.Spec.Material == nil {
		sub.Status.Phase = phasePending
		sub.Status.Message = "waiting for spec.material to be set"
		_ = client.Status().Update(ctx, sub)
		return reconcile.Result{}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, sub.Namespace, sub.Spec.SecretRefs)
	if err != nil {
		sub.Status.Phase = phaseFailed
		sub.Status.Message = err.Error()
		_ = client.Status().Update(ctx, sub)
		return reconcile.Result{}, err
	}

	resp, err := simodClient.CreateMaterial(ctx, token, buildMaterialRequest(sub.Spec.Material))
	if err != nil {
		sub.Status.Phase = phaseFailed
		sub.Status.Message = err.Error()
		_ = client.Status().Update(ctx, sub)
		return reconcile.Result{}, err
	}

	materialURL, err := materialURL(resp.ID)
	if err != nil {
		sub.Status.Phase = phaseFailed
		sub.Status.Message = err.Error()
		_ = client.Status().Update(ctx, sub)
		return reconcile.Result{}, err
	}

	sub.Status.Phase = phaseReady
	sub.Status.MaterialID = resp.ID
	sub.Status.MaterialURL = materialURL
	sub.Status.SubmittedAt = time.Now().UTC().Format(time.RFC3339)
	sub.Status.Message = ""
	log.Info("Submitted material", "materialId", resp.ID)

	if err := client.Status().Update(ctx, sub); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update materialsubmission status: %w", err)
	}

	return reconcile.Result{}, nil
}

// buildMaterialRequest translates a MaterialSubmissionData (the portal's
// copy of an SKZ Material's spec, field-for-field typed) into the
// StrömungsRaum Platform API's sections.{temperature,viscosity,heat,
// injection_molding,pvt} shape, each a {method, parameters} pair - the SKZ
// CRD groups data by model (e.g. tempModel.c1), the backend groups it by
// generic method/parameters, so this mapping is the one place that
// knowledge lives (see ianus-simod's MaterialController::SECTION_TO_COLUMN_PREFIX
// for the backend side of this same mapping).
func buildMaterialRequest(m *materialsv1alpha1.MaterialSubmissionData) simod.MaterialRequest {
	req := simod.MaterialRequest{
		Name: m.Name,
		Sections: simod.MaterialSections{
			Data: map[string]any{
				"type":             m.Data.Type,
				"norm":             m.Data.Norm,
				"material_number":  m.Data.MaterialNumber,
				"density":          m.Data.Density,
				"heatconductivity": m.Data.HeatConductivity,
				"heatcapacity":     m.Data.HeatCapacity,
			},
			Temperature: &simod.MaterialSection{
				Method: m.TempModel.Model,
				Parameters: map[string]any{
					"c1":                   m.TempModel.C1,
					"c2":                   m.TempModel.C2,
					"D3":                   m.TempModel.D3,
					"referenceTemperature": m.TempModel.ReferenceTemperature,
				},
			},
			Viscosity: &simod.MaterialSection{
				Method: m.ViscoModel.Model,
				Parameters: map[string]any{
					"exponent":              m.ViscoModel.Exponent,
					"zero_viscosity":        m.ViscoModel.ZeroViscosity,
					"critical_stress_level": m.ViscoModel.CriticalStressLevel,
				},
			},
			Heat: &simod.MaterialSection{
				Method:     m.HeatModel.Model,
				Parameters: buildThermalData(m.HeatModel.Thermaldata),
			},
			InjectionMolding: &simod.MaterialSection{
				Parameters: map[string]any{
					"structure":            m.Structure,
					"min_melt_temp":        m.MinMeltTemp,
					"max_melt_temp":        m.MaxMeltTemp,
					"abs_max_melt_temp":    m.AbsMaxMeltTemp,
					"eject_temp":           m.EjectTemp,
					"freeze_temp":          m.FreezeTemp,
					"melt_density":         m.MeltDensity,
					"solid_density":        m.SolidDensity,
					"mech_properties":      m.MechProperties,
					"shrinkage_properties": m.ShrinkageProperties,
					"max_shear_stress":     m.MaxShearStress,
					"max_shear_rate":       m.MaxShearRate,
				},
			},
			PVT: &simod.MaterialSection{
				Method: m.PVTModel.Model,
				Parameters: map[string]any{
					"b1s": m.PVTModel.B1s,
					"b2s": m.PVTModel.B2s,
					"b3s": m.PVTModel.B3s,
					"b4s": m.PVTModel.B4s,
					"b1m": m.PVTModel.B1m,
					"b2m": m.PVTModel.B2m,
					"b3m": m.PVTModel.B3m,
					"b4m": m.PVTModel.B4m,
					"b5":  m.PVTModel.B5,
					"b6":  m.PVTModel.B6,
					"b7":  m.PVTModel.B7,
					"b8":  m.PVTModel.B8,
					"b9":  m.PVTModel.B9,
				},
			},
		},
	}
	if m.RawJSON != "" {
		req.Temporary = json.RawMessage(m.RawJSON)
	}
	return req
}

// buildThermalData converts HeatModel.Thermaldata rows into the shape
// sections.heat.parameters expects - a list, not a flat map, since it's
// temperature-indexed data (see pkg/simod.MaterialSection.Parameters' doc).
func buildThermalData(rows []materialsv1alpha1.ThermalDataPoint) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"temperature":      row.Temperature,
			"heatcapacity":     row.HeatCapacity,
			"heatconductivity": row.HeatConductivity,
			"density":          row.Density,
		})
	}
	return out
}
