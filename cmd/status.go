package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status <recipe1> <recipe2>",
	Short: "Status of installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}

		verbose, _ := cmd.Flags().GetBool("verbose")

		sortedPackages := conf.Packages
		sort.Slice(sortedPackages, func(i, j int) bool {
			spi := sortedPackages[i]
			spj := sortedPackages[j]

			if spi.RecipeName != spj.RecipeName {
				return spi.RecipeName < spj.RecipeName
			}
			riTagSplit := strings.SplitN(spi.Version, ".", 3)
			rjTagSplit := strings.SplitN(spj.Version, ".", 3)
			for i := 0; i < 3; i++ {
				if riTagSplit[i] < rjTagSplit[i] {
					return false
				} else if riTagSplit[i] > rjTagSplit[i] {
					return true
				}
			}
			return true
		})

		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoMergeCells(true)
		table.SetRowLine(true)
		if !verbose {
			table.SetHeader([]string{"recipe", "version", "active"})
		} else {
			table.SetHeader([]string{"recipe", "version", "active", "mod time", "path"})
		}

		for _, p := range sortedPackages {
			if len(args) > 0 {
				found := false
				for _, a := range args {
					if a == p.RecipeName {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			var d []string
			if !verbose {
				d = make([]string, 3)
			} else {
				d = make([]string, 5)
			}
			d[0] = p.RecipeName
			d[1] = p.Version
			if p.Active {
				d[2] = fmt.Sprintf("%s@%s", p.RecipeName, p.Version)
			}

			if verbose {
				foundBinaries := false
				for _, i := range conf.CurrentlyInstalled {
					if !strings.Contains(i.SymlinkAbsolutePath, fmt.Sprintf("_pacm/%s_%s", p.RecipeName, p.Version)) {
						continue
					}
					foundBinaries = true
					d[3] = fmt.Sprintf("%s", i.ModTime.Truncate(time.Second))
					d[4] = i.AbsolutePath
				}
				if !foundBinaries {
					d[4] = "error: missing binary files on disk"
				}
			}
			table.Append(d)
		}
		table.Render() // Send output
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolP("verbose", "v", false, "display more information about what is installed")
}
