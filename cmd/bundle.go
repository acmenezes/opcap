/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

func bundleCmd() *cobra.Command {
	// bundleCmd represents the bundle command
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Bundle commands",
		Long:  `Takes commands that will operate on bundles such as bundle list to discover operator versions`,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(bundleListCmd())

	return cmd
}
