package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
)

func env(config *Config, packages []*Package) error {
	binPath, err := ioutil.TempDir("", "pacm")
	if err != nil {
		return err
	}
	// TODO: Error if this is not set, or maybe fallback
	// to /bin/sh ??
	defaultShell := os.Getenv("SHELL")

	fmt.Printf("pacm env: shell=%s bindir=%s\n", defaultShell, binPath)
	for _, p := range packages {
		fmt.Printf(">> using %s@%s\n", p.RecipeName, p.Version)
	}

	// Set the output dir to be the temp binary path
	config.OutputDir = binPath

	// Only keep the packages that are being used
	// TODO: Error if we are already using a recipe of the same name?
	pkgs := []*Package{}
	for _, pkg := range packages {
		for _, p := range config.Packages {
			if p.RecipeName == pkg.RecipeName && p.Version == pkg.Version {
				pkg.Active = true
				pkgs = append(pkgs, pkg)
				break
			}
		}
	}
	config.Packages = pkgs

	if err := createRecipes(runtime.GOARCH, runtime.GOOS, config); err != nil {
		return err
	}

	envPath := os.Getenv("PATH")
	envPath = binPath + ":" + envPath
	environ := os.Environ()
	environ = append(environ, []string{"PATH=" + envPath, "PACM_IN_SHELL=true"}...)

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, defaultShell)
	cmd.Env = environ
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	fmt.Println("Exited pacm shell!")
	return nil
}
