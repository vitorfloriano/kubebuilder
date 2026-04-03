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

package v1_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
	v1 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/decoupled-go/v1"
)

// runPlugin is a test helper that sends req to the plugin via Run and returns
// the decoded PluginResponse.
func runPlugin(t *testing.T, req external.PluginRequest) external.PluginResponse {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var out bytes.Buffer
	if err = v1.Run(bytes.NewReader(b), &out); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var resp external.PluginResponse
	if err = json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

// TestRun_Init verifies that the init command produces expected platform files.
func TestRun_Init(t *testing.T) {
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "init",
		Args:       []string{"--domain", "my.org", "--repo", "github.com/myorg/my-op"},
		Universe:   map[string]string{},
	}

	resp := runPlugin(t, req)

	if resp.Error {
		t.Fatalf("unexpected error: %v", resp.ErrorMsgs)
	}

	// Lock file must be present.
	lockPath := "gen/kb/decoupled-go.v1/.kb-scaffold-lock.yaml"
	if _, ok := resp.Universe[lockPath]; !ok {
		t.Error("lock file not found in universe")
	}

	// Platform files must be present.
	for _, path := range []string{
		"internal/platform/doc.go",
		"internal/platform/manager/manager.go",
		"go.mod",
		"Makefile",
		"Dockerfile",
	} {
		if _, ok := resp.Universe[path]; !ok {
			t.Errorf("expected file %q not in universe", path)
		}
	}
}

// TestRun_Init_MissingRepo verifies that init without --repo returns an error.
func TestRun_Init_MissingRepo(t *testing.T) {
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "init",
		Args:       []string{"--domain", "my.org"}, // no --repo
		Universe:   map[string]string{},
	}

	resp := runPlugin(t, req)

	if !resp.Error {
		t.Fatal("expected error response when --repo is missing, got success")
	}
	if len(resp.ErrorMsgs) == 0 {
		t.Fatal("expected non-empty error messages")
	}
}

// TestRun_CreateAPI verifies that create api produces platform and app files.
func TestRun_CreateAPI(t *testing.T) {
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Args:       []string{"--group", "apps", "--version", "v1", "--kind", "MyApp"},
		Universe:   map[string]string{},
	}

	resp := runPlugin(t, req)

	if resp.Error {
		t.Fatalf("unexpected error: %v", resp.ErrorMsgs)
	}

	// Platform files must be present and contain DO NOT EDIT.
	for _, path := range []string{
		"internal/platform/reconciler/myapp_reconciler.go",
		"internal/platform/reconciler/hooks.go",
		"internal/platform/reconciler/noop.go",
	} {
		content, ok := resp.Universe[path]
		if !ok {
			t.Errorf("platform file %q not in universe", path)
			continue
		}
		if !strings.Contains(content, "DO NOT EDIT") {
			t.Errorf("platform file %q missing 'DO NOT EDIT' header", path)
		}
	}

	// App stub must be present but must not contain DO NOT EDIT.
	appHooks, ok := resp.Universe["internal/app/myapp/hooks.go"]
	if !ok {
		t.Fatal("app hooks file not in universe")
	}
	if strings.Contains(appHooks, "DO NOT EDIT") {
		t.Error("app hooks.go must not have 'DO NOT EDIT' header")
	}
}

// TestRun_CreateAPI_PreservesAppFile verifies that create api does not
// overwrite an existing app-layer file.
func TestRun_CreateAPI_PreservesAppFile(t *testing.T) {
	userContent := "// my custom hooks — should not be overwritten\npackage myapp\n"
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Args:       []string{"--group", "apps", "--version", "v1", "--kind", "MyApp"},
		Universe: map[string]string{
			"internal/app/myapp/hooks.go": userContent,
		},
	}

	resp := runPlugin(t, req)

	if resp.Error {
		t.Fatalf("unexpected error: %v", resp.ErrorMsgs)
	}

	got := resp.Universe["internal/app/myapp/hooks.go"]
	if got != userContent {
		t.Errorf("app hooks.go was overwritten:\ngot: %q\nwant: %q", got, userContent)
	}
}

// TestRun_CreateAPI_Force verifies that --force overwrites existing app files.
func TestRun_CreateAPI_Force(t *testing.T) {
	userContent := "// user content\npackage myapp\n"
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Args:       []string{"--group", "apps", "--version", "v1", "--kind", "MyApp", "--force"},
		Universe: map[string]string{
			"internal/app/myapp/hooks.go": userContent,
		},
	}

	resp := runPlugin(t, req)

	if resp.Error {
		t.Fatalf("unexpected error: %v", resp.ErrorMsgs)
	}

	got := resp.Universe["internal/app/myapp/hooks.go"]
	if got == userContent {
		t.Error("expected app hooks.go to be overwritten with --force, but content is unchanged")
	}
}

// TestRun_CreateAPI_MissingGVK verifies that missing GVK flags return an error.
func TestRun_CreateAPI_MissingGVK(t *testing.T) {
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Args:       []string{"--group", "apps", "--version", "v1"}, // missing --kind
		Universe:   map[string]string{},
	}

	resp := runPlugin(t, req)

	if !resp.Error {
		t.Fatal("expected error when --kind is missing")
	}
}

// TestRun_Flags verifies that the flags command returns non-empty flag lists.
func TestRun_Flags(t *testing.T) {
	for _, subcmd := range []string{"--init", "--create api", "--create webhook", "--edit"} {
		req := external.PluginRequest{
			APIVersion: "v1alpha1",
			Command:    "flags",
			Args:       []string{subcmd},
			Universe:   map[string]string{},
		}
		resp := runPlugin(t, req)

		if resp.Error {
			t.Errorf("[%s] unexpected error: %v", subcmd, resp.ErrorMsgs)
		}
		if len(resp.Flags) == 0 {
			t.Errorf("[%s] expected non-empty flags list", subcmd)
		}
	}
}

// TestRun_Metadata verifies that the metadata command returns non-empty descriptions.
func TestRun_Metadata(t *testing.T) {
	for _, subcmd := range []string{"--init", "--create api", "--create webhook", "--edit"} {
		req := external.PluginRequest{
			APIVersion: "v1alpha1",
			Command:    "metadata",
			Args:       []string{subcmd},
			Universe:   map[string]string{},
		}
		resp := runPlugin(t, req)

		if resp.Error {
			t.Errorf("[%s] unexpected error: %v", subcmd, resp.ErrorMsgs)
		}
		if resp.Metadata.Description == "" {
			t.Errorf("[%s] expected non-empty description", subcmd)
		}
	}
}

// TestRun_UnknownCommand verifies that unknown commands pass the universe through.
func TestRun_UnknownCommand(t *testing.T) {
	original := map[string]string{"some/file.go": "content"}
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "some-unknown-command",
		Args:       []string{},
		Universe:   original,
	}

	resp := runPlugin(t, req)

	if resp.Error {
		t.Fatalf("unexpected error for unknown command: %v", resp.ErrorMsgs)
	}
	if got := resp.Universe["some/file.go"]; got != "content" {
		t.Errorf("universe modified for unknown command: got %q, want %q", got, "content")
	}
}

// TestFlagValue is a unit test for the internal flagValue helper
// exposed indirectly through Run behaviour.
func TestRun_CreateWebhook_MissingFlags(t *testing.T) {
	req := external.PluginRequest{
		APIVersion: "v1alpha1",
		Command:    "create webhook",
		// No --defaulting, --validating, or --programmatic-validation
		Args:     []string{"--group", "apps", "--version", "v1", "--kind", "MyApp"},
		Universe: map[string]string{},
	}

	resp := runPlugin(t, req)

	if !resp.Error {
		t.Fatal("expected error when no webhook type flags are set")
	}
}
