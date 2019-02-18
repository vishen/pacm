package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/vishen/pacm/config"
	"github.com/vishen/pacm/env"
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
	case "env":
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
