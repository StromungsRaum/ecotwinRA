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
)

// MaterialCatalogReconciler browses StrömungsRaum's existing materials into
// status, HttpBin-style but singleton (the portal offers no "+" for this
// one - there is exactly one MaterialCatalog per account):
//   - spec.refreshRequestedAt, bumped by the UI's Refresh button, triggers a
//     fetch of the account's Materials into status.availableMaterials.
type MaterialCatalogReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *MaterialCatalogReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("materialcatalog-controller").
		For(&materialsv1alpha1.MaterialCatalog{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of MaterialCatalog resources across
// clusters.
func (r *MaterialCatalogReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	catalog := &materialsv1alpha1.MaterialCatalog{}
	if err := client.Get(ctx, req.NamespacedName, catalog); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get materialcatalog: %w", err)
	}

	needsRefresh := catalog.Spec.RefreshRequestedAt != catalog.Status.LastProcessedRefreshRequestedAt
	if !needsRefresh {
		catalog.Status.Phase = phaseReady
		if err := client.Status().Update(ctx, catalog); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update materialcatalog status: %w", err)
		}
		return reconcile.Result{}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, catalog.Namespace, catalog.Spec.SecretRefs)
	if err != nil {
		catalog.Status.Phase = phaseFailed
		catalog.Status.Message = err.Error()
		_ = client.Status().Update(ctx, catalog)
		return reconcile.Result{}, err
	}

	materials, err := simodClient.ListMaterials(ctx, token)
	if err != nil {
		catalog.Status.Phase = phaseFailed
		catalog.Status.Message = err.Error()
		_ = client.Status().Update(ctx, catalog)
		return reconcile.Result{}, err
	}

	summaries := make([]materialsv1alpha1.MaterialSummary, 0, len(materials))
	for _, m := range materials {
		summaries = append(summaries, materialsv1alpha1.MaterialSummary{
			ID:        m.ID,
			Name:      m.Name,
			CreatedAt: m.CreatedAt,
		})
	}

	catalog.Status.AvailableMaterials = summaries
	catalog.Status.LastRefreshedAt = time.Now().UTC().Format(time.RFC3339)
	catalog.Status.LastProcessedRefreshRequestedAt = catalog.Spec.RefreshRequestedAt
	catalog.Status.Phase = phaseReady
	catalog.Status.Message = ""
	log.Info("Refreshed available materials", "count", len(summaries))

	if err := client.Status().Update(ctx, catalog); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update materialcatalog status: %w", err)
	}

	return reconcile.Result{}, nil
}
