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
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/update"
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
	opts := update.Update{}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update your project to a newer version (3-way merge; optional single-commit)",
		Long: `Upgrade your project scaffold using a 3-way merge strategy while preserving your code.

The command creates temporary branches to perform the update:
  - ancestor:  clean scaffold from the original version
  - current:   your existing project state
  - upgrade:   scaffold generated with the target version
  - merge:     result of the 3-way merge (committed; conflict markers kept with --force)

By default, the merge result is committed on the temporary 'merge' branch. 
Use --squash to snapshot that result into a SINGLE commit on a stable branch,
ready for a PR:

  kubebuilder-alpha-update-to-<to-version>

This keeps history tidy (one commit per update run) and enables idempotent PRs.

Notes:
  • --force commits even if conflicts occur (markers are kept).
  • --preserve-path lets you keep files from your base branch when squashing
    (useful for CI configs like .github/workflows).
  • --output-branch optionally overrides the default squashed branch name.

Examples:
  # Update from the version in PROJECT to the latest, stop on conflicts
  kubebuilder alpha update

  # Update from a specific version to latest
  kubebuilder alpha update --from-version v4.6.0

  # Update from v4.5.0 to v4.7.0 and keep conflict markers (automation-friendly)
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force

  # Same as above, but produce ONE squashed commit on a stable PR branch
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force --squash

  # Squash while preserving CI workflows from base (e.g., main) on the squashed branch
  kubebuilder alpha update --force --squash --preserve-path .github/workflows

  # Squash into a custom output branch name
  kubebuilder alpha update --force --squash --output-branch my-update-branch

  # Create a PR automatically after update (requires gh CLI)
  kubebuilder alpha update --force --squash --open-pr

  # Create a PR with custom title/body via environment variables (supports Go templates)
  export KUBEBUILDER_UPDATE_PR_TITLE="feat: Update scaffold from {{.FromVersion}} to {{.ToVersion}}"
  export KUBEBUILDER_UPDATE_PR_BODY="Automated scaffold update from {{.FromVersion}} to {{.ToVersion}} \\
    on branch {{.BranchName}}"
  kubebuilder alpha update --force --squash --open-pr

  # Create both PR and issue (issue as fallback if PR fails)  
  kubebuilder alpha update --force --squash --open-pr --open-issue

Behavior summary:
  • Without --force:
      - If conflicts occur during the 3-way merge, the command stops on the 'merge' branch
        for manual resolution (no commit made).
  • With --force:
      - Conflicted files are committed on the 'merge' branch with conflict markers.
  • With --squash:
      - After the merge step, the exact 'merge' tree is copied to a new/updated branch
        (default: kubebuilder-alpha-update-to-<to-version>) and committed ONCE, keeping markers
        if present. This branch is intended for opening/refreshing a PR.
  • With --open-pr:
      - Requires gh CLI to be installed and authenticated ('gh auth login')
      - Creates a PR from the update branch to the base branch
      - PR title/body support Go templates with {{.FromVersion}}, {{.ToVersion}}, {{.BranchName}}
      - Can be customized via KUBEBUILDER_UPDATE_PR_TITLE/KUBEBUILDER_UPDATE_PR_BODY env vars
  • With --open-issue:
      - Can be used standalone or as fallback when --open-pr fails
      - Issue title/body support Go templates with version/branch information
      - Can be customized via KUBEBUILDER_UPDATE_ISSUE_TITLE/KUBEBUILDER_UPDATE_ISSUE_BODY env vars`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			err := opts.Prepare()
			if err != nil {
				return fmt.Errorf("failed to prepare update: %w", err)
			}
			return opts.Validate()
		},
		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				log.Fatalf("Update failed: %s", err)
			}
		},
	}

	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"binary release version to upgrade from. Should match the version used to init the project and be"+
			"a valid release version, e.g., v4.6.0. If not set, "+
			"it defaults to the version specified in the PROJECT file. ")
	updateCmd.Flags().StringVar(&opts.ToVersion, "to-version", "",
		"binary release version to upgrade to. Should be a valid release version, e.g., v4.7.0. "+
			"If not set, it defaults to the latest release version available in the project repository.")
	updateCmd.Flags().StringVar(&opts.FromBranch, "from-branch", "",
		"Git branch to use as current state of the project for the update.")
	updateCmd.Flags().BoolVar(&opts.Force, "force", false,
		"Force the update even if conflicts occur. Conflicted files will include conflict markers, and a "+
			"commit will be created automatically. Ideal for automation (e.g., cronjobs, CI).")
	updateCmd.Flags().BoolVar(&opts.Squash, "squash", false,
		"After merging, write a single squashed commit with the merge result to a fixed branch "+
			"named kubebuilder-alpha-update-to-<to-version>.")
	updateCmd.Flags().StringArrayVar(&opts.PreservePath, "preserve-path", nil,
		"Paths to preserve from the base branch when squashing (repeatable). "+
			"Example: --preserve-path .github/workflows")
	updateCmd.Flags().StringVar(&opts.OutputBranch, "output-branch", "",
		"Override the default kubebuilder-alpha-update-to-<to-version> branch name (used with --squash).")
	updateCmd.Flags().BoolVar(&opts.OpenPR, "open-pr", false,
		"Create a pull request using gh CLI after successful update. Requires gh CLI to be installed and authenticated.")
	updateCmd.Flags().BoolVar(&opts.OpenIssue, "open-issue", false,
		"Create an issue using gh CLI. Can be used standalone or as a fallback when --open-pr fails.")
	updateCmd.Flags().StringVar(&opts.CommitMessage, "commit-message", "",
		"Custom commit message for the squashed commit (used with --squash). "+
			"If not set, uses default format.")

	return updateCmd
}
