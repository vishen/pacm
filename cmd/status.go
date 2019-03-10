package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status of installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}

		fmt.Println("Installed packages:")
		for _, p := range conf.Packages {
			fmt.Printf("> %s@%s", p.RecipeName, p.Version)
			if p.Active {
				fmt.Printf(" [ACTIVE]")
			}
			if p.ExecutableName != "" {
				fmt.Printf(" executable_name=%s", p.ExecutableName)
			}
			fmt.Println()
			foundBinaries := false
			for _, i := range conf.CurrentlyInstalled {
				if !strings.Contains(i.SymlinkAbsolutePath, fmt.Sprintf("_pacm/%s_%s", p.RecipeName, p.Version)) {
					continue
				}
				foundBinaries = true
				fmt.Printf("  - %s (%s)\n", i.AbsolutePath, i.ModTime)
			}
			if !foundBinaries {
				fmt.Printf("  - error: missing binary files on disk...\n")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
