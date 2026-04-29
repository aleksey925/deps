package semver

import (
	"slices"
	"sort"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.0", "1.0.0", 0},
		{"1.0", "2.0", -1},
		{"2.0", "1.0", 1},
		{"1.2.3", "1.2.4", -1},
		{"1.0a1", "1.0", -1},
		{"1.0", "1.0a1", 1},
		{"1.0a1", "1.0a2", -1},
		{"1.0a1", "1.0b1", -1},
		{"1.0b1", "1.0rc1", -1},
		{"1.0rc1", "1.0", -1},
		{"1.0.dev1", "1.0", -1},
		{"1.0.dev1", "1.0a1", -1},
		{"1.0a1.dev1", "1.0a1", -1},
		{"1.0a1.dev1", "1.0.dev1", 1},
		{"1.0", "1.0.post1", -1},
		{"1.0.post1", "1.0", 1},
		{"1.0.post1", "1.0.post2", -1},
		{"1!1.0", "2.0", 1},
		{"1.0a", "1.0a0", 0},
		{"v1.0", "1.0", 0},
		{"1.0.0rc1", "1.0.0", -1},
		{"1.0alpha1", "1.0a1", 0},
		{"1.0beta1", "1.0b1", 0},
		{"1.0c1", "1.0rc1", 0},
		{"1.0+local", "1.0", 0},
	}
	for _, tc := range cases {
		got := Compare(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCompare_SortAscending(t *testing.T) {
	versions := []string{
		"1.0", "1.0a1", "1.0.dev1", "1.0b1", "1.0rc1",
		"1.0.post1", "0.9", "2.0", "1.0a1.dev1",
	}

	sort.Slice(versions, func(i, j int) bool {
		return Compare(versions[i], versions[j]) < 0
	})

	want := []string{
		"0.9", "1.0.dev1", "1.0a1.dev1", "1.0a1", "1.0b1",
		"1.0rc1", "1.0", "1.0.post1", "2.0",
	}
	if !slices.Equal(versions, want) {
		t.Errorf("sorted = %v, want %v", versions, want)
	}
}

func TestCompare_InvalidFallsBackToNaive(t *testing.T) {
	if got := Compare("not-a-version", "also-bad"); got != 0 {
		t.Errorf("Compare on garbage = %d, want 0", got)
	}
}

func TestIsStable(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"1.0", true},
		{"1.0.0", true},
		{"1.0a1", false},
		{"1.0b1", false},
		{"1.0rc1", false},
		{"1.0.dev1", false},
		{"1.0.post1", true},
		{"1.0.post1.dev1", false},
		{"1.0alpha1", false},
		{"1.0beta1", false},
		{"1!2.0", true},
		{"v1.0", true},
	}
	for _, tc := range cases {
		if got := IsStable(tc.v); got != tc.want {
			t.Errorf("IsStable(%q) = %v, want %v", tc.v, got, tc.want)
		}
	}
}
