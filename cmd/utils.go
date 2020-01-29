package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/logging"
)

func getConfig(cmd *cobra.Command) (*config.Config, error) {
	activateLogLevel(cmd)
	configPath, _ := cmd.Flags().GetString("config")
	return config.LoadWithoutDownload(configPath)
}

func activateLogLevel(cmd *cobra.Command) {
	logging.ShouldPrintCommands, _ = cmd.Flags().GetBool("log-commands")
	logging.Debug, _ = cmd.Flags().GetBool("verbose")
}

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
