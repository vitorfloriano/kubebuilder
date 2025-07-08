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
	"os/exec"
	"strings"
)

// GitHelper provides utilities for git operations in tests
type GitHelper struct {
	dir string
	env []string
}

// NewGitHelper creates a new GitHelper for the specified directory
func NewGitHelper(dir string, env []string) *GitHelper {
	return &GitHelper{
		dir: dir,
		env: env,
	}
}

func (g *GitHelper) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	cmd.Env = g.env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (g *GitHelper) Init() error {
	_, err := g.run("init")
	return err
}

func (g *GitHelper) ConfigUser(name, email string) error {
	if _, err := g.run("config", "user.name", name); err != nil {
		return err
	}
	_, err := g.run("config", "user.email", email)
	return err
}

func (g *GitHelper) Add(files ...string) error {
	args := append([]string{"add"}, files...)
	_, err := g.run(args...)
	return err
}

func (g *GitHelper) Commit(message string) error {
	_, err := g.run("commit", "-m", message)
	return err
}

func (g *GitHelper) CheckoutNewBranch(branch string) error {
	_, err := g.run("checkout", "-b", branch)
	return err
}

func (g *GitHelper) GetCurrentBranch() (string, error) {
	output, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}
