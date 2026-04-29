package semver

import (
	"regexp"
	"strconv"
	"strings"
)

// IsStable returns true if the version string looks like a stable release.
// Post-releases (e.g. 1.0.post1) count as stable per PEP 440; pre-releases and
// dev releases do not.
func IsStable(version string) bool {
	p := parsePEP440(version)
	if !p.valid {
		return true
	}
	return p.preSent == 1 && p.devSent == 1
}

// pep440Re matches a subset of PEP 440 versions: optional epoch, release segments,
// optional pre-release (a/b/c/rc/alpha/beta/pre/preview), post, dev, and local part (ignored).
var pep440Re = regexp.MustCompile(
	`^v?(?:(\d+)!)?` +
		`(\d+(?:\.\d+)*)` +
		`(?:[-_.]?(a|b|c|rc|alpha|beta|pre|preview)[-_.]?(\d*))?` +
		`(?:[-_.]?(post)[-_.]?(\d*))?` +
		`(?:[-_.]?(dev)[-_.]?(\d*))?` +
		`(?:\+[a-z0-9.]+)?$`,
)

type parsedVersion struct {
	valid    bool
	epoch    int
	release  []int
	preSent  int // -1 dev-only (NegInf), 0 has pre, 1 no pre (PosInf)
	preKind  int // 0=a, 1=b, 2=rc/c/pre/preview (only when preSent==0)
	preNum   int
	postSent int // -1 no post (NegInf), 0 has post
	postNum  int
	devSent  int // 0 has dev, 1 no dev (PosInf)
	devNum   int
}

func parsePEP440(v string) parsedVersion {
	s := strings.TrimSpace(strings.ToLower(v))
	m := pep440Re.FindStringSubmatch(s)
	if m == nil {
		return parsedVersion{}
	}
	p := parsedVersion{valid: true}
	if m[1] != "" {
		p.epoch, _ = strconv.Atoi(m[1])
	}
	for r := range strings.SplitSeq(m[2], ".") {
		n, _ := strconv.Atoi(r)
		p.release = append(p.release, n)
	}

	hasPre := m[3] != ""
	hasPost := m[5] != ""
	hasDev := m[7] != ""

	switch {
	case !hasPre && !hasPost && hasDev:
		// dev-only release sorts before any pre-release of the same base
		p.preSent = -1
	case !hasPre:
		p.preSent = 1
	default:
		p.preSent = 0
		p.preKind = preKindRank(m[3])
		p.preNum, _ = strconv.Atoi(m[4])
	}

	if hasPost {
		p.postSent = 0
		p.postNum, _ = strconv.Atoi(m[6])
	} else {
		p.postSent = -1
	}

	if hasDev {
		p.devSent = 0
		p.devNum, _ = strconv.Atoi(m[8])
	} else {
		p.devSent = 1
	}

	return p
}

func preKindRank(letter string) int {
	switch letter {
	case "a", "alpha":
		return 0
	case "b", "beta":
		return 1
	case "c", "rc", "pre", "preview":
		return 2
	}
	return 0
}

// Compare compares two PEP 440-style version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b string) int {
	pa := parsePEP440(a)
	pb := parsePEP440(b)
	if !pa.valid || !pb.valid {
		return compareNaive(a, b)
	}
	if c := cmpInt(pa.epoch, pb.epoch); c != 0 {
		return c
	}
	if c := cmpReleases(pa.release, pb.release); c != 0 {
		return c
	}
	if c := cmpInt(pa.preSent, pb.preSent); c != 0 {
		return c
	}
	if pa.preSent == 0 {
		if c := cmpInt(pa.preKind, pb.preKind); c != 0 {
			return c
		}
		if c := cmpInt(pa.preNum, pb.preNum); c != 0 {
			return c
		}
	}
	if c := cmpInt(pa.postSent, pb.postSent); c != 0 {
		return c
	}
	if pa.postSent == 0 {
		if c := cmpInt(pa.postNum, pb.postNum); c != 0 {
			return c
		}
	}
	if c := cmpInt(pa.devSent, pb.devSent); c != 0 {
		return c
	}
	if pa.devSent == 0 {
		if c := cmpInt(pa.devNum, pb.devNum); c != 0 {
			return c
		}
	}
	return 0
}

func compareNaive(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	maxLen := max(len(pa), len(pb))
	for i := range maxLen {
		var va, vb int
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if c := cmpInt(va, vb); c != 0 {
			return c
		}
	}
	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpReleases(a, b []int) int {
	maxLen := max(len(a), len(b))
	for i := range maxLen {
		var va, vb int
		if i < len(a) {
			va = a[i]
		}
		if i < len(b) {
			vb = b[i]
		}
		if c := cmpInt(va, vb); c != 0 {
			return c
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
