// Copyright Jetstack Ltd. See LICENSE for details.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/tarmak/pkg/tarmak"
)

var clusterKubectlCmd = &cobra.Command{
	Use:   "kubectl [kubectl command arguments]",
	Short: "Run kubectl on the current cluster",
	Run: func(cmd *cobra.Command, args []string) {
		t := tarmak.New(globalFlags)
		t.Perform(t.NewCmdTarmak(cmd.Flags(), args).Kubectl())
	},
	DisableFlagsInUseLine: true,
}

func init() {
	clusterCmd.AddCommand(clusterKubectlCmd)
}
