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
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const (
	fromVersion = "v4.5.2"
	toVersion   = "v4.6.0"
)

var _ = Describe("kubebuilder", func() {
	Context("alpha update", func() {
		var (
			kbc             *utils.TestContext
			mockProjectDir  string
			kbOldBinaryPath string
		)

		BeforeEach(func() {
			var err error
			By("setting up test context for kubebuilder binary management")
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("creating isolated mock project directory in /tmp to avoid git conflicts")
			mockProjectDir, err = os.MkdirTemp("", "kubebuilder-mock-project-")
			Expect(err).NotTo(HaveOccurred())

			By("downloading kubebuilder v4.5.2 binary to isolated /tmp directory")
			kbOldBinaryPath, err = downloadKubebuilder()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("cleaning up test artifacts")
			if mockProjectDir != "" {
				_ = os.RemoveAll(mockProjectDir)
			}
			if kbOldBinaryPath != "" {
				_ = os.RemoveAll(filepath.Dir(kbOldBinaryPath))
			}
			kbc.Destroy()
		})

		It("should update project from v4.5.2 to v4.6.0 preserving custom code", func() {
			By("creating mock project with kubebuilder v4.5.2")
			createMockProject(mockProjectDir, kbOldBinaryPath)

			By("injecting custom code in API and controller")
			injectCustomCode(mockProjectDir)

			By("initializing git repository and committing mock project")
			initializeGitRepo(mockProjectDir)

			By("running alpha update from v4.5.2 to v4.6.0")
			runAlphaUpdate(kbc, mockProjectDir)

			By("validating custom code preservation")
			validateCustomCodePreservation(mockProjectDir)
		})
	})
})

// downloadKubebuilder downloads the --from-version kubebuilder binary to a temporary directory
func downloadKubebuilder() (string, error) {
	binaryDir, err := os.MkdirTemp("", "kubebuilder-v4.5.2-")
	if err != nil {
		return "", fmt.Errorf("failed to create binary directory: %w", err)
	}

	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_linux_amd64",
		fromVersion,
	)
	binaryPath := filepath.Join(binaryDir, "kubebuilder")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download kubebuilder %s: %w", fromVersion, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download kubebuilder %s: HTTP %d", fromVersion, resp.StatusCode)
	}

	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write binary: %w", err)
	}

	err = os.Chmod(binaryPath, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

func createMockProject(projectDir, binaryPath string) {
	err := os.Chdir(projectDir)
	Expect(err).NotTo(HaveOccurred())

	By("running kubebuilder init")
	cmd := exec.Command(binaryPath, "init", "--domain", "example.com", "--repo", "github.com/example/test-operator")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("running kubebuilder create api")
	cmd = exec.Command(
		binaryPath, "create", "api",
		"--group", "webapp",
		"--version", "v1",
		"--kind", "TestOperator",
		"--resource", "--controller",
	)
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("running make all")
	cmd = exec.Command("make", "all")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func injectCustomCode(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	err := pluginutil.InsertCode(
		typesFile,
		"Foo string `json:\"foo,omitempty\"`",
		"\n\t// +kubebuilder:validation:Minimum=0"+
			"\n\t// +kubebuilder:validation:Maximum=3"+
			"\n\t// +kubebuilder:default=1"+
			"\n\t// Size is the size of the memcached deployment"+
			"\n\tSize int32 `json:\"size,omitempty\"`",
	)
	Expect(err).NotTo(HaveOccurred())
	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	err = pluginutil.InsertCode(
		controllerFile,
		"// TODO(user): your logic here",
		"// Custom reconciliation logic\n\tlog := ctrl.LoggerFrom(ctx)"+
			"\n\tlog.Info(\"Reconciling TestOperator\")"+
			"\n\n\t// Fetch the TestOperator instance"+
			"\n\ttestOperator := &webappv1.TestOperator{}"+
			"\n\terr := r.Get(ctx, req.NamespacedName, testOperator)"+
			"\n\tif err != nil {"+
			"\n\t\treturn ctrl.Result{}, client.IgnoreNotFound(err)"+
			"\n\t}"+
			"\n\n\t// Custom logic: log the size field"+
			"\n\tlog.Info(\"TestOperator size\", \"size\", testOperator.Spec.Size)",
	)
	Expect(err).NotTo(HaveOccurred())
}

func initializeGitRepo(projectDir string) {
	By("initializing git repository")
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("adding all files to git")
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	By("committing initial project state")
	cmd = exec.Command("git", "commit", "-m", "Initial project with custom code")
	cmd.Dir = projectDir
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func runAlphaUpdate(kbc *utils.TestContext, projectDir string) {
	err := os.Chdir(projectDir)
	Expect(err).NotTo(HaveOccurred())
	cmd := exec.Command(kbc.BinaryName, "alpha", "update", "--from-version", fromVersion, "--to-version", toVersion)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Alpha update failed: %s", string(output))
}

func validateCustomCodePreservation(projectDir string) {
	typesFile := filepath.Join(projectDir, "api", "v1", "testoperator_types.go")
	content, err := os.ReadFile(typesFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("Size int32 `json:\"size,omitempty\"`"))
	Expect(string(content)).To(ContainSubstring("Size is the size of the memcached deployment"))

	controllerFile := filepath.Join(projectDir, "internal", "controller", "testoperator_controller.go")
	content, err = os.ReadFile(controllerFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("Custom reconciliation logic"))
	Expect(string(content)).To(ContainSubstring("log.Info(\"Reconciling TestOperator\")"))
	Expect(string(content)).To(ContainSubstring("log.Info(\"TestOperator size\", \"size\", testOperator.Spec.Size)"))

	projectFile := filepath.Join(projectDir, "PROJECT")
	content, err = os.ReadFile(projectFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(ContainSubstring("version: \"3\""))
}
