package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/releases"
)

func main() {
	conf, err := config.Load("./pacmconfig")
	if err != nil {
		log.Fatal(err)
	}

	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "activate", "use":
		if len(os.Args[2:]) == 0 {
			fmt.Printf("need <recipe>@<version>'s to make active\n")
			return
		}
		for _, recipeAndVersion := range os.Args[2:] {
			pkg, err := extractAndCheckRecipeAndVersion(conf, recipeAndVersion)
			if err != nil {
				log.Fatal(err)
			}
			conf.MakePackageActive(pkg)
		}
		// TODO: Should only do the unlinking and linking of packages.
		if err := conf.CreatePackages(runtime.GOARCH, runtime.GOOS); err != nil {
			log.Fatal(err)
		}
	case "status":
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
	case "updates":
		for _, r := range conf.Recipes {
			if r.ReleasesGithub == "" {
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
				d[0] = fmt.Sprintf("%s-%s", r.Name, g.TagName)
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
	default:
		if err := conf.CreatePackages(runtime.GOARCH, runtime.GOOS); err != nil {
			log.Fatal(err)
		}
	}
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
