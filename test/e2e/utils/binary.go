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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// DownloadKubebuilderBinary downloads the specified kubebuilder version and returns the path
func DownloadKubebuilderBinary(version string) (string, error) {
	tempDir, err := os.MkdirTemp("", "kubebuilder-"+version+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryPath := filepath.Join(tempDir, "kubebuilder")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Determine platform-specific download URL
	var platform string
	switch runtime.GOOS {
	case "darwin":
		platform = "darwin_amd64"
	case "linux":
		platform = "linux_amd64"
	case "windows":
		platform = "windows_amd64"
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	url := fmt.Sprintf("https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s",
		version, platform)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

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

// CleanupBinary removes the temporary directory containing the downloaded binary
func CleanupBinary(binaryPath string) error {
	if binaryPath == "" {
		return nil
	}
	if err := os.RemoveAll(filepath.Dir(binaryPath)); err != nil {
		return fmt.Errorf("failed to remove binary directory: %w", err)
	}
	return nil
}
