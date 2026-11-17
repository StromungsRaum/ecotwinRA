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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"

	skzv1alpha1 "github.com/StromungsRaum/ecotwinRA/apis/skz/v1alpha1"
	configcatalogskz "github.com/StromungsRaum/ecotwinRA/config/catalog-skz"
	configcrdsskz "github.com/StromungsRaum/ecotwinRA/config/crds-skz"
	configkcpskz "github.com/StromungsRaum/ecotwinRA/config/kcp-skz"
	configproviderskz "github.com/StromungsRaum/ecotwinRA/config/provider-skz"
)

// skzMaterialsEndpointSlice is the ClusterCachedResourceEndpointSlice
// (config/catalog-skz) the skz APIExport serves its materials from.
const skzMaterialsEndpointSlice = "materials"

// BootstrapSKZ creates all skz provider resources from embedded files.
//
// skz is a dummy/demo provider with no operator. Its Materials are a static
// catalog, identical for every consumer: they are created once here, in the
// provider workspace, and published read-only to consumers through kcp's
// cached resources (cache.kcp.io, needs the CacheAPIs feature gate) instead
// of each consumer creating its own copies. So, unlike the other providers,
// this bootstrap also installs the CRD and the catalog objects themselves:
//
//  1. config/crds-skz    - the Material CRD, so the provider workspace can hold Materials
//  2. config/catalog-skz - Materials (from materials/*.json) + ClusterCachedResource
//  3. config/kcp-skz     - APIResourceSchema + APIExport (materials switched to virtual storage)
//  4. config/catalog-skz - ClusterCachedResourceEndpointSlice (references the APIExport)
//  5. config/provider-skz - Platform Mesh provider registration
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

	logger.Info("Bootstrapping CRDs")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configcrdsskz.FS); err != nil {
		return fmt.Errorf("failed to bootstrap CRDs: %w", err)
	}

	logger.Info("Bootstrapping material catalog")
	if err := bootstrapSKZMaterials(ctx, dynamicClient, mapper, cache); err != nil {
		return fmt.Errorf("failed to bootstrap material catalog: %w", err)
	}

	logger.Info("Bootstrapping kcp resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configkcpskz.FS, skzVirtualMaterialsStorage); err != nil {
		return fmt.Errorf("failed to bootstrap kcp resources: %w", err)
	}

	logger.Info("Bootstrapping cached resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configcatalogskz.FS); err != nil {
		return fmt.Errorf("failed to bootstrap cached resources: %w", err)
	}

	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configproviderskz.FS); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}

	logger.Info("Bootstrap completed successfully")
	return nil
}

// skzVirtualMaterialsStorage switches the skz APIExport's materials resource
// from CRD storage to the cached-resource virtual storage. This can't live in
// config/kcp-skz/apiexport-*.yaml itself: apigen regenerates that file with
// `storage: {crd: {}}` on every `make generate-skz`.
func skzVirtualMaterialsStorage(u *unstructured.Unstructured) error {
	if u.GetKind() != "APIExport" {
		return nil
	}
	resources, found, err := unstructured.NestedSlice(u.Object, "spec", "resources")
	if err != nil || !found {
		return fmt.Errorf("APIExport has no spec.resources: %v", err)
	}
	patched := false
	for i := range resources {
		res, ok := resources[i].(map[string]interface{})
		if !ok || res["name"] != "materials" || res["group"] != skzv1alpha1.GroupVersion.Group {
			continue
		}
		res["storage"] = map[string]interface{}{
			"virtual": map[string]interface{}{
				"reference": map[string]interface{}{
					"apiGroup": "cache.kcp.io",
					"kind":     "ClusterCachedResourceEndpointSlice",
					"name":     skzMaterialsEndpointSlice,
				},
			},
		}
		patched = true
	}
	if !patched {
		return fmt.Errorf("APIExport does not export materials.%s", skzv1alpha1.GroupVersion.Group)
	}
	return unstructured.SetNestedSlice(u.Object, resources, "spec", "resources")
}

// bootstrapSKZMaterials creates one cluster-scoped Material per
// config/catalog-skz/materials/*.json, retrying until the freshly applied CRD
// is served.
func bootstrapSKZMaterials(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, cache discovery.CachedDiscoveryInterface) error {
	materials, err := skzCatalogMaterials()
	if err != nil {
		return err
	}

	logger := klog.FromContext(ctx)
	var lastErr error
	err = wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		var errs []error
		for _, m := range materials {
			if err := applyObject(ctx, client, mapper, m.DeepCopy()); err != nil {
				errs = append(errs, err)
			}
		}
		if lastErr = utilerrors.NewAggregate(errs); lastErr != nil {
			logger.Info("failed to bootstrap material catalog, retrying", "error", lastErr)
			cache.Invalidate()
			return false, nil
		}
		return true, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%w: %v", err, lastErr)
	}
	return err
}

// skzCatalogMaterials decodes the embedded material characterization files
// into Material objects. The object name is the file name without ".json";
// the file content is kept verbatim in spec.rawJson (it is what the materials
// provider forwards to the backend as-is).
func skzCatalogMaterials() ([]*unstructured.Unstructured, error) {
	const dir = "materials"
	entries, err := configcatalogskz.Materials.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded material catalog: %w", err)
	}

	var out []*unstructured.Unstructured
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := configcatalogskz.Materials.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", e.Name(), err)
		}

		material := &skzv1alpha1.Material{
			TypeMeta: metav1.TypeMeta{
				APIVersion: skzv1alpha1.GroupVersion.String(),
				Kind:       "Material",
			},
			ObjectMeta: metav1.ObjectMeta{Name: strings.TrimSuffix(e.Name(), ".json")},
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&material.Spec); err != nil {
			return nil, fmt.Errorf("failed to parse material %s: %w", e.Name(), err)
		}
		material.Spec.RawJSON = strings.TrimSpace(string(raw))

		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(material)
		if err != nil {
			return nil, fmt.Errorf("failed to convert material %s: %w", e.Name(), err)
		}
		out = append(out, &unstructured.Unstructured{Object: obj})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("embedded material catalog is empty")
	}
	return out, nil
}
