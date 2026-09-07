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

// DigitalTwinReconciler creates DigitalTwin resources on the SIMOD backend.
type DigitalTwinReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *DigitalTwinReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("digitaltwin-controller").
		For(&ianusv1alpha1.DigitalTwin{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of DigitalTwin resources across clusters.
func (r *DigitalTwinReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	digitalTwin := &ianusv1alpha1.DigitalTwin{}
	if err := client.Get(ctx, req.NamespacedName, digitalTwin); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get digitaltwin: %w", err)
	}

	if digitalTwin.Status.DigitalTwinID != "" {
		return reconcile.Result{}, nil
	}

	component := &ianusv1alpha1.Component{}
	componentKey := types.NamespacedName{Namespace: digitalTwin.Namespace, Name: digitalTwin.Spec.ComponentRef.Name}
	if err := client.Get(ctx, componentKey, component); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get referenced component %s: %w", componentKey, err)
	}
	if component.Status.Phase != phaseReady {
		digitalTwin.Status.Phase = phasePending
		_ = client.Status().Update(ctx, digitalTwin)
		log.Info("Waiting for component to become ready", "component", componentKey)
		return reconcile.Result{RequeueAfter: dependencyRequeueAfter}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, digitalTwin.Namespace, digitalTwin.Spec.SecretRefs)
	if err != nil {
		digitalTwin.Status.Phase = phaseFailed
		digitalTwin.Status.Message = err.Error()
		_ = client.Status().Update(ctx, digitalTwin)
		return reconcile.Result{}, err
	}

	resp, err := simodClient.CreateDigitalTwin(ctx, token, simod.DigitalTwinRequest{
		Name:        digitalTwin.Spec.Name,
		ComponentID: component.Status.ComponentID,
	})
	if err != nil {
		digitalTwin.Status.Phase = phaseFailed
		digitalTwin.Status.Message = err.Error()
		_ = client.Status().Update(ctx, digitalTwin)
		return reconcile.Result{}, err
	}

	digitalTwin.Status.Phase = phaseReady
	digitalTwin.Status.DigitalTwinID = resp.ID
	digitalTwin.Status.Message = ""
	if err := client.Status().Update(ctx, digitalTwin); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update digitaltwin status: %w", err)
	}

	log.Info("Created digital twin", "digitalTwinId", resp.ID)
	return reconcile.Result{}, nil
}
