// Copyright Jetstack Ltd. See LICENSE for details.
package cmd

import (
	"github.com/spf13/cobra"
)

var clusterInstancesCmd = &cobra.Command{
	Use:     "instances",
	Short:   "Operations on instances",
	Aliases: []string{"instance"},
}

func init() {
	clusterCmd.AddCommand(clusterInstancesCmd)
}
