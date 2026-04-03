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

// Package scaffolds provides the template rendering engine for decoupled-go/v1.
//
// It exposes typed data structures (InitData, APIData, WebhookData) and
// rendering functions (RenderInit, RenderAPIPlatform, etc.) that the command
// handlers in the parent package use to produce the file universe.
//
// All templates use Go's text/template package. Template names match the
// relative file paths they produce.
package scaffolds
