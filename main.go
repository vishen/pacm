package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/h2non/filetype"
)

var (
	// Default paths to search the archive for when
	// looking for binaries.
	wantedPaths = []string{
		"*",
		"bin/*",
		"*/*",
		"*/bin/*",
	}
)

func main() {
	config, err := Load("./pacmconfig")
	if err != nil {
		log.Fatal(err)
	}
	if err := createRecipes(config); err != nil {
		log.Fatal(err)
	}
}

func createRecipes(config *Config) error {
	for _, r := range config.Recipes {
		url, err := generateURL(config, r)
		if err != nil {
			return err
		}
		log.Printf("Getting %s...\n", url)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("%s -> %s -- reading body\n", url, resp.Status)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		ok, err := verifyChecksum(r, b)
		if err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("checksum failed")
		}

		log.Printf("Getting archive type\n")
		typ, err := filetype.Archive(b)
		if err != nil {
			return err
		}
		log.Printf("Found archive type=%#v\n", typ)

		buf := bytes.NewReader(b)
		if typ.Extension == "zip" {
			log.Printf("Zip reader\n")
			rdr, err := zip.NewReader(buf, resp.ContentLength)
			if err != nil {
				return err
			}
			for _, f := range rdr.File {
				log.Printf("Contents of %s\n", f.Name)
				if !shouldExtract(f.Name) {
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
				isExec := isExecutable(bytes.NewReader(b))
				if !isExec {
					continue
				}
				if err := writeFile(config, r, f.FileInfo(), b); err != nil {
					return err
				}
			}
		} else if typ.Extension == "gz" {
			log.Printf("GZ reader\n")
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
				if !shouldExtract(hdr.Name) {
					continue
				}
				b, err := ioutil.ReadAll(rdr)
				if err != nil {
					return err
				}
				isExec := isExecutable(bytes.NewReader(b))
				if !isExec {
					continue
				}
				if err := writeFile(config, r, hdr.FileInfo(), b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func verifyChecksum(r Recipe, checksumBytes []byte) (bool, error) {
	// Ignore recipes without checksums.
	// TODO(vishen): Should we do this? Or should we force checksums
	// unless a flag is passed to override this, --ignore-checksum.
	if r.Checksum == "" || r.ChecksumType == "" {
		return true, nil
	}
	var checksum string
	switch ct := r.ChecksumType; ct {
	case "md5":
		checksum = fmt.Sprintf("%x", md5.Sum(checksumBytes))
	case "sha1":
		checksum = fmt.Sprintf("%x", sha1.Sum(checksumBytes))
	case "sha256":
		checksum = fmt.Sprintf("%x", sha256.Sum256(checksumBytes))
	default:
		return false, fmt.Errorf("%s currently not handled", ct)
	}
	return checksum == r.Checksum, nil
}

func writeFile(c *Config, r Recipe, fi os.FileInfo, data []byte) error {
	outPath := c.OutputDir
	filename := r.FilenameWithVersion(fi.Name())
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
	symlinkPath := filepath.Join(outPath, fi.Name())
	os.Remove(symlinkPath)
	// I don't quite understand why it is like this...? The cwd is
	// not the 'outPath', so why...
	// TODO: Will this cause issues on other systems? Test out on mac.
	return os.Symlink(filename, symlinkPath)
}

func generateURL(config *Config, recipe Recipe) (string, error) {
	tmpl, err := template.New("recipe-" + recipe.Name).Parse(recipe.URL)
	if err != nil {
		return "", err
	}

	td := struct {
		Recipe Recipe
		Config Config
	}{
		Recipe: recipe,
		Config: *config,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func shouldExtract(path string) bool {
	// TODO: There is likely a much better way to do this.
	pathSplit := strings.Split(path, "/")
	shouldExtract := false
	for _, wp := range wantedPaths {
		wpSplit := strings.Split(wp, "/")
		if len(pathSplit) != len(wpSplit) {
			continue
		}

		shouldExtract = true
		for i, wpi := range wpSplit {
			if wpi != "*" && wpi != pathSplit[i] {
				shouldExtract = false
				break
			}
		}
		if shouldExtract {
			break
		}
	}
	return shouldExtract
}

func isExecutable(r io.ReaderAt) bool {
	currentArch := runtime.GOARCH
	currentOS := runtime.GOOS

	switch currentOS {
	case "darwin":
		m, err := macho.NewFile(r)
		if err != nil {
			return false
		}

		if m.Type != macho.TypeExec {
			return false
		}
		switch currentArch {
		case "386":
			return m.Cpu == macho.Cpu386
		case "amd64":
			return m.Cpu == macho.CpuAmd64
		case "arm":
			return m.Cpu == macho.CpuArm
		case "arm64":
			return m.Cpu == macho.CpuArm64
		case "ppc64":
			return m.Cpu == macho.CpuPpc64
		}
	case "linux", "dragonfly", "freebsd", "openbsd", "solaris", "netbsd":
		e, err := elf.NewFile(r)
		if err != nil {
			return false
		}
		// Is is an executable type
		if e.Type != elf.ET_REL && e.Type != elf.ET_EXEC {
			return false
		}

		switch currentArch {
		case "386":
			return e.Machine == elf.EM_386
		case "amd64":
			return e.Machine == elf.EM_X86_64
		case "arm":
			return e.Machine == elf.EM_ARM
		case "arm64":
			return e.Machine == elf.EM_AARCH64
		case "mips":
			return e.Machine == elf.EM_MIPS
		case "ppc64":
			return e.Machine == elf.EM_PPC64
		}
	default:
		return false
	}
	return false
}
