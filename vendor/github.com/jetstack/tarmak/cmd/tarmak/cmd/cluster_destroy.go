// Copyright Jetstack Ltd. See LICENSE for details.
package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/jetstack/tarmak/pkg/tarmak"
)

// clusterDestroyCmd handles `tarmak clusters destroy`
var clusterDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy the current cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		store := &globalFlags.Cluster.Destroy
		if store.DryRun {
			return errors.New("dry run is not yet supported")
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		t := tarmak.New(globalFlags)
		t.CancellationContext().WaitOrCancel(t.NewCmdTarmak(cmd.Flags(), args).Destroy)
	},
}

func init() {
	clusterDestroyFlags(clusterDestroyCmd.PersistentFlags())
	clusterCmd.AddCommand(clusterDestroyCmd)
}
