/*
Copyright 2024 The Kubernetes Authors.

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

package v1

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/decoupled-go/v1/scaffolds"
)

// handleCreateAPI processes the "create api" command.
//
// Platform-layer files (internal/platform/reconciler/) are always regenerated.
// App-layer stub files (internal/app/<kind>/) are created only if they do not
// already exist in the universe — preserving user edits across invocations.
func handleCreateAPI(req *external.PluginRequest, resp *external.PluginResponse) error {
	group := flagValue(req.Args, "--group", "")
	version := flagValue(req.Args, "--version", "")
	kind := flagValue(req.Args, "--kind", "")
	namespaced := !flagBool(req.Args, "--namespaced=false")
	force := flagBool(req.Args, "--force")

	if err := validateGVK(group, version, kind); err != nil {
		return err
	}

	data := scaffolds.APIData{
		Group:      group,
		Version:    version,
		Kind:       kind,
		Namespaced: namespaced,
		PluginKey:     PluginKey,
		PluginVersion: PluginVersion,
	}

	// Platform files — always overwrite (regenerated on every run).
	platformFiles, err := scaffolds.RenderAPIPlatform(data)
	if err != nil {
		return fmt.Errorf("render platform API scaffold: %w", err)
	}
	for path, content := range platformFiles {
		resp.Universe[path] = content
	}

	// App stub files — create-only unless --force is set.
	appFiles, err := scaffolds.RenderAPIApp(data)
	if err != nil {
		return fmt.Errorf("render app API scaffold: %w", err)
	}
	for path, content := range appFiles {
		if _, exists := resp.Universe[path]; !exists || force {
			resp.Universe[path] = content
		}
		// If the file already exists and --force is not set, skip it.
		// This preserves user business logic.
	}

	return nil
}
