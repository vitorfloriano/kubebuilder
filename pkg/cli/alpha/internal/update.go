package internal

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

type Update struct {
	// TODO: populate the struct
	FromVersion string
}

func (opts *Update) Update() error {
	// TODO: add implementation logic
	fmt.Println("WIP...")

	projectConfigFile := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil { // TODO: assess if DefaultPath could be renamed to ConfigFilePath
		return fmt.Errorf("TODO: add error message: %w", err)
	}

	cliVersion := projectConfigFile.Config().GetCliVersion()

	// TODO: add logic for overriding the cliVersion from the PROJECT file with the
	// value passed to the --from-version flag just so that projects prior to 4.6.0
	// can use it.

	log.Infof("TODO: add message for CLI version being used: %s", cliVersion)

	return nil
}
