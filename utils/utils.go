package utils

import (
	"debug/elf"
	"debug/macho"
	"io"
	"runtime"
	"strings"
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

func StringBool(str string) bool {
	switch strings.ToLower(str) {
	case "true", "t", "yes", "y":
		return true
	}
	return false
}

func IsValidOSArchPair(value string) bool {
	// TODO: Is this the best way to split the <os>_<arch> with a '_'??
	osAndArch := strings.Split(value, "_")
	if len(osAndArch) != 2 {
		return false
	}
	os := osAndArch[0]
	arch := osAndArch[1]

	switch arch {
	case "386", "amd64", "arm", "arm64", "ppc64":
		// Expected architectures
	default:
		return false
	}

	switch os {
	case "darwin", "linux", "dragonfly", "freebsd", "openbsd", "solaris", "netbsd":
		// Expected os
	default:
		return false
	}

	return true
}

func ShouldExtract(path string) bool {
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

func IsExecutable(r io.ReaderAt) bool {
	currentArch := runtime.GOARCH
	currentOS := runtime.GOOS

	switch currentOS {
	case "darwin":
		m, err := macho.NewFile(r)
		if err != nil {
			// TODO: Log errors?
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
		// TODO: This should return an error about a not-supported architecture.
		return false
	}
	return false
}