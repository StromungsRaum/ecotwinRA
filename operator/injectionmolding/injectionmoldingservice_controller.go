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

package injectionmolding

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

	injectionmoldingv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/injectionmolding/v1alpha1"
)

// simulationPollInterval is how often a monitored, not-yet-complete
// simulation's status is re-fetched.
const simulationPollInterval = 5 * time.Second

// InjectionMoldingServiceReconciler monitors a single SIMOD Simulation per
// resource, HttpBin-style (the portal creates one resource per card via its
// "+" button, not one shared resource per account):
//   - spec.refreshRequestedAt, bumped by the UI's Refresh button, triggers a
//     fetch of the account's Simulations into status.availableSimulations,
//     populating the portal's selection dropdown.
//   - spec.simulationId, set once the user picks from that dropdown, starts
//     polling that simulation's status every 5 seconds until it completes.
type InjectionMoldingServiceReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *InjectionMoldingServiceReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("injectionmoldingservice-controller").
		For(&injectionmoldingv1alpha1.InjectionMoldingService{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of InjectionMoldingService resources
// across clusters.
func (r *InjectionMoldingServiceReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	svc := &injectionmoldingv1alpha1.InjectionMoldingService{}
	if err := client.Get(ctx, req.NamespacedName, svc); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get injectionmoldingservice: %w", err)
	}

	url, err := modelsURL()
	if err != nil {
		svc.Status.Phase = phaseFailed
		svc.Status.Message = err.Error()
		_ = client.Status().Update(ctx, svc)
		return reconcile.Result{}, err
	}
	svc.Status.URL = url

	needsRefresh := svc.Spec.RefreshRequestedAt != svc.Status.LastProcessedRefreshRequestedAt
	monitoring := svc.Spec.SimulationID != ""
	alreadyComplete := svc.Status.MonitoredSimulation != nil && svc.Status.MonitoredSimulation.Complete

	if !needsRefresh && (!monitoring || alreadyComplete) {
		// Nothing to do: status.url is already current (cheap to always set),
		// and there's no pending refresh request or active polling.
		svc.Status.Phase = phaseReady
		if err := client.Status().Update(ctx, svc); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update injectionmoldingservice status: %w", err)
		}
		return reconcile.Result{}, nil
	}

	simodClient, token, err := loginSIMOD(ctx, client, svc.Namespace, svc.Spec.SecretRefs)
	if err != nil {
		svc.Status.Phase = phaseFailed
		svc.Status.Message = err.Error()
		_ = client.Status().Update(ctx, svc)
		return reconcile.Result{}, err
	}

	if needsRefresh {
		simulations, err := simodClient.ListSimulations(ctx, token)
		if err != nil {
			svc.Status.Phase = phaseFailed
			svc.Status.Message = err.Error()
			_ = client.Status().Update(ctx, svc)
			return reconcile.Result{}, err
		}

		summaries := make([]injectionmoldingv1alpha1.SimulationSummary, 0, len(simulations))
		for _, sim := range simulations {
			summaries = append(summaries, injectionmoldingv1alpha1.SimulationSummary{
				ID:     sim.ID,
				Name:   sim.Name,
				Status: sim.Status,
			})
		}

		svc.Status.AvailableSimulations = summaries
		svc.Status.LastRefreshedAt = time.Now().UTC().Format(time.RFC3339)
		svc.Status.LastProcessedRefreshRequestedAt = svc.Spec.RefreshRequestedAt
		log.Info("Refreshed available simulations", "count", len(summaries))
	}

	requeueAfter := time.Duration(0)
	if monitoring && !alreadyComplete {
		sim, err := simodClient.GetSimulation(ctx, token, svc.Spec.SimulationID)
		if err != nil {
			svc.Status.Phase = phaseFailed
			svc.Status.Message = err.Error()
			_ = client.Status().Update(ctx, svc)
			return reconcile.Result{}, err
		}

		complete := sim.Status == "complete"
		monitored := &injectionmoldingv1alpha1.MonitoredSimulationStatus{
			ID:           sim.ID,
			Name:         sim.Name,
			Status:       sim.Status,
			Complete:     complete,
			LastPolledAt: time.Now().UTC().Format(time.RFC3339),
		}
		if complete {
			// The Platform API's per-file download endpoint needs the same
			// Bearer token the operator uses, which a browser link can't
			// supply - point at the SIMOD web UI instead, where the user is
			// already authenticated via session cookie.
			resultsURL, err := simulationURL(sim.ID)
			if err != nil {
				svc.Status.Phase = phaseFailed
				svc.Status.Message = err.Error()
				_ = client.Status().Update(ctx, svc)
				return reconcile.Result{}, err
			}
			monitored.ResultsURL = resultsURL
		}
		svc.Status.MonitoredSimulation = monitored
		log.Info("Polled monitored simulation", "simulationId", sim.ID, "status", sim.Status, "complete", complete)

		if !complete {
			requeueAfter = simulationPollInterval
		}
	}

	svc.Status.Phase = phaseReady
	svc.Status.Message = ""
	if err := client.Status().Update(ctx, svc); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update injectionmoldingservice status: %w", err)
	}

	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}
