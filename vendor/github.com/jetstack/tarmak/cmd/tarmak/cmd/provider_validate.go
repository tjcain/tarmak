// Copyright Jetstack Ltd. See LICENSE for details.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/tarmak/pkg/tarmak"
)

var providerValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate provider(s) used by current cluster",
	Run: func(cmd *cobra.Command, args []string) {
		t := tarmak.New(globalFlags)
		t.Perform(t.Environment().Provider().Validate())
	},
}

func init() {
	providerCmd.AddCommand(providerValidateCmd)
}
