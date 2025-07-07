/*
Copyright 2025 The Kubernetes Authors.

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

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"

	. "github.com/onsi/gomega"
)

// ProjectValidator provides validation utilities for PROJECT files and project structure
type ProjectValidator struct {
	projectPath string
	config      config.Config
}

// NewProjectValidator creates a new PROJECT file validator
func NewProjectValidator(projectPath string) (*ProjectValidator, error) {
	cfg, err := loadConfigFromProjectFile(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}

	return &ProjectValidator{
		projectPath: projectPath,
		config:      cfg,
	}, nil
}

// ValidateBasicProjectStructure validates basic project structure after update
func (v *ProjectValidator) ValidateBasicProjectStructure(kbc *TestContext) {
	ExpectWithOffset(1, v.config.GetDomain()).To(Equal(kbc.Domain), "Domain should be preserved")
	ExpectWithOffset(1, v.config.GetRepository()).To(ContainSubstring(kbc.TestSuffix), "Repository should contain test suffix")
	ExpectWithOffset(1, v.config.GetPluginChain()).To(ContainElement("go.kubebuilder.io/v4"), "Should have Go v4 plugin")
}

// ValidateResourcePreservation validates that all resources are preserved after update
func (v *ProjectValidator) ValidateResourcePreservation(expectedResources []resource.Resource) {
	for _, expectedResource := range expectedResources {
		gvk := expectedResource.GVK
		ExpectWithOffset(1, v.config.HasResource(gvk)).To(BeTrue(),
			fmt.Sprintf("Resource %s/%s/%s should be preserved", gvk.Group, gvk.Version, gvk.Kind))

		actualResource, err := v.config.GetResource(gvk)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(),
			fmt.Sprintf("Should be able to retrieve resource %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))

		ExpectWithOffset(1, actualResource.Controller).To(Equal(expectedResource.Controller),
			fmt.Sprintf("Controller flag should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))

		if expectedResource.API != nil {
			ExpectWithOffset(1, actualResource.API).NotTo(BeNil(),
				fmt.Sprintf("API should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
			ExpectWithOffset(1, actualResource.API.Namespaced).To(Equal(expectedResource.API.Namespaced),
				fmt.Sprintf("Namespaced flag should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
		}

		if expectedResource.Webhooks != nil {
			ExpectWithOffset(1, actualResource.Webhooks).NotTo(BeNil(),
				fmt.Sprintf("Webhooks should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
			ExpectWithOffset(1, actualResource.Webhooks.Defaulting).To(Equal(expectedResource.Webhooks.Defaulting),
				fmt.Sprintf("Defaulting webhook should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
			ExpectWithOffset(1, actualResource.Webhooks.Validation).To(Equal(expectedResource.Webhooks.Validation),
				fmt.Sprintf("Validation webhook should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
			ExpectWithOffset(1, actualResource.Webhooks.Conversion).To(Equal(expectedResource.Webhooks.Conversion),
				fmt.Sprintf("Conversion webhook should be preserved for %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind))
		}
	}
}

// ValidateDeployImagePlugin validates that DeployImage plugin configuration is preserved
func (v *ProjectValidator) ValidateDeployImagePlugin(expectedConfigs []v1alpha1.ResourceData) {
	var deployImageConfig v1alpha1.PluginConfig
	err := v.config.DecodePluginConfig("deploy-image.go.kubebuilder.io/v1-alpha", &deployImageConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to decode DeployImage plugin config")

	ExpectWithOffset(1, deployImageConfig.Resources).ToNot(BeEmpty(),
		"Should have DeployImage resources configured")

	// Validate that expected resources exist in the config
	for _, expectedConfig := range expectedConfigs {
		found := false
		for _, actualConfig := range deployImageConfig.Resources {
			if actualConfig.Group == expectedConfig.Group &&
				actualConfig.Kind == expectedConfig.Kind &&
				actualConfig.Version == expectedConfig.Version {
				found = true
				break
			}
		}
		ExpectWithOffset(1, found).To(BeTrue(),
			fmt.Sprintf("Expected DeployImage resource %s/%s/%s should be found",
				expectedConfig.Group, expectedConfig.Version, expectedConfig.Kind))
	}
}

// ValidateGrafanaPlugin validates that Grafana plugin configuration is preserved
func (v *ProjectValidator) ValidateGrafanaPlugin() {
	var grafanaPluginConfig map[string]interface{}
	err := v.config.DecodePluginConfig("grafana.kubebuilder.io/v1-alpha", &grafanaPluginConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to decode Grafana plugin config")
	ExpectWithOffset(1, grafanaPluginConfig).NotTo(BeNil(), "Grafana plugin config should not be nil")
}

// ValidateHelmPlugin validates that Helm plugin configuration is preserved
func (v *ProjectValidator) ValidateHelmPlugin() {
	var helmPluginConfig map[string]interface{}
	err := v.config.DecodePluginConfig("helm.kubebuilder.io/v1-alpha", &helmPluginConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to decode Helm plugin config")
	ExpectWithOffset(1, helmPluginConfig).NotTo(BeNil(), "Helm plugin config should not be nil")
}

// ValidateCustomCodePreservation validates that custom code markers are preserved
func (v *ProjectValidator) ValidateCustomCodePreservation(kbc *TestContext, customMarkers map[string]string) {
	for filePath, marker := range customMarkers {
		fullPath := filepath.Join(filepath.Dir(v.projectPath), filePath)
		content, err := os.ReadFile(fullPath)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(),
			fmt.Sprintf("Should be able to read file %s", filePath))
		ExpectWithOffset(1, string(content)).To(ContainSubstring(marker),
			fmt.Sprintf("Custom code marker should be preserved in %s", filePath))
	}
}

// ValidateProjectBuilds validates that the project can be built successfully
func (v *ProjectValidator) ValidateProjectBuilds(kbc *TestContext) {
	ExpectWithOffset(1, kbc.Make("build")).To(Succeed(), "Project should build successfully")
	ExpectWithOffset(1, kbc.Make("manifests")).To(Succeed(), "Manifests should generate successfully")
	ExpectWithOffset(1, kbc.Make("generate")).To(Succeed(), "Code generation should succeed")
}

// ValidateUpdateOutcome validates the overall outcome of the alpha update command
func (v *ProjectValidator) ValidateUpdateOutcome(kbc *TestContext, git *GitHelper) {
	// Check if we're on the expected merge branch
	currentBranch, err := git.GetCurrentBranch()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to get current branch")
	ExpectWithOffset(1, currentBranch).To(ContainSubstring("tmp-merge-"),
		"Should be on merge branch after update")

	// Check if there are conflicts
	hasConflicts, err := git.HasConflicts()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to check for conflicts")

	if hasConflicts {
		// When conflicts exist, verify they were properly committed
		lastCommit, err := git.GetLastCommitMessage()
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should be able to get last commit")
		ExpectWithOffset(1, lastCommit).To(ContainSubstring("Merge from"),
			"Should have merge commit message")
	} else {
		// When no conflicts, project should build successfully
		v.ValidateProjectBuilds(kbc)
	}
}

// loadConfigFromProjectFile loads configuration from a PROJECT file
func loadConfigFromProjectFile(projectFilePath string) (config.Config, error) {
	fs := afero.NewOsFs()
	store := yaml.New(machinery.Filesystem{FS: fs})
	err := store.LoadFrom(projectFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load PROJECT configuration: %w", err)
	}

	return store.Config(), nil
}

// CodeInjector provides utilities for injecting custom code into generated files
type CodeInjector struct {
	projectDir string
}

// NewCodeInjector creates a new code injector for the specified project directory
func NewCodeInjector(projectDir string) *CodeInjector {
	return &CodeInjector{
		projectDir: projectDir,
	}
}

// InjectAPICode injects custom code into the API types file
func (c *CodeInjector) InjectAPICode(kbc *TestContext, customMarker string) error {
	apiFile := filepath.Join(c.projectDir, "api", kbc.Version,
		fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind)))
	return c.injectCodeIntoFile(apiFile, customMarker, "type "+kbc.Kind+"Spec struct {")
}

// InjectControllerCode injects custom code into the controller file
func (c *CodeInjector) InjectControllerCode(kbc *TestContext, customMarker string) error {
	controllerFile := filepath.Join(c.projectDir, "internal", "controller",
		fmt.Sprintf("%s_controller.go", strings.ToLower(kbc.Kind)))
	return c.injectCodeIntoFile(controllerFile, customMarker, "func (r *"+kbc.Kind+"Reconciler) Reconcile(")
}

// InjectWebhookCode injects custom code into the webhook file
func (c *CodeInjector) InjectWebhookCode(kbc *TestContext, customMarker string) error {
	webhookFile := filepath.Join(c.projectDir, "internal", "webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	return c.injectCodeIntoFile(webhookFile, customMarker, "func (r *"+kbc.Kind+") ValidateCreate() error {")
}

// injectCodeIntoFile injects custom code before a specific line
func (c *CodeInjector) injectCodeIntoFile(filePath, customCode, beforeLine string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		if strings.Contains(line, beforeLine) {
			newLines = append(newLines, customCode)
		}
		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
