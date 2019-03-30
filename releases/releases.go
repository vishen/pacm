package releases

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// https://api.github.com/repos/kubernetes/kubernetes/releases
	githubReleaseURL = "https://api.github.com/repos/%s/%s/releases"
)

type GithubRelease struct {
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Body            string    `json:"body"`
}

func GithubReleases(url string) ([]GithubRelease, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	releases := []GithubRelease{}
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return orderBySemver(releases), nil
}

// TODO: move into common location. Has been copied and modified
// in cmd/status.go
func orderBySemver(releases []GithubRelease) []GithubRelease {
	sort.Slice(releases, func(i, j int) bool {
		ri := releases[i]
		rj := releases[j]

		riTagSplit := strings.SplitN(ri.TagName, ".", 3)
		rjTagSplit := strings.SplitN(rj.TagName, ".", 3)
		for i := 0; i < 3; i++ {
			if riTagSplit[i] < rjTagSplit[i] {
				return false
			} else if riTagSplit[i] > rjTagSplit[i] {
				return true
			}
		}
		return true
	})
	return releases
}

func extractFirstNumber(s string) int {
	start := 0
	end := 0
	isDigit := false
	for i, c := range s {
		if unicode.IsDigit(c) && !isDigit {
			isDigit = true
			start = i
		} else if isDigit {
			end = i
			break
		}
	}
	fmt.Println(s, s[start:end])
	val, _ := strconv.Atoi(s[start:end])
	return val
}
