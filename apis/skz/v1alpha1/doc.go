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

// Package v1alpha1 contains API Schema definitions for the genericmaterials
// v1alpha1 API group - the "Generic Materials Service" provider.
//
// The provider started out as "SKZ" (a demo stand-in for a material-testing
// institute) and was renamed to a vendor-neutral API. Only what users and API
// consumers see was renamed (API group, APIExport, portal path and labels);
// the Go package, config/*-skz directories and `make *-skz` targets keep the
// internal name "skz".
//
// Material is cluster-scoped on purpose: Materials are a static,
// provider-owned catalog, the same for every consumer. They exist once, in
// the provider workspace (created by pkg/bootstrap from
// config/catalog-skz/materials/*.json, which is also where spec.rawJson comes
// from), and are published read-only to consumers via a kcp
// ClusterCachedResource, which only supports cluster-scoped resources.
//
// This lives here rather than on the Material type because type and field
// comments end up in the generated APIResourceSchema, which kcp treats as
// immutable: rewording one changes the schema under an unchanged name (the
// name only encodes date + commit) and breaks re-bootstrap until the next
// commit.
// +kubebuilder:object:generate=true
// +groupName=genericmaterials.platform-mesh.io
package v1alpha1
