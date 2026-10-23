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

package bootstrap

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"

	configcontrollerim "github.com/StromungsRaum/ecotwinRA/config/controller-injectionmolding"
	configkcpim "github.com/StromungsRaum/ecotwinRA/config/kcp-injectionmolding"
	configproviderim "github.com/StromungsRaum/ecotwinRA/config/provider-injectionmolding"
)

// BootstrapInjectionMolding creates all injectionmolding provider resources
// from embedded YAML files. Structurally identical to Bootstrap, but targets
// the injectionmolding provider's own kcp/provider/controller config sets and
// controller Secret names, so it can run against a separate kcp workspace
// alongside (not instead of) the ianus provider.
func BootstrapInjectionMolding(ctx context.Context, config *rest.Config, hostOverride string) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	cache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	logger := klog.FromContext(ctx)

	logger.Info("Bootstrapping kcp resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configkcpim.FS); err != nil {
		return fmt.Errorf("failed to bootstrap kcp resources: %w", err)
	}

	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configproviderim.FS); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}

	logger.Info("Bootstrapping controller resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configcontrollerim.FS); err != nil {
		return fmt.Errorf("failed to bootstrap controller resources: %w", err)
	}

	logger.Info("Creating controller kubeconfig secret")
	if err := createControllerKubeconfigSecret(ctx, kubeClient, config, hostOverride, "default", "injectionmolding-controller-token", "injectionmolding-controller-kubeconfig"); err != nil {
		return fmt.Errorf("failed to create controller kubeconfig secret: %w", err)
	}

	logger.Info("Bootstrap completed successfully")
	return nil
}
