package utils

import (
	"debug/elf"
	"debug/macho"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

var (
	// Default paths to search the archive for when
	// looking for binaries.
	defaultExtractPaths = []string{
		"*",
		"bin/*",
		"*/*",
		"*/bin/*",
	}
)

func StringBool(str string) (bool, error) {
	switch strings.ToLower(str) {
	case "true", "t", "yes", "y":
		return true, nil
	case "false", "f", "no", "n":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a boolean value", str)
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

func ShouldExtract(path string, extractPaths []string) bool {
	// Filter out any empty paths.
	shouldExtractPaths := []string{}
	for _, p := range extractPaths {
		if p != "" {
			shouldExtractPaths = append(shouldExtractPaths, p)
		}
	}
	if len(shouldExtractPaths) == 0 {
		shouldExtractPaths = defaultExtractPaths
	}

	// TODO: There is likely a much better way to do this.
	pathSplit := strings.Split(path, "/")
	shouldExtract := false
	for _, wp := range shouldExtractPaths {
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

func SemvarIsBigger(semvar1, semvar2 string) bool {
	s1 := strings.SplitN(semvar1, ".", 3)
	s2 := strings.SplitN(semvar2, ".", 3)

	for i := 0; i < 3; i++ {
		s1i := s1[i]
		s2i := s2[i]
		s1in, s1OnlyNumber := extractFirstNumber(s1i)
		s2in, s2OnlyNumber := extractFirstNumber(s2i)
		if s1in == s2in {
			if s1OnlyNumber && !s2OnlyNumber {
				return true
			} else if s2OnlyNumber && !s1OnlyNumber {
				return false
			}
			continue
		}
		if s1in < s2in {
			return false
		} else if s1in > s2in {
			return true
		}
	}
	return semvar1 > semvar2
}

func extractFirstNumber(s string) (int, bool) {
	start := 0
	end := len(s)
	isDigit := false
	for i, c := range s {
		if !isDigit && unicode.IsDigit(c) {
			isDigit = true
			start = i
		} else if isDigit && !unicode.IsDigit(c) {
			end = i
			break
		}
	}
	val, _ := strconv.Atoi(s[start:end])
	return val, len(s) == len(s[start:end])
}
