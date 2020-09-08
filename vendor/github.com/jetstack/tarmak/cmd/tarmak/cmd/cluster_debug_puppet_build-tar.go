// Copyright Jetstack Ltd. See LICENSE for details.
package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/jetstack/tarmak/pkg/tarmak"
)

// distCmd represents the dist command
var clusterDebugPuppetBuildTarCmd = &cobra.Command{
	Use:   "build-tar",
	Short: "Build a puppet.tar.gz in the current working directory",
	Run: func(cmd *cobra.Command, args []string) {
		t := tarmak.New(globalFlags)
		defer t.Cleanup()

		path := "puppet.tar.gz"

		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Fatalf("error creating %s: %s", path, err)
		}

		if err = t.Puppet().TarGz(file); err != nil {
			log.Fatalf("error writing to %s: %s", path, err)
		}

		if err := file.Close(); err != nil {
			log.Fatalf("error closing %s: %s", path, err)
		}
	},
}

func init() {
	clusterDebugPuppetCmd.AddCommand(clusterDebugPuppetBuildTarCmd)
}
