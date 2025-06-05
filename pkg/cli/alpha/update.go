package alpha

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal"
)

func NewUpdateCommand() *cobra.Command {
	opts := internal.Update{}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "TODO: add a short description of the command",
		Long: `TODO: add a long description of the command.
Provide usage examples.`,
		// 	TODO: add validation here
		//	PreRunE: func(_ *cobra.Command, _ []string) error {
		//		return opts.Validate()
		//	},
		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				log.Fatalf("TODO: fail message: %s", err)
			}
		},
	}
	// TODO: add flags here later
	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"TODO: add usage of the --from-version tag here")

	return updateCmd
}
