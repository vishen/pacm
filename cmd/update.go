package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update <recipe>@<version> <recipe>@<version>",
	Short: "Update packages",
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
		currentArch := runtime.GOARCH
		currentOS := runtime.GOOS
		for _, recipeAndVersion := range args {
			parts := strings.Split(recipeAndVersion, "@")
			if len(parts) != 2 {
				fmt.Printf("expected <recipe>@<version>, received %q\n", recipeAndVersion)
				return
			}
			// TODO: HACK: Dumb hack to remove leading 'v' from the version since most
			// recipes don't have the v. THIS IS NOT A FIX, and won't always
			// work.
			version := parts[1]
			if len(version) > 0 && version[0] == 'v' {
				version = version[1:]
			}

			if err := conf.AddPackage(currentArch, currentOS, parts[0], version); err != nil {
				fmt.Printf("unable to add package %q: %v\n", recipeAndVersion, err)
				return
			}
			if err := conf.CreatePackagesForRecipe(parts[0], currentArch, currentOS); err != nil {
				fmt.Printf("error downloading and installing packages: %v", err)
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
