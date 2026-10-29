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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"

	configkcpskz "github.com/StromungsRaum/ecotwinRA/config/kcp-skz"
	configproviderskz "github.com/StromungsRaum/ecotwinRA/config/provider-skz"
)

// BootstrapSKZ creates all skz provider resources from embedded YAML files.
// Unlike Bootstrap/BootstrapInjectionMolding, there is no controller set to
// bootstrap and no controller kubeconfig Secret to create - skz is a
// dummy/demo provider with no operator, just a CRD the portal writes to
// directly via the CRD Gateway.
func BootstrapSKZ(ctx context.Context, config *rest.Config, hostOverride string) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	cache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	logger := klog.FromContext(ctx)

	logger.Info("Bootstrapping kcp resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configkcpskz.FS); err != nil {
		return fmt.Errorf("failed to bootstrap kcp resources: %w", err)
	}

	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configproviderskz.FS); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}

	logger.Info("Bootstrap completed successfully")
	return nil
}
