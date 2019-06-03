package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up cached archives",
	Run: func(cmd *cobra.Command, args []string) {
		activateLogLevel(cmd)
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}
		conf.RemoveUnusedCachedArchivePackages(runtime.GOARCH, runtime.GOOS)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
