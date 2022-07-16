/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	pkgserverv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"go/types"
	"opcap/internal/operator"

	"opcap/internal/capability"

	"github.com/spf13/cobra"
)

type CheckCommandFlags struct {
	CatalogSource          string `json:"catalogsource"`
	CatalogSourceNamespace string `json:"catalogsourcenamespace"`
}

var checkflags CheckCommandFlags

// TODO: provide godoc compatible comment for checkCmd
var checkCmd = &cobra.Command{
	Use: "check",
	// TODO: provide Short description for check command
	Short: "",
	// TODO: provide Long description for check command
	Long: ``,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		psc, err := operator.NewOpCapClient()
		if err != nil {
			return types.Error{Msg: "Unable to create OpCap client."}
		}
		var packageManifestList pkgserverv1.PackageManifestList
		err = psc.ListPackageManifests(context.TODO(), &packageManifestList)
		if err != nil {
			return types.Error{Msg: "Unable to list PackageManifests."}
		}

		if len(packageManifestList.Items) == 0 {
			return types.Error{Msg: "No PackageManifests returned from PackageServer."}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("check called")
		capability.OperatorInstallAllFromCatalog(checkflags.CatalogSource, checkflags.CatalogSourceNamespace)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	flags := checkCmd.Flags()

	flags.StringVar(&checkflags.CatalogSource, "catalogsource", "certified-operators",
		"")
	flags.StringVar(&checkflags.CatalogSourceNamespace, "catalogsourcenamespace", "openshift-marketplace",
		"")
}
