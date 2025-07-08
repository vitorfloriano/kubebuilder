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

package alphaupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	fromVersion            = "v4.5.2"
	customAPIMarker        = "// CUSTOM_API_CODE: This is custom API code"
	customControllerMarker = "// CUSTOM_CONTROLLER_CODE: This is custom controller code"
)

var _ = Describe("kubebuilder alpha update", func() {
	var (
		kbc           *utils.TestContext
		oldBinaryPath string
	)

	BeforeEach(func() {
		By("downloading old kubebuilder version")
		var err error
		oldBinaryPath, err = utils.DownloadKubebuilderBinary(fromVersion)
		Expect(err).NotTo(HaveOccurred())

		By("setting up test context")
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		if kbc != nil {
			kbc.Destroy()
		}
		if oldBinaryPath != "" {
			_ = utils.CleanupBinary(oldBinaryPath) // Fix errcheck: explicitly ignore error
		}
	})

	It("should upgrade a basic project preserving custom code", func() {
		By("initializing git repository")
		initGitRepo(kbc)

		By("creating project with old kubebuilder version")
		createProjectWithOldVersion(kbc, oldBinaryPath)

		By("adding custom code to generated files")
		addCustomCode(kbc)

		By("committing initial state")
		commitChanges(kbc, "Initial project with custom code")

		By("running alpha update command")
		Expect(kbc.AlphaUpdate("--from-version", fromVersion)).To(Succeed())

		By("verifying custom code is preserved")
		verifyCustomCodePreserved(kbc)
	})
})

// Helper functions - keep them simple for now
func addCustomCode(kbc *utils.TestContext) {
	// Add custom code to API types
	apiFile := filepath.Join(kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind)))

	content, err := os.ReadFile(apiFile)
	Expect(err).NotTo(HaveOccurred())

	// Look for the specific struct definition line
	structPattern := fmt.Sprintf("type %sSpec struct {", kbc.Kind)
	if !strings.Contains(string(content), structPattern) {
		Fail(fmt.Sprintf("Could not find struct definition '%s' in API file", structPattern))
	}

	newContent := strings.Replace(string(content), structPattern,
		fmt.Sprintf("%s\n%s", customAPIMarker, structPattern), 1)

	Expect(os.WriteFile(apiFile, []byte(newContent), 0o644)).To(Succeed())

	// Add custom code to controller
	controllerFile := filepath.Join(kbc.Dir, "internal", "controller",
		fmt.Sprintf("%s_controller.go", strings.ToLower(kbc.Kind)))

	content, err = os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())

	// Look for the Reconcile function specifically
	reconcilePattern := fmt.Sprintf("func (r *%sReconciler) Reconcile(", kbc.Kind)
	if !strings.Contains(string(content), reconcilePattern) {
		Fail(fmt.Sprintf("Could not find Reconcile function '%s' in controller file", reconcilePattern))
	}

	newContent = strings.Replace(string(content), reconcilePattern,
		fmt.Sprintf("%s\n%s", customControllerMarker, reconcilePattern), 1)

	Expect(os.WriteFile(controllerFile, []byte(newContent), 0o644)).To(Succeed())
}

func createProjectWithOldVersion(kbc *utils.TestContext, oldBinaryPath string) {
	// Temporarily switch binary
	originalBinary := kbc.BinaryName
	kbc.BinaryName = oldBinaryPath
	defer func() { kbc.BinaryName = originalBinary }()

	// Fix ginkgolinter: Use simpler version check approach
	cmd := exec.Command(oldBinaryPath, "version")
	cmd.Dir = kbc.Dir
	cmd.Env = kbc.Env
	_, err := kbc.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	Expect(kbc.Init(
		"--domain", kbc.Domain,
		"--repo", fmt.Sprintf("github.com/example/%s", kbc.TestSuffix),
	)).To(Succeed())

	Expect(kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--resource",
		"--controller",
	)).To(Succeed())

	Expect(kbc.Tidy()).To(Succeed())
}

// Improved git operations
func initGitRepo(kbc *utils.TestContext) {
	git := utils.NewGitHelper(kbc.Dir, kbc.Env)
	Expect(git.Init()).To(Succeed())
	Expect(git.ConfigUser("Test User", "test@example.com")).To(Succeed())

	// Check if we need to create main branch (newer git versions default to main)
	if currentBranch, err := git.GetCurrentBranch(); err != nil || currentBranch != "main" {
		Expect(git.CheckoutNewBranch("main")).To(Succeed())
	}
}

func commitChanges(kbc *utils.TestContext, message string) {
	git := utils.NewGitHelper(kbc.Dir, kbc.Env)
	Expect(git.Add(".")).To(Succeed())
	Expect(git.Commit(message)).To(Succeed())
}

func verifyCustomCodePreserved(kbc *utils.TestContext) {
	// Simple file content checks - no kubectl needed!
	apiFile := filepath.Join(kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind)))

	content, err := os.ReadFile(apiFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring(customAPIMarker))

	controllerFile := filepath.Join(kbc.Dir, "internal", "controller",
		fmt.Sprintf("%s_controller.go", strings.ToLower(kbc.Kind)))

	content, err = os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring(customControllerMarker))
}
