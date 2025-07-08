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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	// Previous version to test migration from
	fromVersion = "v4.5.2"
	// Test runs are non-interactive, we need this env var to skip prompts
	nonInteractiveEnv = "KUBEBUILDER_NON_INTERACTIVE=true"
)

var _ = Describe("kubebuilder alpha update", func() {
	Context("basic upgrade scenarios", func() {
		var (
			kbc           *utils.TestContext
			oldBinaryPath string
			git           *utils.GitHelper
			injector      *utils.CodeInjector
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on", nonInteractiveEnv)
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			// Download and setup old version binary
			oldBinaryPath, err = utils.DownloadKubebuilderBinary(fromVersion)
			Expect(err).NotTo(HaveOccurred())

			// Initialize helpers
			git = utils.NewGitHelper(kbc.Dir, kbc.Env)
			injector = utils.NewCodeInjector(kbc.Dir)
		})

		AfterEach(func() {
			By("cleaning up downloaded binary")
			if oldBinaryPath != "" {
				err := utils.CleanupBinary(oldBinaryPath)
				if err != nil {
					// Log the error but don't fail the test during cleanup
					_, _ = fmt.Fprintf(GinkgoWriter, "Warning: failed to cleanup binary: %v\n", err)
				}
			}

			By("destroying test directory")
			kbc.Destroy()
		})

		It("should successfully upgrade project from previous version preserving custom code", func() {
			By("initializing git repository")
			initializeGitRepository(git)

			By("scaffolding project with old kubebuilder version")
			scaffoldProjectWithOldVersion(kbc, oldBinaryPath)

			By("creating API with old version")
			createAPIWithOldVersion(kbc, oldBinaryPath)

			By("injecting custom code into API and controller")
			injectAndCommitCustomCode(kbc, git, injector)

			By("running alpha update command with from-version flag")
			runAlphaUpdateWithFromVersion(kbc, fromVersion)

			By("verifying custom code is preserved")
			verifyCustomCodePreservation(kbc)

			By("verifying project state after update")
			verifyUpdateOutcome(kbc, git)
		})

		It("should successfully upgrade project using from-branch flag", func() {
			By("initializing git repository")
			initializeGitRepository(git)

			By("scaffolding project with old kubebuilder version")
			scaffoldProjectWithOldVersion(kbc, oldBinaryPath)

			By("creating API with old version")
			createAPIWithOldVersion(kbc, oldBinaryPath)

			By("committing initial project state")
			commitInitialState(git)

			By("creating feature branch")
			Expect(git.CheckoutNewBranch("feature-branch")).To(Succeed())

			By("injecting custom code in feature branch")
			injectAndCommitCustomCode(kbc, git, injector)

			By("running alpha update command with from-branch flag")
			runAlphaUpdateWithFromBranch(kbc, "feature-branch", fromVersion)

			By("verifying custom code is preserved")
			verifyCustomCodePreservation(kbc)

			By("verifying project state after update")
			verifyUpdateOutcome(kbc, git)
		})

		It("should successfully upgrade project with specific to-version flag", func() {
			By("initializing git repository")
			initializeGitRepository(git)

			By("scaffolding project with old kubebuilder version")
			scaffoldProjectWithOldVersion(kbc, oldBinaryPath)

			By("creating API with old version")
			createAPIWithOldVersion(kbc, oldBinaryPath)

			By("injecting custom code into API and controller")
			injectAndCommitCustomCode(kbc, git, injector)

			By("running alpha update command with to-version flag")
			runAlphaUpdateWithToVersion(kbc, fromVersion, "v4.6.0")

			By("verifying custom code is preserved")
			verifyCustomCodePreservation(kbc)

			By("verifying project state after update")
			verifyUpdateOutcome(kbc, git)
		})

		It("should successfully upgrade project with specific to-version flag", func() {
			By("initializing git repository")
			initializeGitRepository(git)

			By("scaffolding project with old kubebuilder version")
			scaffoldProjectWithOldVersion(kbc, oldBinaryPath)

			By("creating API with old version")
			createAPIWithOldVersion(kbc, oldBinaryPath)

			By("injecting custom code into API and controller")
			injectAndCommitCustomCode(kbc, git, injector)

			By("running alpha update command with to-version flag")
			runAlphaUpdateWithToVersion(kbc, fromVersion, "v4.6.0")

			By("verifying custom code is preserved")
			verifyCustomCodePreservation(kbc)

			By("verifying project state after update")
			verifyUpdateOutcome(kbc, git)
		})
	})
})

// Helper functions for test scenarios

// initializeGitRepository initializes a git repository with proper configuration
func initializeGitRepository(git *utils.GitHelper) {
	Expect(git.Init()).To(Succeed())
	Expect(git.ConfigUser("Test User", "test@example.com")).To(Succeed())
	// Ensure we're on a main branch for alpha update command
	Expect(git.CheckoutNewBranch("main")).To(Succeed())
}

// scaffoldProjectWithOldVersion scaffolds a project using the old kubebuilder version
func scaffoldProjectWithOldVersion(kbc *utils.TestContext, oldBinaryPath string) {
	initArgs := []string{
		"init",
		"--domain", kbc.Domain,
		"--repo", fmt.Sprintf("github.com/example/%s", kbc.TestSuffix),
	}
	Expect(runCommandWithBinary(kbc, oldBinaryPath, initArgs...)).To(Succeed())
	Expect(kbc.Tidy()).To(Succeed())
}

// createAPIWithOldVersion creates an API using the old kubebuilder version
func createAPIWithOldVersion(kbc *utils.TestContext, oldBinaryPath string) {
	apiArgs := []string{
		"create", "api",
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--resource",
		"--controller",
	}
	Expect(runCommandWithBinary(kbc, oldBinaryPath, apiArgs...)).To(Succeed())
	Expect(kbc.Tidy()).To(Succeed())
}

// injectAndCommitCustomCode injects custom code and commits the changes
func injectAndCommitCustomCode(kbc *utils.TestContext, git *utils.GitHelper, injector *utils.CodeInjector) {
	const (
		customAPIMarker        = "// CUSTOM_API_CODE: This is custom API code"
		customControllerMarker = "// CUSTOM_CONTROLLER_CODE: This is custom controller code"
	)

	Expect(injector.InjectAPICode(kbc, customAPIMarker)).To(Succeed())
	Expect(injector.InjectControllerCode(kbc, customControllerMarker)).To(Succeed())

	Expect(git.Add(".")).To(Succeed())
	Expect(git.Commit("Add custom code for testing preservation")).To(Succeed())
}

// commitInitialState commits the initial project state
func commitInitialState(git *utils.GitHelper) {
	Expect(git.Add(".")).To(Succeed())
	Expect(git.Commit("Initial project state")).To(Succeed())
}

// Alpha update command execution functions

// runAlphaUpdateWithFromVersion runs alpha update with --from-version flag
func runAlphaUpdateWithFromVersion(kbc *utils.TestContext, fromVersion string) {
	Expect(kbc.AlphaUpdate("--from-version", fromVersion)).To(Succeed())
}

// runAlphaUpdateWithFromBranch runs alpha update with --from-branch flag
func runAlphaUpdateWithFromBranch(kbc *utils.TestContext, fromBranch, fromVersion string) {
	Expect(kbc.AlphaUpdate("--from-branch", fromBranch, "--from-version", fromVersion)).To(Succeed())
}

// runAlphaUpdateWithToVersion runs alpha update with both --from-version and --to-version flags
func runAlphaUpdateWithToVersion(kbc *utils.TestContext, fromVersion, toVersion string) {
	Expect(kbc.AlphaUpdate("--from-version", fromVersion, "--to-version", toVersion)).To(Succeed())
}

// Validation functions

// verifyCustomCodePreservation verifies that custom code markers are preserved
func verifyCustomCodePreservation(kbc *utils.TestContext) {
	const (
		customAPIMarker        = "// CUSTOM_API_CODE: This is custom API code"
		customControllerMarker = "// CUSTOM_CONTROLLER_CODE: This is custom controller code"
	)

	validator, err := utils.NewProjectValidator(filepath.Join(kbc.Dir, "PROJECT"))
	Expect(err).NotTo(HaveOccurred())

	customMarkers := map[string]string{
		filepath.Join("api", kbc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind))): customAPIMarker,
		filepath.Join("internal", "controller", fmt.Sprintf("%s_controller.go",
			strings.ToLower(kbc.Kind))): customControllerMarker,
	}

	validator.ValidateCustomCodePreservation(kbc, customMarkers)
}

// verifyUpdateOutcome verifies the overall outcome of the alpha update command
func verifyUpdateOutcome(kbc *utils.TestContext, git *utils.GitHelper) {
	validator, err := utils.NewProjectValidator(filepath.Join(kbc.Dir, "PROJECT"))
	Expect(err).NotTo(HaveOccurred())

	validator.ValidateUpdateOutcome(kbc, git)
	validator.ValidateBasicProjectStructure(kbc)
}

// Utility functions for command execution with old binary

// runCommandWithBinary runs a command with a specific binary path
func runCommandWithBinary(kbc *utils.TestContext, binaryPath string, args ...string) error {
	cmd := kbc.BinaryName
	kbc.BinaryName = binaryPath
	defer func() { kbc.BinaryName = cmd }()

	switch args[0] {
	case "init":
		if err := kbc.Init(args[1:]...); err != nil {
			return fmt.Errorf("failed to init project: %w", err)
		}
		return nil
	case "create":
		if len(args) > 1 && args[1] == "api" {
			if err := kbc.CreateAPI(args[2:]...); err != nil {
				return fmt.Errorf("failed to create API: %w", err)
			}
			return nil
		} else if len(args) > 1 && args[1] == "webhook" {
			if err := kbc.CreateWebhook(args[2:]...); err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}
			return nil
		}
		return fmt.Errorf("unsupported create command: %v", args)
	case "edit":
		if err := kbc.Edit(args[1:]...); err != nil {
			return fmt.Errorf("failed to edit project: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported command: %v", args)
	}
}
