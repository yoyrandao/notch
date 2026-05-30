// Package semver computes the next semantic version from commit bumps.
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch uint64
	PreRelease          string
}

func Parse(s string) (Version, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return Version{}, fmt.Errorf("semver: empty version")
	}

	pre := ""
	if i := strings.Index(s, "-"); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
		if pre == "" {
			return Version{}, fmt.Errorf("semver: empty prerelease")
		}
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("semver: invalid version %q (need MAJOR.MINOR.PATCH)", s)
	}

	nums := [3]uint64{}
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("semver: invalid number %q: %w", p, err)
		}
		nums[i] = n
	}
	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2], PreRelease: pre}, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	return s
}

func (v Version) stripped() Version {
	v.PreRelease = ""
	return v
}

// Compare returns -1, 0, or 1 per SemVer 2.0 precedence.
func Compare(a, b Version) int {
	if c := cmpUint(a.Major, b.Major); c != 0 {
		return c
	}
	if c := cmpUint(a.Minor, b.Minor); c != 0 {
		return c
	}
	if c := cmpUint(a.Patch, b.Patch); c != 0 {
		return c
	}
	return comparePre(a.PreRelease, b.PreRelease)
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// SemVer 2.0 §11: a version without prerelease has higher precedence
// than one with prerelease. Otherwise compare identifiers left to right.
func comparePre(a, b string) int {
	switch {
	case a == "" && b == "":
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}

	ai := strings.Split(a, ".")
	bi := strings.Split(b, ".")
	n := min(len(ai), len(bi))
	for i := range n {
		if c := cmpIdent(ai[i], bi[i]); c != 0 {
			return c
		}
	}
	return cmpUint(uint64(len(ai)), uint64(len(bi)))
}

func cmpIdent(a, b string) int {
	an, aErr := strconv.ParseUint(a, 10, 64)
	bn, bErr := strconv.ParseUint(b, 10, 64)
	switch {
	case aErr == nil && bErr == nil:
		return cmpUint(an, bn)
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	default:
		return strings.Compare(a, b)
	}
}
