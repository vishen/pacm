package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"debug/elf"
	"debug/macho"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/h2non/filetype"
)

/*
	- go-chromecast
	- go/bin/go
	- go/bin/fmt
	- go/doc
*/

var (
	wantedPaths = []string{
		"*",
		"bin/*",
		"*/*",
		"*/bin/*",
	}

	outPath = "./extracted"
	// TODO: Come up with a better alternative
	filePermissions = os.FileMode(0777)
)

// TODO(vishen): Should handle zip at least
func archivePath(header interface{}) string {
	if h, ok := header.(*tar.Header); ok {
		return h.Name
	}
	return ""
}

func main() {
	config := Config{
		Arch:    "x86_64",
		ArchAlt: "amd64",
		OS:      "linux",
	}

	recipes := []Recipe{terraform, protoc}
	if err := createRecipes(config, recipes); err != nil {
		log.Fatal(err)
	}
}

func createRecipes(config Config, recipes []Recipe) error {
	for _, r := range recipes {
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

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
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
			log.Printf("Zip files\n")
			for _, f := range rdr.File {
				fmt.Printf("Contents of %s\n", f.Name)
				if !shouldExtract(f.Name) {
					continue
				}
				fmt.Println("can extract")
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
				fmt.Println("Is executable")
			}
		}
	}
	return nil
}

func generateURL(config Config, recipe Recipe) (string, error) {
	tmpl, err := template.New("recipe-" + recipe.Name).Parse(recipe.URL)
	if err != nil {
		return "", err
	}

	td := struct {
		Recipe Recipe
		Config Config
	}{
		Recipe: recipe,
		Config: config,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", err
	}
	return buf.String(), nil
}

/*
func extractFromArchive(r io.ReaderCloser) error {
	unarc, err := archiver.ByHeader(r)
	if err != nil {
		return err
	}
	f, err := archiver.ByExtension(filename)
	if err != nil {
		log.Fatal(err)
	}
	w, ok := f.(archiver.Walker)
	if !ok {
		log.Fatalf("unknown archive type for %q", filename)
	}

	w, ok := unarc.(archiver.Walker)
	if !ok {
		return fmt.Errorf("unable to type convert to archiver.Walker")
	}

	err = w.Walk(filename, func(f archiver.File) error {
		isDir := f.IsDir()
		if isDir {
			return nil
		}

		path := archivePath(f.Header)
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil
		}

		if !shouldExtract(path) {
			return nil
		}

		isExec := isExecutable(bytes.NewReader(b))
		if !isExec {
			return nil
		}

		// TODO: This should prefix executable with a name or something else
		outFilename := filepath.Join(outPath, f.Name())

		// change that. Likely using a different function.
		// This will overwrite the file, but not the file permissions, so we
		// need to manually set them afterwars.
		if err := ioutil.WriteFile(outFilename, b, f.Mode()); err != nil {
			return err
		}
		return os.Chmod(outFilename, f.Mode())
	})
	if err != nil {
		log.Fatal(err)
	}
}
*/

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
