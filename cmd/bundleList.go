/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-registry/pkg/api"
	opRegistry "github.com/operator-framework/operator-registry/pkg/client"
	"github.com/spf13/cobra"
)

// bundleListCmd represents the bundleList command

func bundleListCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists bundles",
		Long:  `Lists all bundles in a given catalog/package/channel`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("bundleList called")

			c, err := opRegistry.NewClient("localhost:50051")
			if err != nil {
				return err
			}

			i, err := c.ListBundles(context.TODO())
			if err != nil {
				return err
			}

			bundleList := []*api.Bundle{}

			for {

				b := i.Next()

				if b == nil {
					break
				}

				bundleList = append(bundleList, b)
				fmt.Println(b.PackageName + " " + b.ChannelName + " " + b.CsvName)

			}

			return nil
		},
	}

	return cmd
}
