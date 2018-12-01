package main

import (
	"archive/tar"
	"bytes"
	"debug/elf"
	"debug/macho"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver"
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
		Arch: "x68_64",
		OS:   "linux",
	}

	recipes := []Recipe{terraform, protoc}
	if err := createRecipes(config, recipes); err != nil {
		log.Fatal(err)
	}
}

func createRecipes(config Config, recipes []Recipe) error {
	for _, r := range recipes {
		//resp, err := http.Get(url)
		url, err := generateURL(config, r)
		if err != nil {
			return err
		}
		fmt.Println(url)
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

func main2() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("one argument is required")
	}
	filename := args[0]

	f, err := archiver.ByExtension(filename)
	if err != nil {
		log.Fatal(err)
	}
	w, ok := f.(archiver.Walker)
	if !ok {
		log.Fatalf("unknown archive type for %q", filename)
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
