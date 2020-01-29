package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/vishen/pacm/utils"
)

type status struct {
	recipe  string
	version string
	active  bool
	modtime time.Time
	path    string
	err     string
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status <recipe1> <recipe2>",
	Short: "Status of installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := getConfig(cmd)
		if err != nil {
			fmt.Printf("unable to load config: %v\n", err)
			return
		}

		showMore, _ := cmd.Flags().GetBool("show-more")

		sortedPackages := conf.Packages
		sort.Slice(sortedPackages, func(i, j int) bool {
			spi := sortedPackages[i]
			spj := sortedPackages[j]

			if spi.RecipeName != spj.RecipeName {
				return spi.RecipeName < spj.RecipeName
			}
			return utils.SemvarIsBigger(spi.Version, spj.Version)
		})

		packageStatus := make([]status, len(sortedPackages))
		foundError := false
		for i, p := range sortedPackages {
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
			s := status{
				recipe:  p.RecipeName,
				version: p.Version,
			}
			if p.Active {
				s.active = true
			}

			foundBinaries := false
			for _, i := range conf.CurrentlyInstalled {
				if !strings.Contains(i.SymlinkAbsolutePath, fmt.Sprintf("_pacm/%s_%s", p.RecipeName, p.Version)) {
					continue
				}
				foundBinaries = true
				s.modtime = i.ModTime.Truncate(time.Second)
				s.path = i.AbsolutePath
			}
			if !foundBinaries {
				s.err = "error: missing binary files on disk"
				foundError = true
			}
			packageStatus[i] = s
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoMergeCells(true)
		table.SetRowLine(true)
		var headerLength int
		if foundError {
			table.SetHeader([]string{"recipe", "version", "active", "error"})
			headerLength = 4
		} else {
			if !showMore {
				table.SetHeader([]string{"recipe", "version", "active"})
				headerLength = 3
			} else {
				table.SetHeader([]string{"recipe", "version", "active", "mod time", "path"})
				headerLength = 5
			}
		}
		for _, s := range packageStatus {
			d := make([]string, headerLength)
			d[0] = s.recipe
			d[1] = s.version
			if s.active {
				d[2] = fmt.Sprintf("%s@%s", s.recipe, s.version)
			}
			if foundError {
				d[3] = s.err
			} else if showMore {
				d[3] = fmt.Sprintf("%s", s.modtime)
				d[4] = s.path
			}
			table.Append(d)
		}
		table.Render() // Send output
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().Bool("show-more", false, "display more information about what is installed")
}
