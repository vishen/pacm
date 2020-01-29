package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pacm",
	Short: "Simple package manager for binaries",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "f", "", "pacm config file to load (defaults to ~/.config/pacm/config)")
	rootCmd.PersistentFlags().BoolP("log-commands", "x", false, "log commands being run")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose debug logging")
}
