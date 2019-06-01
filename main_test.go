package main

import (
	"os/exec"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScripts(t *testing.T) {
	t.Parallel()
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(e *testscript.Env) error {
			cmd := exec.Command("go", "install")
			if _, err := cmd.Output(); err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					t.Fatalf("%s", e.Stderr)
				}
				t.Fatalf("%v", err)
			}
			return nil
		},
	})
}
