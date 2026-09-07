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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	ianusv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/ianus/v1alpha1"
	"github.com/StromungsRaum/ecotwinRA/pkg/simod"
)

// ExecutionParameterReconciler mints ExecutionParameter tokens on the SIMOD
// backend. Unlike the other SIMOD resources this has no dependency on
// another local CR - the referenced Form lives only on the backend.
type ExecutionParameterReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutionParameterReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("executionparameter-controller").
		For(&ianusv1alpha1.ExecutionParameter{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of ExecutionParameter resources across clusters.
func (r *ExecutionParameterReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	executionParameter := &ianusv1alpha1.ExecutionParameter{}
	if err := client.Get(ctx, req.NamespacedName, executionParameter); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get executionparameter: %w", err)
	}

	if executionParameter.Status.Token != "" {
		return reconcile.Result{}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, executionParameter.Namespace, executionParameter.Spec.SecretRefs)
	if err != nil {
		executionParameter.Status.Phase = phaseFailed
		executionParameter.Status.Message = err.Error()
		_ = client.Status().Update(ctx, executionParameter)
		return reconcile.Result{}, err
	}

	resp, err := simodClient.CreateExecutionParameter(ctx, token, simod.ExecutionParameterRequest{
		FormID:     executionParameter.Spec.FormID,
		Parameters: executionParameter.Spec.Parameters,
	})
	if err != nil {
		executionParameter.Status.Phase = phaseFailed
		executionParameter.Status.Message = err.Error()
		_ = client.Status().Update(ctx, executionParameter)
		return reconcile.Result{}, err
	}

	executionParameter.Status.Phase = phaseReady
	executionParameter.Status.Token = resp.ID
	executionParameter.Status.Message = ""
	if err := client.Status().Update(ctx, executionParameter); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update executionparameter status: %w", err)
	}

	log.Info("Created execution parameter")
	return reconcile.Result{}, nil
}
