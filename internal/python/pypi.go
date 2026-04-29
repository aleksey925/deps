package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/aleksey925/deps/internal/python/semver"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

type pypiResponse struct {
	Info     pypiInfo                   `json:"info"`
	Releases map[string]json.RawMessage `json:"releases"`
}

type pypiInfo struct {
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Version    string `json:"version"`
	License    string `json:"license"`
	HomePage   string `json:"home_page"`
	ProjectURL string `json:"project_url"`
	Author     string `json:"author"`
	RequiresPy string `json:"requires_python"`
}

type PackageInfo struct {
	Name       string
	Summary    string
	Version    string
	License    string
	HomePage   string
	Author     string
	RequiresPy string
}

func FetchPackageInfo(packageName string) (*PackageInfo, error) {
	data, err := fetchPyPI(packageName)
	if err != nil {
		return nil, err
	}

	home := data.Info.HomePage
	if home == "" {
		home = data.Info.ProjectURL
	}

	return &PackageInfo{
		Name:       data.Info.Name,
		Summary:    data.Info.Summary,
		Version:    data.Info.Version,
		License:    data.Info.License,
		HomePage:   home,
		Author:     data.Info.Author,
		RequiresPy: data.Info.RequiresPy,
	}, nil
}

func fetchPyPI(packageName string) (*pypiResponse, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching PyPI data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI returned status %d", resp.StatusCode)
	}

	var data pypiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding PyPI response: %w", err)
	}

	return &data, nil
}

func stableVersions(data *pypiResponse) []string {
	versions := make([]string, 0, len(data.Releases))
	for v := range data.Releases {
		if semver.IsStable(v) {
			versions = append(versions, v)
		}
	}
	return versions
}

func FetchLatestVersion(packageName string) (string, error) {
	data, err := fetchPyPI(packageName)
	if err != nil {
		return "", err
	}

	versions := stableVersions(data)
	if len(versions) == 0 {
		return "", errors.New("no stable versions found")
	}

	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	return versions[len(versions)-1], nil
}

func FetchVersions(packageName string) ([]string, error) {
	data, err := fetchPyPI(packageName)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(data.Releases))
	for v := range data.Releases {
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) > 0
	})

	return versions, nil
}
