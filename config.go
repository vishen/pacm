package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knq/ini"
	"github.com/knq/ini/parser"
	homedir "github.com/mitchellh/go-homedir"
)

const cachePath = "~/.config/pacm/cache"

var possibleConfigPaths = []string{
	"~/.config/pacm/config",
	"~/.pacmconfig",
}

type Cache struct {
	packagePath string
	Installed   []Package `json:"installed_packages"`
}

func LoadCache() (*Cache, error) {
	var c *Cache
	cp, err := homedir.Expand(cachePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(cp)
	if err != nil {
		if file, err = os.Create(cp); err != nil {
			return err
		}
	}
	if err := json.NewDecoder(file).Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c Cache) WriteToFile() error {
	cp, err := homedir.Expand(cachePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(cp)
	if err != nil {
		if file, err = os.Create(cp); err != nil {
			return err
		}
	}
	if err := json.NewEncoder(file).Encode(c); err != nil {
		return nil, err
	}
}

type Package struct {
	RecipeName     string `json:"recipe"`
	Active         bool   `json:"active"`
	Version        string `json:"version"`
	ExecutableName string `json:"executable_name"`
}

func (p Package) FilenameWithVersion(filename string) string {
	return fmt.Sprintf("%s_%s", filename, p.Version)
}

type Recipe struct {
	Name            string
	URL             string
	AvailableArchOS map[string]string

	// NOT YET IMPLEMENTED
	ChecksumType string
	Checksum     string
}

func (r Recipe) MappedArchOS(arch, os string) (string, string) {
	// TODO: Fix the ordering, or better yet, make these concrete types.
	if m := r.AvailableArchOS[os+"_"+arch]; m != "" {
		mSplit := strings.Split(m, ":")
		return mSplit[1], mSplit[0]
	}
	return arch, os
}

type Installed struct {
	Filename  string
	ModTime   time.Time
	Symlinked bool
}

type Config struct {
	OutputDir string

	Recipes  []Recipe
	Packages []Package

	CurrentlyInstalled []Installed
}

func (c Config) RecipeForPackage(p Package) Recipe {
	for _, r := range c.Recipes {
		if r.Name == p.RecipeName {
			return r
		}
	}
	panic(fmt.Sprintf("no recipe %q found for package %s-%s", p.RecipeName, p.RecipeName, p.Version))
}

func (c Config) Validate() error {
	for i := 0; i < len(c.Packages); i++ {
		p := c.Packages[i]
		foundRecipe := false
		for j := 0; j < len(c.Recipes); j++ {
			r := c.Recipes[j]
			if p.RecipeName == r.Name {
				foundRecipe = true
				break
			}
		}
		if !foundRecipe {
			return fmt.Errorf("recipe with name %q does not exist", p.RecipeName)
		}
	}
	return nil
}

func Load(configPath string) (*Config, error) {
	var configPaths []string
	if configPath != "" {
		configPaths = []string{configPath}
	}
	if configPath == "" {
		for _, cp := range possibleConfigPaths {
			configPath, err := homedir.Expand(cp)
			if err != nil {
				return nil, err
			}
			configPaths = append(configPaths, configPath)
		}
	}

	for _, cp := range configPaths {
		reader, err := os.Open(cp)
		if err != nil {
			continue
		}
		defer reader.Close()
		f, err := ini.Load(reader)
		if err != nil {
			continue
		}
		config := &Config{
			Recipes:  []Recipe{},
			Packages: []Package{},
		}
		for _, s := range f.AllSections() {
			n := s.Name()
			switch {
			case n == "":
				if err := handleGlobal(s, config); err != nil {
					return nil, err
				}
				continue
			case strings.HasPrefix(n, "recipe "):
				if err := handleRecipe(s, config); err != nil {
					return nil, err
				}
			case strings.HasPrefix(n, "checksum "):
				// TODO: Handle checksums
			default:
				if err := handlePackage(s, config); err != nil {
					return nil, err
				}
			}
		}
		if err := config.Validate(); err != nil {
			return nil, err
		}
		return config, nil
	}
	return nil, fmt.Errorf("did not find any pacmconfig")
}

func handleGlobal(section *parser.Section, config *Config) error {
	keys := section.RawKeys()
	for _, k := range keys {
		v := section.GetRaw(k)
		switch k {
		case "dir":
			config.OutputDir = v
		default:
			return fmt.Errorf("unexpected key %q in global section", k)
		}
	}
	return nil
}

func handleRecipe(section *parser.Section, config *Config) error {
	name := strings.Replace(section.Name(), "recipe ", "", 1)
	name = strings.TrimSpace(name)
	r := Recipe{
		Name:            name,
		AvailableArchOS: map[string]string{},
	}
	for _, k := range section.RawKeys() {
		v := section.GetRaw(k)
		switch k {
		case "url":
			r.URL = v
		default:
			if isValidOSArchPair(k) {
				r.AvailableArchOS[k] = v
			} else {
				return fmt.Errorf("%q is an unhandled arch and os", k)
			}
		}
	}
	config.Recipes = append(config.Recipes, r)
	return nil
}

func handlePackage(section *parser.Section, config *Config) error {
	n := section.Name()
	nameAndVersion := strings.Split(n, " ")
	if len(nameAndVersion) != 2 {
		return fmt.Errorf("was expecting a recipe name and version: [<recipe> <version>], recieved [%q]", n)
	}
	p := Package{
		RecipeName: nameAndVersion[0],
		Version:    nameAndVersion[1],
	}
	for _, k := range section.RawKeys() {
		v := section.GetRaw(k)
		switch k {
		case "active":
			p.Active = true
		case "executable":
			p.ExecutableName = v
		default:
			return fmt.Errorf("unexpected key %q in [%s]", k, n)
		}
	}
	config.Packages = append(config.Packages, p)
	return nil
}

func isValidOSArchPair(value string) bool {
	// TODO: Is this the best way to split the <os>_<arch> with a '_'??
	osAndArch := strings.Split(value, "_")
	if len(osAndArch) != 2 {
		return false
	}
	os := osAndArch[0]
	arch := osAndArch[1]

	switch arch {
	case "386", "amd64", "arm", "arm64", "ppc64":
		// Expected architectures
	default:
		return false
	}

	switch os {
	case "darwin", "linux", "dragonfly", "freebsd", "openbsd", "solaris", "netbsd":
		// Expected os
	default:
		return false
	}

	return true
}
