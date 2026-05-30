package semver

import (
	"strings"
	"testing"

	"github.com/yoyrandao/autotag/internal/semconv"
)

func mustParse(t *testing.T, s string) Version {
	t.Helper()
	v, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}

func commits(specs ...string) []semconv.Commit {
	out := make([]semconv.Commit, 0, len(specs))
	for _, s := range specs {
		c, ok := semconv.Parse(s)
		if !ok {
			continue
		}
		out = append(out, c)
	}
	return out
}

func TestClassify(t *testing.T) {
	cases := []struct {
		msg  string
		want Bump
	}{
		{"feat: x", BumpMinor},
		{"fix: x", BumpPatch},
		{"chore: x", BumpPatch},
		{"docs: x", BumpPatch},
		{"refactor: x", BumpPatch},
		{"feat!: x", BumpMajor},
		{"fix!: x", BumpMajor},
		{"feat: x\n\nBREAKING CHANGE: y", BumpMajor},
	}
	for _, tc := range cases {
		c, ok := semconv.Parse(tc.msg)
		if !ok {
			t.Fatalf("parse %q failed", tc.msg)
		}
		if got := Classify(c); got != tc.want {
			t.Errorf("Classify(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestAggregate(t *testing.T) {
	cs := commits("fix: a", "feat: b", "chore: c")
	if got := Aggregate(cs); got != BumpMinor {
		t.Fatalf("Aggregate = %v, want BumpMinor", got)
	}
	cs = commits("fix: a", "feat: b", "feat!: break")
	if got := Aggregate(cs); got != BumpMajor {
		t.Fatalf("Aggregate = %v, want BumpMajor", got)
	}
	cs = commits("not a conventional commit")
	if got := Aggregate(cs); got != BumpNone {
		t.Fatalf("Aggregate empty = %v, want BumpNone", got)
	}
}

func TestNext(t *testing.T) {
	type tc struct {
		name string
		last string // "" = nil
		agg  Bump
		pre  string
		want string
	}
	cases := []tc{
		{"no tag", "", BumpMajor, "", "0.1.0"},
		{"no tag with pre", "", BumpMajor, "rc", "0.1.0-rc.1"},
		{"fix only", "v1.2.3", BumpPatch, "", "1.2.4"},
		{"feat", "v1.2.3", BumpMinor, "", "1.3.0"},
		{"breaking post-1.0", "v1.2.3", BumpMajor, "", "2.0.0"},
		{"breaking pre-1.0 -> minor", "v0.3.4", BumpMajor, "", "0.4.0"},
		{"feat pre-1.0", "v0.3.4", BumpMinor, "", "0.4.0"},
		{"fix pre-1.0", "v0.3.4", BumpPatch, "", "0.3.5"},
		{"chore only counts as patch", "v1.2.3", BumpPatch, "", "1.2.4"},
		{"no commits no prerelease -> same", "v1.2.3", BumpNone, "", "1.2.3"},

		// prerelease increment same suffix, base unchanged because in cycle
		{"rc cycle fix same suffix", "v1.0.0-rc.1", BumpPatch, "rc", "1.0.0-rc.2"},
		{"rc cycle feat same suffix", "v1.0.0-rc.1", BumpMinor, "rc", "1.0.0-rc.2"},

		// prerelease suffix switch -> fresh counter, base unchanged
		{"rc -> beta non-major", "v1.0.0-rc.3", BumpPatch, "beta", "1.0.0-beta.1"},

		// prerelease promotion to stable
		{"finalize rc", "v1.0.0-rc.5", BumpNone, "", "1.0.0"},
		{"finalize rc with fix", "v1.0.0-rc.5", BumpPatch, "", "1.0.0"},

		// major during prerelease cycle pushes base
		{"breaking during rc post-1.0", "v1.0.0-rc.1", BumpMajor, "rc", "2.0.0-rc.1"},
		{"breaking during rc pre-1.0", "v0.5.0-rc.1", BumpMajor, "rc", "0.6.0-rc.1"},

		// stable + pre flag -> open new cycle on bumped base
		{"open rc on feat", "v1.2.3", BumpMinor, "rc", "1.3.0-rc.1"},
		{"open rc on fix", "v1.2.3", BumpPatch, "rc", "1.2.4-rc.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var last *Version
			if c.last != "" {
				v := mustParse(t, c.last)
				last = &v
			}
			got, err := Next(last, c.agg, Options{PreSuffix: c.pre})
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if got.String() != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestNextReleaseAs(t *testing.T) {
	t.Run("promote 0.x to 1.0.0", func(t *testing.T) {
		last := mustParse(t, "v0.5.3")
		target := mustParse(t, "1.0.0")
		got, err := Next(&last, BumpMinor, Options{ReleaseAs: &target})
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != "1.0.0" {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("as with pre", func(t *testing.T) {
		last := mustParse(t, "v0.5.3")
		target := mustParse(t, "1.0.0")
		got, err := Next(&last, BumpNone, Options{ReleaseAs: &target, PreSuffix: "rc"})
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != "1.0.0-rc.1" {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("as not greater rejected", func(t *testing.T) {
		last := mustParse(t, "v1.2.3")
		target := mustParse(t, "1.2.3")
		_, err := Next(&last, BumpNone, Options{ReleaseAs: &target})
		if err == nil || !strings.Contains(err.Error(), "must be greater") {
			t.Fatalf("expected error, got %v", err)
		}
	})
	t.Run("as lower rejected", func(t *testing.T) {
		last := mustParse(t, "v2.0.0")
		target := mustParse(t, "1.5.0")
		_, err := Next(&last, BumpNone, Options{ReleaseAs: &target})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("as without last", func(t *testing.T) {
		target := mustParse(t, "1.0.0")
		got, err := Next(nil, BumpNone, Options{ReleaseAs: &target})
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != "1.0.0" {
			t.Fatalf("got %s", got)
		}
	})
}
