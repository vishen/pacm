package cmd

import (
	"fmt"
	"log"
	"runtime"

	"github.com/spf13/cobra"
)

// activateCmd represents the activate command
var activateCmd = &cobra.Command{
	Use:   "activate <recipe>@<version> <recipe>@<version>",
	Short: "Activate packages",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("need <recipe>@<version>'s to make active\n")
			return
		}
		conf, err := getConfig(cmd)
		if err != nil {
			fmt.Printf("unable to load config: %v\n", err)
			return
		}
		for _, recipeAndVersion := range args {
			pkg, err := extractAndCheckRecipeAndVersion(conf, recipeAndVersion)
			if err != nil {
				log.Fatal(err)
			}
			conf.MakePackageActive(pkg)
			if err := conf.CreatePackagesForRecipe(pkg.RecipeName, runtime.GOARCH, runtime.GOOS); err != nil {
				fmt.Printf("error downloading and installing packages: %v", err)
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
}
