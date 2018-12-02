package main

import (
	"fmt"
	"os"

	"github.com/knq/ini"
	homedir "github.com/mitchellh/go-homedir"
)

var possibleConfigPaths = []string{
	"~/.config/pacm/config",
	"~/pacmconfig",
}

type Recipe struct {
	Arch           string
	OS             string
	Name           string
	ExecutableName string
	URL            string
	Version        string
	Checksum       string
	ChecksumType   string
}

func (r Recipe) LinkName() string {
	if r.ExecutableName != "" {
		return r.ExecutableName
	}
	return r.Name
}

func (r Recipe) FilenameWithVersion(filename string) string {
	// TODO: This should be configurable
	return fmt.Sprintf("%s-%s", filename, r.Version)
}

type Config struct {
	Arch      string
	OS        string
	OutputDir string

	Recipes []Recipe
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
		fmt.Println(cp)
		reader, err := os.Open(cp)
		if err != nil {
			continue
		}
		defer reader.Close()
		f, err := ini.Load(reader)
		if err != nil {
			continue
		}
		config := Config{}
		// It is possible for their to be a global section which is just
		// for 'config' variables.
		//recipes := make([]Recipe, len(f.AllSections())-1, len(f.AllSections()))
		recipes := []Recipe{}
		for _, s := range f.AllSections() {
			// The global section
			if s.Name() == "" {
				keys := s.RawKeys()
				for _, k := range keys {
					v := s.GetRaw(k)
					switch k {
					case "dir":
						config.OutputDir = v
					default:
						// TODO(vishen): Add these as extra keyvalues on the
						// recipe that can be access somehow from go templates.
						// For now just return an error
						return nil, fmt.Errorf("unexpected key %q in global section", k)
					}
				}
				continue
			}
			r := Recipe{Name: s.Name()}
			keys := s.RawKeys()
			for _, k := range keys {
				v := s.GetRaw(k)
				switch k {
				case "os":
					r.OS = v
				case "arch":
					r.Arch = v
				case "version":
					r.Version = v
				case "url":
					r.URL = v
				case "checksum_md5":
					r.ChecksumType = "md5"
					r.Checksum = v
				case "checksum_sha1":
					r.ChecksumType = "sha1"
					r.Checksum = v
				case "checksum_sha256":
					r.ChecksumType = "sha256"
					r.Checksum = v
				default:
					// TODO(vishen): Add these as extra keyvalues on the
					// recipe that can be access somehow from go templates.
					// For now just return an error
					return nil, fmt.Errorf("unexpected key %q in %s section", k, r.Name)
				}
			}
			recipes = append(recipes, r)
		}
		config.Recipes = recipes
		// TODO(vishen): Validate the config and recipes.
		return &config, nil
	}
	return nil, fmt.Errorf("did not find any pacmconfig")
}
