package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <recipe>@<version> <recipe>@<version>",
	Short: "Install packages",
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
			if err := conf.AddPackage(currentArch, currentOS, parts[0], parts[1]); err != nil {
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
	rootCmd.AddCommand(installCmd)
}
