package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/releases"
)

// listUpdatesCmd represents the listUpdates command
var listUpdatesCmd = &cobra.Command{
	Use:   "list-updates <recipe1> <recipe2>",
	Short: "Available updates for installed package",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			fmt.Printf("requires at least one argument\n")
			return
		}

		configPath, _ := cmd.Flags().GetString("config")
		conf, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return
		}

		for _, r := range conf.Recipes {
			if r.ReleasesGithub == "" {
				continue
			}
			found := false
			for _, a := range args {
				if a == r.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}

			grs, err := releases.GithubReleases(r.ReleasesGithub)
			if err != nil {
				log.Fatal(err)
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Tag", "Status", "GithubStatus", "Published"})

			for _, g := range grs {
				d := make([]string, 4)
				d[0] = fmt.Sprintf("%s@%s", r.Name, g.TagName)
				for _, p := range conf.Packages {
					if p.RecipeName == r.Name {
						if g.TagName == p.Version || g.TagName == "v"+p.Version {
							if p.Active {
								d[1] = "active,"
							}
							d[1] += "installed"
						}
					}
				}
				if g.Draft {
					d[2] = "draft"
				}
				if g.Prerelease {
					if len(d[2]) > 0 {
						d[2] += ","
					}
					d[2] += "pre-release"
				}
				d[3] = fmt.Sprintf("%s", g.PublishedAt)
				table.Append(d)
			}
			table.Render() // Send output
		}
	},
}

func init() {
	rootCmd.AddCommand(listUpdatesCmd)
}
