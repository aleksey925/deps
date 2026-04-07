package semver

import (
	"strconv"
	"strings"
)

// IsStable returns true if the version string looks like a stable release
// (no pre-release suffixes like alpha, beta, rc, dev).
func IsStable(version string) bool {
	lower := strings.ToLower(version)
	for _, suffix := range []string{"a", "b", "rc", "dev", "alpha", "beta", "pre", "post"} {
		if strings.Contains(lower, suffix) {
			return false
		}
	}
	return true
}

// Compare compares two PEP 440-style version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b string) int {
	partsA := parseVersion(a)
	partsB := parseVersion(b)

	maxLen := max(len(partsA), len(partsB))

	for i := range maxLen {
		var va, vb int
		if i < len(partsA) {
			va = partsA[i]
		}
		if i < len(partsB) {
			vb = partsB[i]
		}

		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}

	return 0
}

func parseVersion(v string) []int {
	parts := strings.Split(strings.TrimSpace(v), ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		result = append(result, n)
	}
	return result
}
