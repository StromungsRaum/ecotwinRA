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
// Package catalogskz embeds the skz provider's static Material catalog and
// the kcp cache.kcp.io resources that publish it read-only to consumers.
//
// materials/*.json are material characterization files (same shape as
// docs/assets/EastmanPlastics1000.json). pkg/bootstrap turns each one into a
// cluster-scoped Material in the provider workspace, named after the file.
// To add a material to every consumer's catalog, drop a JSON file here and
// re-run `make init-skz`.
//
// The YAML files are hand-written (not codegen output, which is why they are
// not in config/kcp-skz - apigen prunes that directory).
package catalogskz

import "embed"

// FS holds the ClusterCachedResource and its endpoint slice.
//
//go:embed *.yaml
var FS embed.FS

// Materials holds the material characterization JSON files.
//
//go:embed materials/*.json
var Materials embed.FS
