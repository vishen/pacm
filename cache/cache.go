package cache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

const cachePath = "~/.config/pacm/cache"

type Cache struct {
	path string

	// TODO: loop through cache directory and see what archives
	// are there.
	Archives map[string]bool
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

	return &Cache{path: cp, Archives: archives}, nil
}

func (c Cache) LoadArchive(filename string) ([]byte, error) {
	log.Printf("found cached archive %q\n", filename)
	// b, err := ioutil.ReadFile(filepath.Join(filename, archivePath))
	filename = filepath.Join(c.path, filename)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// NOTE: checksum verified here, and after we read response body;
	// before we write to disk.
	// TODO:
	/*ok, err := verifyChecksum(r, b)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("checksum failed")
	}*/

	return b, nil
}

func (c Cache) WriteArchive(filename string, data []byte) error {
	outPath := filepath.Join(c.path, filename)
	return ioutil.WriteFile(outPath, data, 0644)
}

func (c Cache) DownloadAndSave(url, filename string) ([]byte, error) {
	log.Printf("Getting %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Printf("%s -> %s -- reading body\n", url, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invalid response code for %s: %d", url, resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// TODO: verify checksum

	/*ok, err := verifyChecksum(r, b)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("checksum failed")
	}*/

	// Write the archive to a cache directory
	if err := c.WriteArchive(filename, b); err != nil {
		return nil, err
	}
	return b, nil
}
