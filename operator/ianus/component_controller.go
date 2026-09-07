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

// ComponentReconciler creates Component resources on the SIMOD backend.
type ComponentReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComponentReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("component-controller").
		For(&ianusv1alpha1.Component{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of Component resources across clusters.
func (r *ComponentReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	component := &ianusv1alpha1.Component{}
	if err := client.Get(ctx, req.NamespacedName, component); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get component: %w", err)
	}

	// The backend has no update/delete for components - creation is the only
	// mutation, so once it succeeds there is nothing left to reconcile.
	if component.Status.ComponentID != "" {
		return reconcile.Result{}, nil
	}

	// Resolve child component references to their backend ids. Every slot
	// must point at a Component that has already reached Ready.
	backendComponents := map[string]string{}
	for slot, ref := range component.Spec.Components {
		child := &ianusv1alpha1.Component{}
		childKey := types.NamespacedName{Namespace: component.Namespace, Name: ref.Name}
		if err := client.Get(ctx, childKey, child); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to get referenced component %s: %w", childKey, err)
		}
		if child.Status.Phase != phaseReady {
			component.Status.Phase = phasePending
			_ = client.Status().Update(ctx, component)
			log.Info("Waiting for child component to become ready", "slot", slot, "component", childKey)
			return reconcile.Result{RequeueAfter: dependencyRequeueAfter}, nil
		}
		backendComponents[slot] = child.Status.ComponentID
	}

	simodClient, token, err := loginSIMOD(ctx, client, component.Namespace, component.Spec.SecretRefs)
	if err != nil {
		component.Status.Phase = phaseFailed
		component.Status.Message = err.Error()
		_ = client.Status().Update(ctx, component)
		return reconcile.Result{}, err
	}

	resp, err := simodClient.CreateComponent(ctx, token, simod.ComponentRequest{
		Name:          component.Spec.Name,
		ComponentType: component.Spec.ComponentType,
		Parameters:    component.Spec.Parameters,
		Components:    backendComponents,
	})
	if err != nil {
		component.Status.Phase = phaseFailed
		component.Status.Message = err.Error()
		_ = client.Status().Update(ctx, component)
		return reconcile.Result{}, err
	}

	component.Status.Phase = phaseReady
	component.Status.ComponentID = resp.ID
	component.Status.Message = ""
	if err := client.Status().Update(ctx, component); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update component status: %w", err)
	}

	log.Info("Created component", "componentId", resp.ID)
	return reconcile.Result{}, nil
}
