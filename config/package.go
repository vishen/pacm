package config

import (
	"fmt"

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
