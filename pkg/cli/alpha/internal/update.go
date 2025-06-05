package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil { // TODO: assess if DefaultPath could be renamed to ConfigFilePath
		return fmt.Errorf("Fail to run command: %w", err)
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
		return fmt.Errorf("Failed to download Kubebuilder %s binary: %w", cliVersion, err)
	}

	log.Infof("Downloaded binary kept at %s for debugging purposes", tempDir)

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
		return "", fmt.Errorf("Failed to create temporary directory: %w", err)
	}

	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("Failed to create the binary file: %w", err)
	}
	defer file.Close()

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Failed to download the binary: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to download the binary: HTTP %d", response.StatusCode)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to write the binary content to file: %w", err)
	}

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("Failed to make binary executable: %w", err)
	}

	log.Infof("Kubebuilder version %s succesfully downloaded to %s", cliVersion, binaryPath)

	return tempDir, nil
}
