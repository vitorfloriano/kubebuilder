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

// handleCreateWebhook processes the "create webhook" command.
//
// Platform-layer webhook files are always regenerated.
// App-layer webhook hook stubs are created only if they don't already exist.
func handleCreateWebhook(req *external.PluginRequest, resp *external.PluginResponse) error {
	group := flagValue(req.Args, "--group", "")
	version := flagValue(req.Args, "--version", "")
	kind := flagValue(req.Args, "--kind", "")
	defaulting := flagBool(req.Args, "--defaulting")
	validating := flagBool(req.Args, "--validating")
	programmatic := flagBool(req.Args, "--programmatic-validation")

	if err := validateGVK(group, version, kind); err != nil {
		return err
	}

	if !defaulting && !validating && !programmatic {
		return fmt.Errorf("at least one of --defaulting, --validating, or --programmatic-validation must be set")
	}

	data := scaffolds.WebhookData{
		Group:                  group,
		Version:                version,
		Kind:                   kind,
		Defaulting:             defaulting,
		Validating:             validating,
		ProgrammaticValidation: programmatic,
		PluginKey:              PluginKey,
		PluginVersion:          PluginVersion,
	}

	// Platform files — always overwrite.
	platformFiles, err := scaffolds.RenderWebhookPlatform(data)
	if err != nil {
		return fmt.Errorf("render platform webhook scaffold: %w", err)
	}
	for path, content := range platformFiles {
		resp.Universe[path] = content
	}

	// App stub files — create-only.
	appFiles, err := scaffolds.RenderWebhookApp(data)
	if err != nil {
		return fmt.Errorf("render app webhook scaffold: %w", err)
	}
	for path, content := range appFiles {
		if _, exists := resp.Universe[path]; !exists {
			resp.Universe[path] = content
		}
	}

	return nil
}
