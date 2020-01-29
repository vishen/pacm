package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up cached archives",
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := getConfig(cmd)
		if err != nil {
			fmt.Printf("unable to load config: %v\n", err)
			return
		}
		conf.RemoveUnusedCachedArchivePackages(runtime.GOARCH, runtime.GOOS)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
