package env

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	"github.com/vishen/pacm/config"
)

func Env(conf *config.Config, packages []*config.Package) error {
	binPath, err := ioutil.TempDir("", "pacm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(binPath)
	// TODO: Error if this is not set, or maybe fallback
	// to /bin/sh ??
	defaultShell := os.Getenv("SHELL")

	fmt.Printf("pacm env: shell=%s bindir=%s\n", defaultShell, binPath)
	for _, p := range packages {
		fmt.Printf(">> using %s@%s\n", p.RecipeName, p.Version)
	}

	// Set the output dir to be the temp binary path
	conf.OutputDir = binPath

	// Only keep the packages that are being used
	// TODO: Error if we are already using a recipe of the same name?
	pkgs := []*config.Package{}
	var pkgsString string
	for _, pkg := range packages {
		for _, p := range conf.Packages {
			if p.RecipeName == pkg.RecipeName && p.Version == pkg.Version {
				pkg.Active = true
				pkgs = append(pkgs, pkg)
				pkgsString += fmt.Sprintf("%s@%s,", p.RecipeName, p.Version)
				break
			}
		}
	}
	conf.Packages = pkgs

	if err := conf.CreatePackages(runtime.GOARCH, runtime.GOOS); err != nil {
		return err
	}

	envPath := os.Getenv("PATH")
	envPath = binPath + ":" + envPath
	environ := os.Environ()
	environ = append(environ, []string{
		"PATH=" + envPath,
		"PACM_IN_SHELL=true",
		fmt.Sprintf("PACM_PACKAGES=%s", pkgsString),
	}...)

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, defaultShell)
	cmd.Env = environ
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	fmt.Println("Exited pacm shell!")
	return nil
}
