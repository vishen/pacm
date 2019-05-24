package config

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	getter "github.com/hashicorp/go-getter"
	"github.com/knq/ini"
	"github.com/knq/ini/parser"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/h2non/filetype.v1"

	pacmcache "github.com/vishen/pacm/cache"
	"github.com/vishen/pacm/utils"
)

const (
	DefaultConfigPath = "~/.config/pacm/config"
)

var cache *pacmcache.Cache

func init() {
	var err error
	cache, err = pacmcache.LoadCache()
	if err != nil {
		log.Fatalf("unable to load cache: %v", err)
	}
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	configPath, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	f, err := ini.Load(reader)
	if err != nil {
		return nil, err
	}
	config := &Config{
		iniFile:  f,
		filename: configPath,
		Recipes:  []Recipe{},
		Packages: []*Package{},
	}

	// NOTE: This needs to go before the parsing of the config
	// file since it will add recipes etc that can be overwritten.
	if err := config.downloadRemoteRecipes(); err != nil {
		return nil, err
	}
	if err := config.parseIniFile(config.iniFile); err != nil {
		return nil, err
	}
	if err := config.populateCurrentlyInstalled(); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) parseIniFile(f *ini.File) error {
	for _, s := range f.AllSections() {
		n := s.Name()
		switch {
		case n == "":
			if err := c.handleGlobal(s); err != nil {
				return err
			}
			continue
		case strings.HasPrefix(n, "recipe "):
			if err := c.handleRecipe(s); err != nil {
				return err
			}
		case strings.HasPrefix(n, "checksum "):
			// TODO: Handle checksums
		default:
			if err := c.handlePackage(s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) downloadRemoteRecipes() error {
	dir, err := homedir.Expand("~/.config/pacm/remote_recipes")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// TODO: Make configurable and allow multiple remotes
	remotes := []string{"github.com/vishen/pacm-recipes"}

	for _, remote := range remotes {
		remoteFolder := filepath.Join(dir, strings.Replace(remote, "/", "_", -1))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		// Build the client
		client := &getter.Client{
			Ctx:  ctx,
			Src:  remote,
			Dst:  remoteFolder,
			Pwd:  ".",
			Mode: getter.ClientModeAny,
		}
		if err := client.Get(); err != nil {
			return err
		}
		if err := c.handleRecipeFiles(remoteFolder); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) handleRecipeFiles(folder string) error {
	// Loop through the downloaded folder and look for 'recipe.ini' files
	// and add them to the config as recipes.

	// TODO: Probably a bad way to do this; if a folder has thousands of
	// nested folders this will loop through them all...
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			if err := c.handleRecipeFiles(filepath.Join(folder, f.Name())); err != nil {
				return err
			}
		} else if f.Name() == "recipe.ini" {
			reader, err := os.Open(filepath.Join(folder, f.Name()))
			if err != nil {
				return err
			}
			defer reader.Close()
			f, err := ini.Load(reader)
			if err != nil {
				return err
			}
			if err := c.parseIniFile(f); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) handleGlobal(section *parser.Section) error {
	keys := section.RawKeys()
	for _, k := range keys {
		v := section.GetRaw(k)
		switch k {
		case "dir":
			c.OutputDir = v
		default:
			return fmt.Errorf("unexpected key %q in global section", k)
		}
	}
	return nil
}

func (c *Config) handleRecipe(section *parser.Section) error {
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
		case "binary":
			var err error
			r.IsBinary, err = utils.StringBool(v)
			if err != nil {
				return fmt.Errorf("unable to extract boolean value from [recipe %s.%s = %q]: %v", name, k, v, err)
			}
		case "binary_name":
			r.BinaryName = v
		case "extract":
			r.ExtractPaths = strings.Split(v, ",")
		case "releases_github":
			r.ReleasesGithub = v
		default:
			if utils.IsValidOSArchPair(k) {
				r.AvailableArchOS[k] = v
			} else {
				return fmt.Errorf("%q is an unhandled arch and os", k)
			}
		}
	}
	// Don't allow duplicate recipes. Replace with any newer recipes.
	replaced := false
	for i, recipe := range c.Recipes {
		if recipe.Name == r.Name {
			c.Recipes[i] = r
			replaced = true
			break
		}
	}
	if !replaced {
		c.Recipes = append(c.Recipes, r)
	}
	return nil
}

func (c *Config) handlePackage(section *parser.Section) error {
	n := section.Name()
	nameAndVersion := strings.Split(n, "@")
	if len(nameAndVersion) != 2 {
		return fmt.Errorf("was expecting a recipe name and version: [<recipe>@<version>], recieved [%q]", n)
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
			var err error
			p.Active, err = utils.StringBool(v)
			if err != nil {
				return fmt.Errorf("unable to extract boolean value from [recipe %s.%s = %q]: %v", n, k, v, err)
			}
		case "executable":
			p.ExecutableName = v
		default:
			return fmt.Errorf("unexpected key %q in [%s]", k, n)
		}
	}
	c.Packages = append(c.Packages, p)
	return nil
}

type Installed struct {
	Filename            string
	AbsolutePath        string
	SymlinkAbsolutePath string
	ModTime             time.Time
}

type Config struct {
	iniFile  *ini.File
	filename string

	OutputDir string

	Recipes  []Recipe
	Packages []*Package

	CurrentlyInstalled []Installed
}

func (c *Config) AddPackage(arch, OS, recipeName, version string) error {
	// Check if the package is already installed.
	for _, p := range c.Packages {
		if p.RecipeName == recipeName && p.Version == version {
			return fmt.Errorf("%s@%s is already installed", recipeName, version)
		}
	}

	// Check to see if the recipe exists.
	var recipe Recipe
	for _, r := range c.Recipes {
		if r.Name == recipeName {
			recipe = r
			break
		}
	}
	if recipe.Name == "" {
		return fmt.Errorf("unknown recipe %q", recipeName)
	}

	if _, err := c.getCachedOrDownload(arch, OS, recipe, version); err != nil {
		return err
	}

	// TODO: move this to a common function and all other occurances.
	recipeAndVersion := fmt.Sprintf("%s@%s", recipeName, version)

	// TODO: This will add the section to the end of the list,
	// and the section will outputted last in the file... Maybe
	// for github.com/knq/ini and add ability to add section to start?
	section := c.iniFile.AddSection(recipeAndVersion)
	pkg := &Package{
		iniSection: section,
		RecipeName: recipeName,
		Version:    version,
	}

	c.Packages = append(c.Packages, pkg)
	return c.MakePackageActive(pkg)
}

func (c *Config) MakePackageActive(p *Package) error {
	for _, pkg := range c.Packages {
		if p.RecipeName == pkg.RecipeName {
			pkg.Active = false
			pkg.iniSection.RemoveKey("active")
		}
	}
	p.Active = true
	p.iniSection.SetKey("active", "true")
	if err := c.iniFile.Write(c.filename); err != nil {
		return errors.Wrap(err, "unable to save config file")
	}
	return nil
}

func (c *Config) RecipeForPackage(p *Package) Recipe {
	for _, r := range c.Recipes {
		if r.Name == p.RecipeName {
			return r
		}
	}
	// Should not be possible.
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
	for _, r := range c.Recipes {
		if r.IsBinary && r.BinaryName == "" {
			return fmt.Errorf("recipe %q is marked binary but missing 'binary_name' field", r.Name)
		}
	}
	return nil
}

func (c *Config) WritePackage(p *Package, filename string, mode os.FileMode, data []byte) error {
	filenameWithVersion := p.FilenameWithVersion(filename)
	filenameWithVersionAndRecipe := fmt.Sprintf("%s_%s_%s", p.RecipeName, p.Version, filename)
	path := filepath.Join(c.OutputDir, "_pacm")
	os.MkdirAll(path, 0755)

	outPath := filepath.Join(path, filenameWithVersionAndRecipe)
	// Need to symlink absolute path for Macos, possibly the same for linux?
	outPath, _ = filepath.Abs(outPath)

	// This will overwrite the file, but not the file permissions, so we
	// need to manually set them afterwards.
	if err := ioutil.WriteFile(outPath, data, mode); err != nil {
		return err
	}
	if err := os.Chmod(outPath, mode); err != nil {
		return err
	}

	if err := c.SymlinkFile(outPath, filenameWithVersion); err != nil {
		return err
	}

	if p.Active {
		if err := c.SymlinkFile(outPath, filename); err != nil {
			return err
		}
	}
	if p.ExecutableName != "" {
		if err := c.SymlinkFile(outPath, p.ExecutableName); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) SymlinkFile(symlink, filename string) error {
	symlinkPath := symlink
	filePath := filepath.Join(c.OutputDir, filename)
	os.Remove(filePath)
	// OldName, NewName
	if err := os.Symlink(symlinkPath, filePath); err != nil {
		return err
	}
	return nil
}

func (c *Config) CreatePackages(arch, OS string) error {
	for _, i := range c.CurrentlyInstalled {
		os.Remove(i.AbsolutePath)
	}
	os.RemoveAll(filepath.Join(c.OutputDir, "_pacm"))
	for _, p := range c.Packages {
		if err := c.CreatePackage(arch, OS, p); err != nil {
			return errors.Wrapf(err, "unable to create package %s@%s", p.RecipeName, p.Version)
		}
	}
	return nil
}

func (c *Config) generateArchivePath(arch, OS string, r Recipe, versionName string) string {
	return fmt.Sprintf("%s_%s_%s-%s", r.Name, versionName, arch, OS)
}

func (c *Config) getCachedOrDownload(arch, OS string, r Recipe, packageVersion string) ([]byte, error) {
	var b []byte
	archivePath := c.generateArchivePath(arch, OS, r, packageVersion)
	// If we have don't an archive on disk, download and save to disk.
	if ok := cache.Archives[archivePath]; !ok {
		url, err := r.generateURL(arch, OS, packageVersion)
		if err != nil {
			return nil, err
		}
		b, err = cache.DownloadAndSave(url, archivePath)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		b, err = cache.LoadArchive(archivePath)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (c *Config) CreatePackage(arch, OS string, p *Package) error {
	r := c.RecipeForPackage(p)
	b, err := c.getCachedOrDownload(arch, OS, r, p.Version)
	if err != nil {
		return err
	}

	// If the recipe is a binary then we just need to
	// save it and we are done.
	if r.IsBinary {
		// TODO: Make the permissions configurable?
		if err := c.WritePackage(p, r.BinaryName, 0755, b); err != nil {
			return err
		}
		return nil
	}
	typ, err := filetype.Archive(b)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(b)
	switch t := typ.Extension; t {
	case "zip":
		if err := c.writeZIP(r, p, buf, int64(len(b))); err != nil {
			return err
		}
	case "gz":
		if err := c.writeGZ(r, p, buf); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported archive %s", t)
	}
	return nil
}

func (c *Config) writeGZ(r Recipe, p *Package, buf io.Reader) error {
	gzr, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	rdr := tar.NewReader(gzr)
	for {
		hdr, err := rdr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !utils.ShouldExtract(hdr.Name, r.ExtractPaths) {
			continue
		}
		b, err := ioutil.ReadAll(rdr)
		if err != nil {
			return err
		}
		isExec := utils.IsExecutable(bytes.NewReader(b))
		if !isExec {
			continue
		}
		fi := hdr.FileInfo()
		if err := c.WritePackage(p, fi.Name(), fi.Mode(), b); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) writeZIP(r Recipe, p *Package, buf io.ReaderAt, bufLen int64) error {
	rdr, err := zip.NewReader(buf, bufLen)
	if err != nil {
		return err
	}
	for _, f := range rdr.File {
		if !utils.ShouldExtract(f.Name, r.ExtractPaths) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		b, err := ioutil.ReadAll(rc)
		if err != nil {
			return err
		}
		isExec := utils.IsExecutable(bytes.NewReader(b))
		if !isExec {
			continue
		}
		fi := f.FileInfo()
		if err := c.WritePackage(p, fi.Name(), fi.Mode(), b); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) populateCurrentlyInstalled() error {
	if len(c.Packages) == 0 {
		return fmt.Errorf("no installed packages")
	}

	dir := c.OutputDir
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	installs := make([]Installed, 0, len(files))
	for _, f := range files {
		if f.Mode()&os.ModeSymlink != os.ModeSymlink {
			continue
		}
		name := f.Name()
		symlink, err := os.Readlink(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		// TODO: Is there a better way to do this, need to check if the path
		// the symlink is pointing to one "managed" by pacm.
		if !strings.Contains(symlink, filepath.Join(c.OutputDir, "_pacm")) {
			continue
		}
		abs, err := filepath.Abs(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		installs = append(installs, Installed{
			Filename:            name,
			AbsolutePath:        abs,
			SymlinkAbsolutePath: symlink,
			ModTime:             f.ModTime(),
		})
	}
	c.CurrentlyInstalled = installs
	return nil
}
