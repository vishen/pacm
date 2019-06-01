package cmd

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/logging"
)

// activateCmd represents the activate command
var activateCmd = &cobra.Command{
	Use:   "activate <recipe>@<version> <recipe>@<version>",
	Short: "Activate packages",
	Run: func(cmd *cobra.Command, args []string) {
		activateLogLevel(cmd)
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.LoadWithoutDownload(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}
		if len(args) == 0 {
			fmt.Printf("need <recipe>@<version>'s to make active\n")
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

func activateLogLevel(cmd *cobra.Command) {
	logging.ShouldPrintCommands, _ = cmd.Flags().GetBool("log-commands")
}

// TODO: move to a utils?
func extractAndCheckRecipeAndVersion(conf *config.Config, recipeAndVersion string) (*config.Package, error) {
	s := strings.Split(recipeAndVersion, "@")
	if len(s) != 2 {
		return nil, fmt.Errorf("%q needs to be in format <recipe>@<version>", recipeAndVersion)
	}
	var pkg *config.Package
	for _, p := range conf.Packages {
		if p.RecipeName == s[0] && p.Version == s[1] {
			pkg = p
			break
		}
	}
	if pkg == nil {
		return nil, fmt.Errorf("%q is not a known package, please add to your pacmconfig", recipeAndVersion)
	}
	return pkg, nil
}

func init() {
	rootCmd.AddCommand(activateCmd)
}
