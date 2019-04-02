package releases

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/vishen/pacm/utils"
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
	sort.Slice(releases, func(i, j int) bool {
		ri := releases[i].TagName
		rj := releases[j].TagName
		return utils.SemvarIsBigger(ri, rj)
	})
	return releases, nil
}
