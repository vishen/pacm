package config

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"strings"
)

type Recipe struct {
	Name string
	URL  string

	// TODO: Consolidate these two fields? One implies the other.
	IsBinary   bool
	BinaryName string

	ExtractPaths []string

	AvailableArchOS map[string]string

	// NOT YET IMPLEMENTED
	ChecksumType string
	Checksum     string
}

func (r Recipe) MappedArchOS(arch, os string) (string, string) {
	// TODO: Fix the ordering, or better yet, make these concrete types.
	if m := r.AvailableArchOS[os+"_"+arch]; m != "" {
		mSplit := strings.Split(m, ":")
		return mSplit[1], mSplit[0]
	}
	return arch, os
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
