package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update <recipe>@<version> <recipe>@<version>",
	Short: "Update packages",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}
		if len(args) == 0 {
			fmt.Printf("need <recipe>@<version>'s to make active\n")
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
			recipe := parts[1]
			if len(recipe) > 0 && recipe[0] == 'v' {
				recipe = recipe[1:]
			}

			if err := conf.AddPackage(currentArch, currentOS, parts[0], recipe); err != nil {
				fmt.Printf("unable to add package %q: %v\n", recipeAndVersion, err)
				return
			}
		}
		// TODO: Should only do the unlinking and linking of packages.
		if err := conf.CreatePackages(currentArch, currentOS); err != nil {
			fmt.Printf("error downloading and installing packages: %v", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
