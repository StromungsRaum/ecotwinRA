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

package v1alpha1

// LocalObjectReference references another IANUS resource by name in the same
// namespace as the resource that holds the reference.
type LocalObjectReference struct {
	// Name of the referenced resource
	Name string `json:"name"`
}

// SecretReference references a Secret by name in the same namespace as the
// resource that holds the reference.
type SecretReference struct {
	// Name of the referenced Secret
	Name string `json:"name"`
}
