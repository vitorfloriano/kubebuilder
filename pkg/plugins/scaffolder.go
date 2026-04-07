/*
Copyright 2020 The Kubernetes Authors.

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

package plugins

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// Scaffolder interface creates files to set up a controller manager
type Scaffolder interface {
	InjectFS(machinery.Filesystem)
	// Scaffold performs the scaffolding
	Scaffold() error
}

// ScaffolderHooks holds optional hook functions that are called before and after
// the main scaffold operation. Use ScaffolderOption functions to configure these hooks
// when constructing a Scaffolder, allowing custom code injection without altering
// the scaffold implementation.
type ScaffolderHooks struct {
	// PreScaffold is an optional function called before the main scaffold operation.
	// It receives the filesystem so it can read or write files before the scaffold runs.
	PreScaffold func(machinery.Filesystem) error
	// PostScaffold is an optional function called after the main scaffold operation.
	PostScaffold func() error
}

// ScaffolderOption is a function type for configuring ScaffolderHooks.
// It follows the functional options pattern, enabling custom code injection
// into scaffold operations without modifying the scaffold implementation itself.
type ScaffolderOption func(*ScaffolderHooks)

// WithPreScaffoldHook returns a ScaffolderOption that registers fn to be called
// before the main scaffold operation. Use this to inject custom setup or
// validation logic without altering the scaffold implementation.
func WithPreScaffoldHook(fn func(machinery.Filesystem) error) ScaffolderOption {
	return func(h *ScaffolderHooks) {
		h.PreScaffold = fn
	}
}

// WithPostScaffoldHook returns a ScaffolderOption that registers fn to be called
// after the main scaffold operation. Use this to inject custom cleanup or
// post-processing logic without altering the scaffold implementation.
func WithPostScaffoldHook(fn func() error) ScaffolderOption {
	return func(h *ScaffolderHooks) {
		h.PostScaffold = fn
	}
}
