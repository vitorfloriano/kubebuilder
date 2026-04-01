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

package scaffolds_test

import (
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/decoupled-go/v1/scaffolds"
)

// TestRenderInit verifies that RenderInit produces all expected platform files
// with the correct provenance header and no template rendering errors.
func TestRenderInit(t *testing.T) {
	data := scaffolds.InitData{
		Domain:        "my.org",
		Repo:          "github.com/myorg/my-operator",
		ProjectName:   "my-operator",
		PluginKey:     "decoupled-go.example.com/v1",
		PluginVersion: "v1.0.0",
	}

	files, err := scaffolds.RenderInit(data)
	if err != nil {
		t.Fatalf("RenderInit failed: %v", err)
	}

	requiredPaths := []string{
		"internal/platform/doc.go",
		"internal/platform/manager/manager.go",
		"internal/app/doc.go",
		"go.mod",
		"Makefile",
		"Dockerfile",
		".gitignore",
	}

	for _, path := range requiredPaths {
		content, ok := files[path]
		if !ok {
			t.Errorf("expected file %q not found in output", path)
			continue
		}
		if content == "" {
			t.Errorf("file %q has empty content", path)
		}
	}

	// Platform files must have the DO NOT EDIT header.
	for _, path := range []string{"internal/platform/doc.go", "internal/platform/manager/manager.go"} {
		content := files[path]
		if !strings.Contains(content, "DO NOT EDIT") {
			t.Errorf("platform file %q missing 'DO NOT EDIT' header", path)
		}
		if !strings.Contains(content, "v1.0.0") {
			t.Errorf("platform file %q missing plugin version in header", path)
		}
	}

	// App doc.go must not say DO NOT EDIT (it's user-owned).
	if strings.Contains(files["internal/app/doc.go"], "DO NOT EDIT") {
		t.Error("app doc.go must not have 'DO NOT EDIT' header — it is user-owned")
	}

	// go.mod must contain the repo module path.
	if !strings.Contains(files["go.mod"], data.Repo) {
		t.Errorf("go.mod does not contain repo %q", data.Repo)
	}
}

// TestRenderAPIPlatform verifies that RenderAPIPlatform generates the correct
// platform-layer files for a given GVK.
func TestRenderAPIPlatform(t *testing.T) {
	data := scaffolds.APIData{
		Group:         "apps",
		Version:       "v1",
		Kind:          "MyApp",
		Namespaced:    true,
		PluginKey:     "decoupled-go.example.com/v1",
		PluginVersion: "v1.0.0",
	}

	files, err := scaffolds.RenderAPIPlatform(data)
	if err != nil {
		t.Fatalf("RenderAPIPlatform failed: %v", err)
	}

	requiredPaths := []string{
		"internal/platform/reconciler/myapp_reconciler.go",
		"internal/platform/reconciler/hooks.go",
		"internal/platform/reconciler/noop.go",
	}

	for _, path := range requiredPaths {
		content, ok := files[path]
		if !ok {
			t.Errorf("expected file %q not found", path)
			continue
		}
		if !strings.Contains(content, "DO NOT EDIT") {
			t.Errorf("platform file %q missing 'DO NOT EDIT' header", path)
		}
	}

	// Reconciler must reference the Kind.
	reconciler := files["internal/platform/reconciler/myapp_reconciler.go"]
	if !strings.Contains(reconciler, "MyApp") {
		t.Error("reconciler does not reference Kind 'MyApp'")
	}
	if !strings.Contains(reconciler, "hooks ReconcileHooks") {
		t.Error("reconciler does not inject hooks via interface")
	}

	// Noop must implement all hook methods.
	noop := files["internal/platform/reconciler/noop.go"]
	for _, method := range []string{"BeforeReconcile", "Reconcile", "AfterReconcile", "OnError"} {
		if !strings.Contains(noop, method) {
			t.Errorf("noop.go missing method %q", method)
		}
	}
}

// TestRenderAPIApp verifies that RenderAPIApp generates app-layer stubs
// without DO NOT EDIT headers.
func TestRenderAPIApp(t *testing.T) {
	data := scaffolds.APIData{
		Group:         "apps",
		Version:       "v1",
		Kind:          "MyApp",
		Namespaced:    true,
		PluginKey:     "decoupled-go.example.com/v1",
		PluginVersion: "v1.0.0",
	}

	files, err := scaffolds.RenderAPIApp(data)
	if err != nil {
		t.Fatalf("RenderAPIApp failed: %v", err)
	}

	hooksPath := "internal/app/myapp/hooks.go"
	content, ok := files[hooksPath]
	if !ok {
		t.Fatalf("expected file %q not found", hooksPath)
	}

	// App file must not have DO NOT EDIT.
	if strings.Contains(content, "DO NOT EDIT") {
		t.Error("app hooks.go must not have 'DO NOT EDIT' header")
	}

	// Must contain hook method stubs.
	for _, method := range []string{"BeforeReconcile", "Reconcile", "AfterReconcile", "OnError"} {
		if !strings.Contains(content, method) {
			t.Errorf("app hooks.go missing method stub %q", method)
		}
	}

	// Must be in the correct package.
	if !strings.Contains(content, "package myapp") {
		t.Error("app hooks.go has incorrect package name")
	}
}

// TestLockFileMarshalRoundtrip verifies that a lock file can be marshalled
// and the version can be parsed back correctly.
func TestLockFileMarshalRoundtrip(t *testing.T) {
	lf := scaffolds.NewLockFile("v1.2.3", "decoupled-go.example.com/v1")
	lf.AddFile("internal/platform/doc.go", "content", scaffolds.ZonePlatform, scaffolds.PolicyAlwaysOverwrite)
	lf.AddFile("internal/app/myapp/hooks.go", "user content", scaffolds.ZoneApp, scaffolds.PolicyCreateOnly)

	content, err := lf.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(content, "DO NOT EDIT") {
		t.Error("lock file missing 'DO NOT EDIT' header comment")
	}
	if !strings.Contains(content, "v1.2.3") {
		t.Error("lock file missing plugin version")
	}

	// Parse version back.
	version, err := scaffolds.ParseLockFileVersion(content)
	if err != nil {
		t.Fatalf("ParseLockFileVersion failed: %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("parsed version %q, want %q", version, "v1.2.3")
	}
}

// TestRenderAPIApp_KindLower verifies that KindLower() produces lowercase output.
func TestAPIData_KindLower(t *testing.T) {
	cases := []struct {
		kind string
		want string
	}{
		{"MyApp", "myapp"},
		{"CronJob", "cronjob"},
		{"Foo", "foo"},
	}
	for _, tc := range cases {
		d := scaffolds.APIData{Kind: tc.kind}
		if got := d.KindLower(); got != tc.want {
			t.Errorf("KindLower(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

// TestRenderInit_Determinism verifies that rendering twice with the same
// data produces byte-for-byte identical output.
func TestRenderInit_Determinism(t *testing.T) {
	data := scaffolds.InitData{
		Domain:        "example.com",
		Repo:          "github.com/test/proj",
		PluginKey:     "decoupled-go.example.com/v1",
		PluginVersion: "v1.0.0",
	}

	files1, err1 := scaffolds.RenderInit(data)
	files2, err2 := scaffolds.RenderInit(data)

	if err1 != nil || err2 != nil {
		t.Fatalf("RenderInit errors: %v, %v", err1, err2)
	}

	for path, content1 := range files1 {
		content2, ok := files2[path]
		if !ok {
			t.Errorf("second render missing file %q", path)
			continue
		}
		if content1 != content2 {
			t.Errorf("file %q is not deterministic:\nfirst:\n%s\nsecond:\n%s", path, content1, content2)
		}
	}
}
