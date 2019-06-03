package cache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/vishen/pacm/logging"
)

const cachePath = "~/.config/pacm/cache"

type Cache struct {
	path     string
	Archives map[string]bool
}

func LoadCache() (*Cache, error) {
	cp, err := homedir.Expand(cachePath)
	if err != nil {
		return nil, err
	}
	logging.PrintCommand("mkdirall %s 0755", cp)
	if err := os.MkdirAll(cp, 0755); err != nil {
		return nil, err
	}

	logging.PrintCommand("readdir %s", cp)
	files, err := ioutil.ReadDir(cp)
	if err != nil {
		return nil, err
	}

	archives := make(map[string]bool, len(files))
	for _, f := range files {
		archives[f.Name()] = true
	}

	return &Cache{path: cp, Archives: archives}, nil
}

func (c Cache) LoadArchive(filename string) ([]byte, error) {
	filename = filepath.Join(c.path, filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c Cache) WriteArchive(filename string, data []byte) error {
	outPath := filepath.Join(c.path, filename)
	logging.PrintCommand("writefile %s 0644", outPath)
	return ioutil.WriteFile(outPath, data, 0644)
}

func (c Cache) DownloadAndSave(url, filename string) ([]byte, error) {
	logging.PrintCommand("HTTP GET %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invalid response code for %s: %d", url, resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Write the archive to a cache directory
	if err := c.WriteArchive(filename, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (c Cache) ArchiveFullPath(archive string) string {
	return filepath.Join(c.path, archive)
}
