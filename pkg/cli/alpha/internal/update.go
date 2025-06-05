package internal

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

type Update struct {
	FromVersion string
}

func (opts *Update) Update() error {
	fmt.Println("WIP...") // TODO: remove this print later

	projectConfigFile := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil { // TODO: assess if DefaultPath could be renamed to ConfigFilePath
		return fmt.Errorf("Fail to run the command: %w", err) // TODO: improve the error message
	}

	cliVersion := projectConfigFile.Config().GetCliVersion()

	if opts.FromVersion != "" {
		// TODO: add normalization here
		// users may input "4.5.0" or "v4.5.0"
		// so better check for that
		log.Infof("Overriding PROJECT cliVersion from %s to %s", cliVersion, opts.FromVersion) // TODO: improve override message
	} else {
		log.Infof("CLI version being used: %s", cliVersion) // TODO: improve the log message
	}

	return nil
}
