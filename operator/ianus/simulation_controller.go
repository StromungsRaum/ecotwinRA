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

package ianus

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	ianusv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/ianus/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

// SimulationReconciler creates Simulation resources on the SIMOD backend.
type SimulationReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimulationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("simulation-controller").
		For(&ianusv1alpha1.Simulation{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of Simulation resources across clusters.
func (r *SimulationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	simulation := &ianusv1alpha1.Simulation{}
	if err := client.Get(ctx, req.NamespacedName, simulation); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get simulation: %w", err)
	}

	if simulation.Status.SimulationID != "" {
		return reconcile.Result{}, nil
	}

	digitalTwin := &ianusv1alpha1.DigitalTwin{}
	digitalTwinKey := types.NamespacedName{Namespace: simulation.Namespace, Name: simulation.Spec.DigitalTwinRef.Name}
	if err := client.Get(ctx, digitalTwinKey, digitalTwin); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get referenced digitaltwin %s: %w", digitalTwinKey, err)
	}

	executionParameter := &ianusv1alpha1.ExecutionParameter{}
	executionParameterKey := types.NamespacedName{Namespace: simulation.Namespace, Name: simulation.Spec.ExecutionParameterRef.Name}
	if err := client.Get(ctx, executionParameterKey, executionParameter); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get referenced executionparameter %s: %w", executionParameterKey, err)
	}

	if digitalTwin.Status.Phase != phaseReady || executionParameter.Status.Phase != phaseReady {
		simulation.Status.Phase = phasePending
		_ = client.Status().Update(ctx, simulation)
		log.Info("Waiting for digital twin and execution parameter to become ready",
			"digitalTwin", digitalTwinKey, "executionParameter", executionParameterKey)
		return reconcile.Result{RequeueAfter: dependencyRequeueAfter}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, simulation.Namespace, simulation.Spec.SecretRefs)
	if err != nil {
		simulation.Status.Phase = phaseFailed
		simulation.Status.Message = err.Error()
		_ = client.Status().Update(ctx, simulation)
		return reconcile.Result{}, err
	}

	resp, err := simodClient.CreateSimulation(ctx, token, simod.SimulationRequest{
		DigitalTwinID:        digitalTwin.Status.DigitalTwinID,
		ExecutionParameterID: executionParameter.Status.Token,
	})
	if err != nil {
		simulation.Status.Phase = phaseFailed
		simulation.Status.Message = err.Error()
		_ = client.Status().Update(ctx, simulation)
		return reconcile.Result{}, err
	}

	simulation.Status.Phase = phaseReady
	simulation.Status.SimulationID = resp.ID
	simulation.Status.SimulationStatus = resp.Status
	simulation.Status.Message = ""
	if err := client.Status().Update(ctx, simulation); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update simulation status: %w", err)
	}

	log.Info("Created simulation", "simulationId", resp.ID, "status", resp.Status)
	return reconcile.Result{}, nil
}
