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
	"os/exec"
	"strings"
)

// GitHelper provides Git operations for test contexts
type GitHelper struct {
	dir string
	env []string
}

// NewGitHelper creates a new Git helper for the specified directory
func NewGitHelper(dir string, env []string) *GitHelper {
	return &GitHelper{
		dir: dir,
		env: env,
	}
}

// Init initializes a git repository in the test directory
func (g *GitHelper) Init() error {
	return g.runCommand("init")
}

// ConfigUser configures git user for the test repository
func (g *GitHelper) ConfigUser(name, email string) error {
	if err := g.runCommand("config", "user.name", name); err != nil {
		return err
	}
	return g.runCommand("config", "user.email", email)
}

// Add adds files to the git staging area
func (g *GitHelper) Add(files ...string) error {
	args := append([]string{"add"}, files...)
	return g.runCommand(args...)
}

// Commit commits changes with the specified message
func (g *GitHelper) Commit(message string) error {
	return g.runCommand("commit", "-m", message)
}

// Checkout checks out to a specific branch
func (g *GitHelper) Checkout(branch string) error {
	return g.runCommand("checkout", branch)
}

// CheckoutNewBranch creates and checks out a new branch
func (g *GitHelper) CheckoutNewBranch(branch string) error {
	return g.runCommand("checkout", "-b", branch)
}

// GetCurrentBranch returns the current branch name
func (g *GitHelper) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.dir
	cmd.Env = g.env

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// HasConflicts checks if there are merge conflicts in the repository
func (g *GitHelper) HasConflicts() (bool, error) {
	cmd := exec.Command("grep", "-r", "<<<<<<< HEAD", ".", "--include=*.go")
	cmd.Dir = g.dir

	err := cmd.Run()
	if err != nil {
		// grep returns non-zero exit code when no matches found
		return false, nil
	}
	return true, nil
}

// GetLastCommitMessage returns the last commit message
func (g *GitHelper) GetLastCommitMessage() (string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = g.dir
	cmd.Env = g.env

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit message: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Status returns the git status
func (g *GitHelper) Status() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.dir
	cmd.Env = g.env

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	return string(output), nil
}

// runCommand executes a git command in the test directory
func (g *GitHelper) runCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	cmd.Env = g.env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}
