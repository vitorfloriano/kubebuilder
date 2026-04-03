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

// Package v1 contains the decoupled-go/v1 external plugin scaffold skeleton.
//
// decoupled-go/v1 is an external Kubebuilder plugin that scaffolds Go-based
// Kubernetes operators with a strict separation between generated framework
// code (platform layer, owned by the plugin) and user business logic (app
// layer, owned by the user).
//
// See the accompanying guide in docs/book/src/decoupled-go-plugin/ for
// a full learning path, plugin specification, and implementation plan.
//
// # Directory Structure
//
// This package contains:
//   - plugin.go    — plugin metadata and entrypoint wiring
//   - init.go      — scaffold for the `init` subcommand
//   - api.go       — scaffold for the `create api` subcommand
//   - webhook.go   — scaffold for the `create webhook` subcommand
//   - edit.go      — scaffold for the `edit` subcommand
//   - scaffolds/   — template data structures and rendering logic
//
// # Usage
//
// Build and install this plugin as an external Kubebuilder plugin:
//
//	go build -o decoupled-go-plugin ./cmd/
//	cp decoupled-go-plugin ~/.local/share/kubebuilder/plugins/decoupled-go.example.com/v1/
//
// Then use it:
//
//	kubebuilder init --plugins decoupled-go.example.com/v1 --domain my.org --repo github.com/myorg/my-op
//	kubebuilder create api --group apps --version v1 --kind MyApp
package v1
