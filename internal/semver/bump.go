package semver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yoyrandao/autotag/internal/conventional"
)

type Bump int

const (
	BumpNone Bump = iota
	BumpPatch
	BumpMinor
	BumpMajor
)

// Options controls Next's behavior.
//
// PreSuffix, when non-empty, makes Next emit a prerelease tag (e.g. "rc",
// "beta"). Counter is incremented when the previous tag shares the same base
// and suffix, otherwise reset to 1.
//
// ReleaseAs, when non-nil, overrides all bump computation; the returned
// version is *ReleaseAs (with PreSuffix applied). It must be strictly greater
// than last's base.
type Options struct {
	PreSuffix string
	ReleaseAs *Version
}

// Classify maps a single commit to its bump weight.
func Classify(c conventional.Commit) Bump {
	if c.Breaking {
		return BumpMajor
	}
	switch c.Type {
	case "":
		return BumpNone
	case "feat":
		return BumpMinor
	case "fix":
		return BumpPatch
	default:
		return BumpPatch
	}
}

// Aggregate returns the highest bump across commits.
func Aggregate(commits []conventional.Commit) Bump {
	highest := BumpNone
	for _, c := range commits {
		if b := Classify(c); b > highest {
			highest = b
		}
	}

	return highest
}

// Next computes the next Version.
//
//   - last == nil: returns 0.1.0 (initial release), with PreSuffix applied.
//   - opts.ReleaseAs != nil: returns the override, validated against last.
//   - Otherwise: applies bump rules (pre-1.0 breaking degrades to minor) then
//     handles prerelease per Options.PreSuffix.
//
// Returns error only when ReleaseAs is not strictly greater than last's base.
func Next(last *Version, agg Bump, opts Options) (Version, error) {
	if opts.ReleaseAs != nil {
		return nextAs(last, *opts.ReleaseAs, opts.PreSuffix)
	}

	if last == nil {
		base := Version{Major: 0, Minor: 1, Patch: 0}
		return applyPre(base, opts.PreSuffix, nil), nil
	}

	if agg == BumpNone && last.PreRelease == "" {
		return *last, nil
	}

	base := nextBase(*last, agg)
	return applyPre(base, opts.PreSuffix, last), nil
}

func nextAs(last *Version, as Version, pre string) (Version, error) {
	as.PreRelease = ""
	if last != nil {
		if Compare(as, last.stripped()) <= 0 {
			return Version{}, fmt.Errorf(
				"semver: as %s must be greater than last %s", as, last.stripped(),
			)
		}
	}

	if pre != "" {
		as.PreRelease = pre + ".1"
	}

	return as, nil
}

// nextBase computes the base (no prerelease) for the next version, given last
// (which may itself be a prerelease) and the aggregated bump.
//
// Within an open prerelease cycle (last.Prerelease != "") non-major bumps roll
// into the same base — the base was already advanced when the cycle opened.
// A major bump still advances the base.
func nextBase(last Version, agg Bump) Version {
	base := last.stripped()
	inCycle := last.PreRelease != ""

	if inCycle && agg != BumpMajor {
		return base
	}

	if base.Major == 0 {
		// Pre-1.0: breaking degrades to minor.
		switch agg {
		case BumpMajor, BumpMinor:
			return Version{Major: 0, Minor: base.Minor + 1, Patch: 0}
		case BumpPatch:
			return Version{Major: 0, Minor: base.Minor, Patch: base.Patch + 1}
		default:
			return base
		}
	}

	switch agg {
	case BumpMajor:
		return Version{Major: base.Major + 1, Minor: 0, Patch: 0}
	case BumpMinor:
		return Version{Major: base.Major, Minor: base.Minor + 1, Patch: 0}
	case BumpPatch:
		return Version{Major: base.Major, Minor: base.Minor, Patch: base.Patch + 1}
	default:
		return base
	}
}

// applyPre composes the final version, attaching/incrementing the prerelease
// counter per Options.PreSuffix.
func applyPre(base Version, suffix string, last *Version) Version {
	if suffix == "" {
		return base
	}

	if last != nil && last.PreRelease != "" {
		prevSuffix, prevN, ok := splitPre(last.PreRelease)
		if ok && prevSuffix == suffix && Compare(base, last.stripped()) == 0 {
			base.PreRelease = suffix + "." + strconv.FormatUint(prevN+1, 10)
			return base
		}
	}

	base.PreRelease = suffix + ".1"
	return base
}

// splitPre parses "<suffix>.<N>" form. Returns ok=false otherwise.
func splitPre(p string) (string, uint64, bool) {
	idx := strings.LastIndex(p, ".")
	if idx < 0 {
		return "", 0, false
	}
	n, err := strconv.ParseUint(p[idx+1:], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return p[:idx], n, true
}
