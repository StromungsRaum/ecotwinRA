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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	ianusv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/ianus/v1alpha1"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("model-controller").
		For(&ianusv1alpha1.Model{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of Model resources across clusters.
func (r *ModelReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	// Retrieve the Model from the cluster.
	model := &ianusv1alpha1.Model{}
	if err := client.Get(ctx, req.NamespacedName, model); err != nil {
		if apierrors.IsNotFound(err) {
			// model was deleted.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get model: %w", err)
	}

	log.Info("Reconciling model", "name", model.Name, "namespace", model.Namespace, "intent", model.Spec.Intent)

	// Update status based on intent
	if model.Spec.Intent != "" && model.Status.Result == "" {
		model.Status.Result = fmt.Sprintf("Great! %s completed", model.Spec.Intent)
		if err := client.Status().Update(ctx, model); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update model status: %w", err)
		}
		log.Info("Updated model status", "result", model.Status.Result)
	}

	// Record an event
	recorder := cl.GetEventRecorderFor("model-controller")
	recorder.Eventf(model, corev1.EventTypeNormal, "Reconciled", "Model %s/%s reconciled", model.Namespace, model.Name)

	return reconcile.Result{}, nil
}
