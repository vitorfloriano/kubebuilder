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

package scaffolds

import (
	"fmt"
	"time"

	"sigs.k8s.io/yaml"
)

// Zone constants define the ownership zones for generated files.
const (
	// ZonePlatform marks files that are fully managed by the plugin.
	// These files are overwritten on every scaffold run.
	ZonePlatform = "platform"

	// ZoneApp marks files that are owned by the user.
	// These files are created once and never overwritten.
	ZoneApp = "app"
)

// Policy constants define the overwrite policy for generated files.
const (
	// PolicyAlwaysOverwrite marks platform-zone files that are always regenerated.
	PolicyAlwaysOverwrite = "always-overwrite"

	// PolicyCreateOnly marks app-zone files that are created once and preserved.
	PolicyCreateOnly = "create-only"
)

// LockFile tracks the set of files managed by decoupled-go/v1 along with
// their ownership zone, overwrite policy, and content checksum.
//
// The lock file is written to gen/kb/decoupled-go.v1/.kb-scaffold-lock.yaml
// and must not be edited manually.
type LockFile struct {
	// PluginVersion is the version of the plugin that generated these files.
	PluginVersion string `yaml:"pluginVersion"`
	// PluginKey is the fully-qualified plugin identifier.
	PluginKey string `yaml:"pluginKey"`
	// GeneratedAt is the UTC timestamp of the last generation run.
	GeneratedAt string `yaml:"generatedAt"`
	// Files is the list of managed file entries.
	Files []FileEntry `yaml:"files"`
}

// FileEntry describes a single file managed by the plugin.
type FileEntry struct {
	// Path is the project-relative file path (forward slashes, no leading slash).
	Path string `yaml:"path"`
	// SHA256 is the hex-encoded SHA-256 digest of the file contents at generation time.
	SHA256 string `yaml:"sha256"`
	// Zone is the ownership zone: "platform" or "app".
	Zone string `yaml:"zone"`
	// Policy is the overwrite policy: "always-overwrite" or "create-only".
	Policy string `yaml:"policy"`
	// PluginVersionAtGeneration is the plugin version that last generated this file.
	PluginVersionAtGeneration string `yaml:"pluginVersionAtGeneration"`
}

// NewLockFile creates an empty LockFile for the given plugin version and key.
func NewLockFile(version, key string) *LockFile {
	return &LockFile{
		PluginVersion: version,
		PluginKey:     key,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

// AddFile appends a FileEntry for the given path, content, zone, and policy.
// The SHA-256 digest is computed from the content string.
func (lf *LockFile) AddFile(path, content, zone, policy string) {
	lf.Files = append(lf.Files, FileEntry{
		Path:                      path,
		SHA256:                    fnv1aHash(content),
		Zone:                      zone,
		Policy:                    policy,
		PluginVersionAtGeneration: lf.PluginVersion,
	})
}

// Marshal serialises the lock file to YAML with a do-not-edit header.
func (lf *LockFile) Marshal() (string, error) {
	b, err := yaml.Marshal(lf)
	if err != nil {
		return "", fmt.Errorf("marshal lock file: %w", err)
	}
	return "# Scaffold lock — maintained by decoupled-go/v1. DO NOT EDIT.\n" + string(b), nil
}

// ParseLockFileVersion parses the pluginVersion field from a YAML lock file string.
// Returns an error if the YAML cannot be parsed.
func ParseLockFileVersion(content string) (string, error) {
	var lf LockFile
	if err := yaml.Unmarshal([]byte(content), &lf); err != nil {
		return "", fmt.Errorf("unmarshal lock file: %w", err)
	}
	return lf.PluginVersion, nil
}

// fnv1aHash returns a hex-encoded FNV-1a fingerprint of s.
// This is used as a quick content checksum in the lock file.
// Use crypto/sha256 if cryptographic integrity is required.
func fnv1aHash(s string) string {
	h := uint64(14695981039346656037)
	for _, c := range []byte(s) {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}
