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

package alpha

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal"
)

// NewUpdateCommand creates and returns a new Cobra command for updating Kubebuilder projects.
// This command helps users upgrade their projects to newer versions of Kubebuilder by performing
// a three-way merge between:
// - The original scaffolding (ancestor)
// - The user's current project state (current)
// - The new version's scaffolding (upgrade)
//
// The update process creates multiple Git branches to facilitate the merge and help users
// resolve any conflicts that may arise during the upgrade process.
func NewUpdateCommand() *cobra.Command {
	opts := internal.Update{}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Kubebuilder project to a newer version",
		Long: `This command upgrades your Kubebuilder project to the latest scaffold layout using a 3-way merge strategy.

It performs the following steps:
  1. Creates an 'ancestor' branch from the version originally used to scaffold the project
  2. Creates a 'current' branch with your project's current state
  3. Creates an 'upgrade' branch using the new version's scaffolding
  4. Attempts a 3-way merge into a 'merge' branch

The process uses Git branches:
  - ancestor: clean scaffold from the original version
  - current: your existing project state
  - upgrade: scaffold from the target version
  - merge: result of the 3-way merge

If conflicts occur during the merge, resolve them manually in the 'merge' branch. 
Once resolved, commit and push it as a pull request. This branch will contain the 
final upgraded project with the latest Kubebuilder layout and your custom code.

Examples:
  # Update using the version specified in PROJECT file
  kubebuilder alpha update

  # Update from a specific version
  kubebuilder alpha update --from-version v3.0.0`,

		PreRunE: func(_ *cobra.Command, _ []string) error {
			return opts.Validate()
		},

		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				log.Fatalf("Update failed: %s", err)
			}
		},
	}

	// Flag to override the version specified in the PROJECT file
	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"Kubebuilder binary release version to upgrade from. Should match the version used to init the project.")

	return updateCmd
}
