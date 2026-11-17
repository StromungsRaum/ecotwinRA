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
// Package crdsskz embeds the skz provider's generated CRD. Unlike the other
// providers' config/crds-* (codegen intermediates only), this one is applied
// to the provider workspace itself: the Material catalog objects live there,
// and a ClusterCachedResource replicates them to every consumer (see
// config/catalog-skz).
package crdsskz

import "embed"

//go:embed *.yaml
var FS embed.FS
