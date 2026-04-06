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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/decoupled-go/v1/scaffolds"
)

// handleInit processes the "init" command.
// It scaffolds the base project structure with platform and app layer stubs.
func handleInit(req *external.PluginRequest, resp *external.PluginResponse) error {
	domain := flagValue(req.Args, "--domain", "example.com")
	repo := flagValue(req.Args, "--repo", "")
	projectName := flagValue(req.Args, "--project-name", "")

	if repo == "" {
		return fmt.Errorf("--repo is required: provide the Go module path (e.g. github.com/myorg/my-operator)")
	}

	data := scaffolds.InitData{
		Domain:      domain,
		Repo:        repo,
		ProjectName: projectName,
		PluginKey:   PluginKey,
		PluginVersion: PluginVersion,
	}

	files, err := scaffolds.RenderInit(data)
	if err != nil {
		return fmt.Errorf("render init scaffold: %w", err)
	}

	for path, content := range files {
		resp.Universe[path] = content
	}

	// Write the scaffold lock file.
	lockContent, err := scaffolds.NewLockFile(PluginVersion, PluginKey).Marshal()
	if err != nil {
		return fmt.Errorf("marshal scaffold lock file: %w", err)
	}
	resp.Universe[lockFilePath()] = lockContent

	return nil
}

// lockFilePath returns the path of the scaffold lock file within the project.
func lockFilePath() string {
	return filepath.Join("gen", "kb", "decoupled-go.v1", ".kb-scaffold-lock.yaml")
}
