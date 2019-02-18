package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knq/ini"
	"github.com/knq/ini/parser"
	homedir "github.com/mitchellh/go-homedir"
)

const cachePath = "~/.config/pacm/cache"

var possibleConfigPaths = []string{
	"~/.config/pacm/config",
}

type Cache struct {
	path string

	// TODO: loop through cache directory and see what archives
	// are there.
	archives map[string]bool
}

func LoadCache() (*Cache, error) {
	cp, err := homedir.Expand(cachePath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cp, 0755); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(cp)
	if err != nil {
		return nil, err
	}

	archives := make(map[string]bool, len(files))
	for _, f := range files {
		archives[f.Name()] = true
	}

	return &Cache{path: cp, archives: archives}, nil
}

func (c Cache) WriteArchive(filename string, data []byte) error {
	outPath := filepath.Join(c.path, filename)
	return ioutil.WriteFile(outPath, data, 0644)
}

type Package struct {
	iniSection     *parser.Section
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
	iniFile  *ini.File
	filename string

	OutputDir string

	Recipes  []Recipe
	Packages []*Package

	CurrentlyInstalled []Installed
}

func (c *Config) MakePackageActive(pkg *Package) {
	for _, p := range c.Packages {
		if p.RecipeName == pkg.RecipeName {
			p.Active = true
			p.iniSection.RemoveKey("active")
		}
	}
	pkg.Active = true
	pkg.iniSection.SetKey("active", "true")
	if err := c.iniFile.Write(c.filename); err != nil {
		log.Fatal(err)
	}

}

func (c *Config) RecipeForPackage(p *Package) Recipe {
	for _, r := range c.Recipes {
		if r.Name == p.RecipeName {
			return r
		}
	}
	panic(fmt.Sprintf("no recipe %q found for package %s-%s", p.RecipeName, p.RecipeName, p.Version))
}

func (c *Config) Validate() error {
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

func (c *Config) WriteFile(p *Package, fi os.FileInfo, data []byte) error {
	outPath := c.OutputDir
	filename := p.FilenameWithVersion(fi.Name())
	// TODO: Delete
	outFilename := filepath.Join(outPath, filename)
	log.Printf("writing to %s...\n", outFilename)
	// This will overwrite the file, but not the file permissions, so we
	// need to manually set them afterwars.
	if err := ioutil.WriteFile(outFilename, data, fi.Mode()); err != nil {
		return err
	}
	if err := os.Chmod(outFilename, fi.Mode()); err != nil {
		return err
	}
	// If this is an active package, then symlink it.
	if p.Active {
		symlinkPath := filepath.Join(outPath, fi.Name())
		// First remove the symlink path if it exists.
		os.Remove(symlinkPath)
		// I don't quite understand why it is like this...? The cwd is
		// not the 'outPath', so why...
		// TODO: Will this cause issues on other systems? Test out on mac.
		if err := os.Symlink(filename, symlinkPath); err != nil {
			return err
		}
	}
	if p.ExecutableName != "" {
		symlinkPath := filepath.Join(outPath, p.ExecutableName)
		// First remove the symlink path if it exists.
		os.Remove(symlinkPath)
		// I don't quite understand why it is like this...? The cwd is
		// not the 'outPath', so why...
		// TODO: Will this cause issues on other systems? Test out on mac.
		if err := os.Symlink(filename, symlinkPath); err != nil {
			return err
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
			iniFile:  f,
			filename: cp,
			Recipes:  []Recipe{},
			Packages: []*Package{},
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
	p := &Package{
		iniSection: section,
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
