package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ErrIndexNotCached = errors.New("index not cached")

const (
	pypiSimpleURL = "https://pypi.org/simple/"
	indexMaxAge   = 7 * 24 * time.Hour
	cacheDir      = "deps"
	cacheFileName = "pypi_index.json"
)

type PackageIndex struct {
	Packages  []string  `json:"packages"`
	UpdatedAt time.Time `json:"updated_at"`
}

var linkRe = regexp.MustCompile(`<a[^>]+href="/simple/([^/]+)/"`)

func cacheFilePath() (string, error) {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
		cacheHome = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheHome, cacheDir, cacheFileName), nil
}

func LoadIndex() (*PackageIndex, error) {
	path, err := cacheFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrIndexNotCached
		}
		return nil, fmt.Errorf("reading index cache: %w", err)
	}

	var idx PackageIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index cache: %w", err)
	}

	return &idx, nil
}

func (idx *PackageIndex) IsExpired() bool {
	return time.Since(idx.UpdatedAt) > indexMaxAge
}

func (idx *PackageIndex) Age() time.Duration {
	return time.Since(idx.UpdatedAt)
}

func FetchIndex() (*PackageIndex, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pypiSimpleURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "text/html")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching PyPI index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading PyPI index: %w", err)
	}

	matches := linkRe.FindAllSubmatch(body, -1)
	packages := make([]string, 0, len(matches))
	for _, m := range matches {
		packages = append(packages, string(m[1]))
	}

	idx := &PackageIndex{
		Packages:  packages,
		UpdatedAt: time.Now(),
	}

	if err := saveIndex(idx); err != nil {
		return idx, fmt.Errorf("saving index cache: %w", err)
	}

	return idx, nil
}

func saveIndex(idx *PackageIndex) error {
	path, err := cacheFilePath()
	if err != nil {
		return err
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return fmt.Errorf("creating cache dir: %w", mkdirErr)
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		return fmt.Errorf("writing index cache: %w", writeErr)
	}

	return nil
}

func SearchIndex(idx *PackageIndex, query string) []string {
	if idx == nil || query == "" {
		return nil
	}

	query = normalizeName(query)

	var exact, prefix, contains []string
	for _, pkg := range idx.Packages {
		switch {
		case pkg == query:
			exact = append(exact, pkg)
		case strings.HasPrefix(pkg, query):
			prefix = append(prefix, pkg)
		case strings.Contains(pkg, query):
			contains = append(contains, pkg)
		}
	}

	sort.Strings(prefix)
	sort.Strings(contains)

	results := make([]string, 0, len(exact)+len(prefix)+len(contains))
	results = append(results, exact...)
	results = append(results, prefix...)
	results = append(results, contains...)

	return results
}

// normalizeName converts query to PyPI normalized form (PEP 503):
// lowercase, underscores/dots replaced with hyphens, consecutive hyphens collapsed.
func normalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.NewReplacer("_", "-", ".", "-").Replace(name)
	return name
}
