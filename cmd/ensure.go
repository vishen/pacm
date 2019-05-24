package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// ensureCmd represents the ensure command
var ensureCmd = &cobra.Command{
	Use:   "ensure",
	Short: "Ensure that your binaries are up-to-date",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}
		if err := conf.CreatePackages(runtime.GOARCH, runtime.GOOS); err != nil {
			fmt.Printf("error downloading and installing packages: %v", err)
			return
		}
		fmt.Println("Everything is up-to-date")
	},
}

func init() {
	rootCmd.AddCommand(ensureCmd)
}
