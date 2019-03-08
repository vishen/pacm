package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/env"
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
	case "env", "shell":
		// TODO: Create a new "shell" and override the PATH to include
		// the specified packages as the pseudo-active ones.
		if len(os.Args[2:]) == 0 {
			fmt.Printf("need <recipe>@<version>'s to make active\n")
			return
		}
		pkgs := []*config.Package{}
		for _, recipeAndVersion := range os.Args[2:] {
			pkg, err := extractAndCheckRecipeAndVersion(conf, recipeAndVersion)
			if err != nil {
				log.Fatal(err)
			}
			pkgs = append(pkgs, pkg)
		}
		if err := env.Env(conf, pkgs); err != nil {
			log.Fatal(err)
		}
	case "status":
		if os.Getenv("PACM_IN_SHELL") == "true" {
			fmt.Println("Currently in a pacm shell")
			fmt.Printf("Using the following packages: %s", os.Getenv("PACM_PACKAGES"))
		} else {
			fmt.Println("Not in a shell")
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
			// TODO: This doesn't actually show the binaries installed
			// on disk.
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
			fmt.Printf("recipe %s\n", r.Name)
			for _, g := range grs {
				if g.Draft {
					continue
				}
				fmt.Printf("> %s", g.TagName)
				for _, p := range conf.Packages {
					if p.RecipeName == r.Name {
						if g.TagName == p.Version || g.TagName == "v"+p.Version {
							fmt.Printf(" [INSTALLED]")
							if p.Active {
								fmt.Printf(" [ACTIVE]")
							}
						}
					}
				}
				if g.Prerelease {
					fmt.Printf(" [PRE-RELEASE]")
				}
				fmt.Println()
				//fmt.Printf(" \tpublished=%s\n", g.PublishedAt)
			}
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
