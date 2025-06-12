package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

type Update struct {
	FromVersion string
}

func (opts *Update) Update() error {

	projectConfigFile := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil { // TODO: assess if DefaultPath could be renamed to a more self-descriptive name
		return fmt.Errorf("fail to run command: %w", err)
	}

	cliVersion := projectConfigFile.Config().GetCliVersion()

	if opts.FromVersion != "" {
		if !strings.HasPrefix(opts.FromVersion, "v") {
			opts.FromVersion = "v" + opts.FromVersion
		}
		log.Infof("Overriding cliVersion field %s from PROJECT file with --from-version %s", cliVersion, opts.FromVersion)
		cliVersion = opts.FromVersion
	} else {
		log.Infof("Using CLI version from PROJECT file: %s", cliVersion)
	}

	tempDir, err := opts.downloadKubebuilderBinary(cliVersion)
	if err != nil {
		return fmt.Errorf("failed to download Kubebuilder %s binary: %w", cliVersion, err)
	}
	log.Infof("Downloaded binary kept at %s for debugging purposes", tempDir)

	if err := opts.checkoutAncestorBranch(); err != nil {
		return fmt.Errorf("failed to checkout the ancestor branch: %w", err)
	}

	if err := opts.runAlphaGenerate(tempDir, cliVersion); err != nil {
		return fmt.Errorf("failed to run alpha generate on ancestor branch: %w", err)
	}

	if err := opts.checkoutCurrentOffAncestor(); err != nil {
		return fmt.Errorf("failed to checkout current off ancestor: %w", err)
	}

	if err := opts.checkoutUpgradeOffAncestor(); err != nil {
		return fmt.Errorf("failed to checkout upgrade off ancestor: %w", err)
	}

	return nil
}

func (opts *Update) downloadKubebuilderBinary(version string) (string, error) {

	cliVersion := version

	url := fmt.Sprintf("https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s",
		cliVersion, runtime.GOOS, runtime.GOARCH)

	log.Infof("Downloading the Kubebuilder %s binary from: %s", cliVersion, url)

	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+cliVersion+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer file.Close()

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download the binary: HTTP %d", response.StatusCode)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write the binary content to file: %w", err)
	}

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	log.Infof("Kubebuilder version %s succesfully downloaded to %s", cliVersion, binaryPath)

	return tempDir, nil
}

func (opts *Update) checkoutAncestorBranch() error {

	gitCmd := exec.Command("git", "checkout", "-b", "ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create and checkout ancestor branch: %w", err)
	}
	log.Info("Created and checked out ancestor branch")

	return nil
}

func (opts *Update) runAlphaGenerate(tempDir, version string) error {
	tempBinaryPath := tempDir + "/kubebuilder"

	originalPath := os.Getenv("PATH")
	tempEnvPath := tempDir + ":" + originalPath

	if err := os.Setenv("PATH", tempEnvPath); err != nil {
		return fmt.Errorf("failed to set temporary PATH: %w", err)
	}
	defer func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			log.Errorf("failed to restore original PATH: %w", err)
		}
	}()

	cmd := exec.Command(tempBinaryPath, "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}
	log.Info("Successfully ran alpha generate using Kubebuilder ", version)

	gitCmd := exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in ancestor: %w", err)
	}
	log.Info("Successfully staged all changes in ancestor")

	gitCmd = exec.Command("git", "commit", "-m", "Re-scaffold in ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes in ancestor: %w", err)
	}
	log.Info("Successfully commited changes in ancestor")

	return nil
}

func (opts *Update) checkoutCurrentOffAncestor() error {
	gitCmd := exec.Command("git", "checkout", "-b", "current", "ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout current branch off ancestor: %w", err)
	}
	log.Info("Successfully checked out current branch off ancestor")

	gitCmd = exec.Command("git", "checkout", "master", "--", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout content from master onto current: %w", err)
	}
	log.Info("Successfully checked out content from main onto current branch")

	gitCmd = exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes in current: %w", err)
	}
	log.Info("Successfully staged all changes in current")

	gitCmd = exec.Command("git", "commit", "-m", "Add content from main onto current branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	log.Info("Successfully commited changes in current")

	return nil
}

func (opts *Update) checkoutUpgradeOffAncestor() error {
	gitCmd := exec.Command("git", "checkout", "-b", "upgrade", "ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout upgrade branch off ancestor: %w", err)
	}
	log.Info("Successfully checked out upgrade branch off ancestor")

	cmd := exec.Command("kubebuilder", "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate on upgrade branch: %w", err)
	}
	log.Info("Successfully ran alpha generate on upgrade branch")

	gitCmd = exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes on upgrade: %w", err)
	}
	log.Info("Successfully staged all changes in upgrade branch")

	gitCmd = exec.Command("git", "commit", "-m", "alpha generate in upgrade branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes in upgrade branch: %w", err)
	}
	log.Info("Successfully commited changes in upgrade branch")

	return nil
}
