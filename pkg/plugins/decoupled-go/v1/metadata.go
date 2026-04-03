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

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// handleFlags returns the accepted flags for a given subcommand.
// The subcommand is identified by the first "--<name>" entry in req.Args.
func handleFlags(req *external.PluginRequest, resp *external.PluginResponse) error {
	for _, arg := range req.Args {
		switch arg {
		case "--init":
			resp.Flags = initFlags()
			return nil
		case "--create api":
			resp.Flags = createAPIFlags()
			return nil
		case "--create webhook":
			resp.Flags = createWebhookFlags()
			return nil
		case "--edit":
			resp.Flags = editFlags()
			return nil
		}
	}
	return nil
}

// handleMetadata returns human-readable metadata for a given subcommand.
func handleMetadata(req *external.PluginRequest, resp *external.PluginResponse) error {
	for _, arg := range req.Args {
		switch arg {
		case "--init":
			resp.Metadata = plugin.SubcommandMetadata{
				Description: initDescription,
				Examples:    initExamples,
			}
			return nil
		case "--create api":
			resp.Metadata = plugin.SubcommandMetadata{
				Description: createAPIDescription,
				Examples:    createAPIExamples,
			}
			return nil
		case "--create webhook":
			resp.Metadata = plugin.SubcommandMetadata{
				Description: createWebhookDescription,
				Examples:    createWebhookExamples,
			}
			return nil
		case "--edit":
			resp.Metadata = plugin.SubcommandMetadata{
				Description: editDescription,
				Examples:    editExamples,
			}
			return nil
		}
	}
	return nil
}

// initFlags returns the flags accepted by the init subcommand.
func initFlags() []external.Flag {
	return []external.Flag{
		{
			Name:    "domain",
			Type:    "string",
			Default: "example.com",
			Usage:   "Domain for the operator's API groups (e.g. my.org)",
		},
		{
			Name:    "repo",
			Type:    "string",
			Default: "",
			Usage:   "Go module path for the generated project (required, e.g. github.com/myorg/my-operator)",
		},
		{
			Name:    "project-name",
			Type:    "string",
			Default: "",
			Usage:   "Project name (defaults to the current directory name)",
		},
	}
}

// createAPIFlags returns the flags accepted by the create api subcommand.
func createAPIFlags() []external.Flag {
	return []external.Flag{
		{
			Name:    "namespaced",
			Type:    "bool",
			Default: "true",
			Usage:   "Resource is namespace-scoped (false = cluster-scoped)",
		},
		{
			Name:    "force",
			Type:    "bool",
			Default: "false",
			Usage:   "Overwrite app-layer stub even if it already exists",
		},
	}
}

// createWebhookFlags returns the flags accepted by the create webhook subcommand.
func createWebhookFlags() []external.Flag {
	return []external.Flag{
		{
			Name:    "defaulting",
			Type:    "bool",
			Default: "false",
			Usage:   "Scaffold a mutating (defaulting) webhook",
		},
		{
			Name:    "validating",
			Type:    "bool",
			Default: "false",
			Usage:   "Scaffold a validating webhook",
		},
		{
			Name:    "programmatic-validation",
			Type:    "bool",
			Default: "false",
			Usage:   "Scaffold a Go-based custom validator (alternative to CEL)",
		},
	}
}

// editFlags returns the flags accepted by the edit subcommand.
func editFlags() []external.Flag {
	return []external.Flag{
		{
			Name:    "regenerate",
			Type:    "bool",
			Default: "false",
			Usage:   "Regenerate all platform-layer files (app-layer files are never touched)",
		},
	}
}

// Flag descriptions and examples for metadata responses.

const initDescription = `Initialize a new decoupled Go operator project.

decoupled-go/v1 scaffolds a project with a strict platform/app separation:

  internal/platform/   ← generated framework code (plugin-owned, always regenerated)
  internal/app/        ← your business logic (user-owned, never overwritten)

Upgrades are as simple as bumping the plugin version and running:
  kubebuilder edit --plugins decoupled-go.example.com/v1 --regenerate`

const initExamples = `  # Initialize a new project
  kubebuilder init \
    --plugins decoupled-go.example.com/v1 \
    --domain my.org \
    --repo github.com/myorg/my-operator

  # Initialize with kustomize/v2 for manifest generation
  kubebuilder init \
    --plugins "decoupled-go.example.com/v1,kustomize.common.kubebuilder.io/v2" \
    --domain my.org \
    --repo github.com/myorg/my-operator`

const createAPIDescription = `Scaffold a new API (CRD + controller) with decoupled business logic hooks.

Generates:
  internal/platform/reconciler/<kind>_reconciler.go  (regenerated on upgrade)
  internal/platform/reconciler/hooks.go              (interface definitions)
  internal/app/<kind>/hooks.go                       (your implementation — created once)`

const createAPIExamples = `  kubebuilder create api \
    --group apps --version v1 --kind MyApp

  # Cluster-scoped resource
  kubebuilder create api \
    --group apps --version v1 --kind MyClusterApp --namespaced=false`

const createWebhookDescription = `Scaffold a new webhook with decoupled validation/defaulting hooks.`

const createWebhookExamples = `  # Validating webhook
  kubebuilder create webhook \
    --group apps --version v1 --kind MyApp --validating

  # Defaulting + validating
  kubebuilder create webhook \
    --group apps --version v1 --kind MyApp --defaulting --validating`

const editDescription = `Edit or regenerate the platform layer of a decoupled-go/v1 project.

Use --regenerate to refresh all platform-layer files with the latest templates.
App-layer files (internal/app/) are never touched.`

const editExamples = `  # Regenerate after upgrading the plugin binary
  kubebuilder edit --plugins decoupled-go.example.com/v1 --regenerate`

// validateGVK validates that group, version, and kind are non-empty.
func validateGVK(group, version, kind string) error {
	if group == "" {
		return fmt.Errorf("--group is required")
	}
	if version == "" {
		return fmt.Errorf("--version is required")
	}
	if kind == "" {
		return fmt.Errorf("--kind is required")
	}
	return nil
}
