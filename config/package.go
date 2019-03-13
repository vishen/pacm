package config

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/knq/ini/parser"
)

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

func (p Package) generateURL(arch, os string, r Recipe) (string, error) {
	tmpl, err := template.New("recipe-" + r.Name).Parse(r.URL)
	if err != nil {
		return "", err
	}

	arch, os = r.MappedArchOS(arch, os)
	td := struct {
		Version string
		OS      string
		Arch    string
	}{
		Version: p.Version,
		OS:      os,
		Arch:    arch,
	}
	fmt.Printf("TD=%#v\n", td)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", err
	}
	return buf.String(), nil
}
