package config

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knq/ini"
	"github.com/knq/ini/parser"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/h2non/filetype.v1"

	"github.com/vishen/pacm/cache"
	"github.com/vishen/pacm/utils"
)

const (
	defaultConfigPath = "~/.config/pacm/config"
)

func Load(path string) (*Config, error) {
	if path == "" {
		path = defaultConfigPath
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
	for _, s := range f.AllSections() {
		n := s.Name()
		switch {
		case n == "":
			if err := config.handleGlobal(s); err != nil {
				return nil, err
			}
			continue
		case strings.HasPrefix(n, "recipe "):
			if err := config.handleRecipe(s); err != nil {
				return nil, err
			}
		case strings.HasPrefix(n, "checksum "):
			// TODO: Handle checksums
		default:
			if err := config.handlePackage(s); err != nil {
				return nil, err
			}
		}
	}
	if err := config.populateCurrentlyInstalled(); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
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
			r.IsBinary = utils.StringBool(v)
		case "binary_name":
			r.BinaryName = v
		case "extract":
			r.ExtractPaths = strings.Split(v, ",")
		default:
			if utils.IsValidOSArchPair(k) {
				r.AvailableArchOS[k] = v
			} else {
				return fmt.Errorf("%q is an unhandled arch and os", k)
			}
		}
	}
	c.Recipes = append(c.Recipes, r)
	return nil
}

func (c *Config) handlePackage(section *parser.Section) error {
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
			p.Active = utils.StringBool(v)
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

func (c *Config) MakePackageActive(p *Package) error {
	for _, pkg := range c.Packages {
		if p.RecipeName == pkg.RecipeName {
			pkg.Active = true
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
	return nil
}

func (c *Config) WritePackage(p *Package, filename string, mode os.FileMode, data []byte) error {
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	// TODO: Clean this up!!!!!
	filenameWithVersion := p.FilenameWithVersion(filename)
	filenameWithVersionAndRecipe := fmt.Sprintf("%s_%s_%s", p.RecipeName, p.Version, filename)
	path := filepath.Join(c.OutputDir, "_pacm")
	// TODO: Move this to somewhere more useful.
	os.MkdirAll(path, 0755)

	outPath := filepath.Join(path, filenameWithVersionAndRecipe)
	// Need to symlink absolute path for Macos, possibly the same for linux?
	outPath, _ = filepath.Abs(outPath)

	log.Printf("writing to %s (filenameWithVersion=%s)\n", outPath, filenameWithVersion)

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
	// TODO: Cleanup
	// symlinkPath := filepath.Join(c.OutputDir, symlink)
	symlinkPath := symlink
	// TODO: Cleanup
	filePath := filepath.Join(c.OutputDir, filename)
	os.Remove(filePath)
	// OldName, NewName
	if err := os.Symlink(symlinkPath, filePath); err != nil {
		return err
	}
	return nil
}

func (c *Config) CreatePackages(currentArch, currentOS string) error {
	cache, err := cache.LoadCache()
	if err != nil {
		return err
	}
	for _, p := range c.Packages {
		if err := c.CreatePackage(currentArch, currentOS, p, cache); err != nil {
			return errors.Wrapf(err, "unable to create package %s@%s", p.RecipeName, p.Version)
		}
	}
	return nil
}

func (c *Config) CreatePackage(currentArch, currentOS string, p *Package, cache *cache.Cache) error {
	r := c.RecipeForPackage(p)
	var b []byte
	archivePath := fmt.Sprintf("%s_%s_%s-%s", p.RecipeName, p.Version, currentArch, currentOS)
	// If we have don't an archive on disk.
	if ok := cache.Archives[archivePath]; !ok {
		url, err := p.generateURL(currentArch, currentOS, r)
		if err != nil {
			return err
		}
		b, err = cache.DownloadAndSave(url, archivePath)
		if err != nil {
			return err
		}
	} else {
		var err error
		b, err = cache.LoadArchive(archivePath)
		if err != nil {
			return err
		}
	}

	// If the recipe is a binary then we just need to
	// save it and we are done.
	if r.IsBinary {
		// TODO: Make the permissions configurable
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

	// TODO: make this a function or something.
	//dir := filepath.Join(c.OutputDir, "_pacm")
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
