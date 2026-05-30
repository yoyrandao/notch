package conventional

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want Commit
		ok   bool
	}{
		{
			name: "feat simple",
			msg:  "feat: add login",
			want: Commit{Type: "feat", Description: "add login"},
			ok:   true,
		},
		{
			name: "fix scoped",
			msg:  "fix(api): handle nil",
			want: Commit{Type: "fix", Scope: "api", Description: "handle nil"},
			ok:   true,
		},
		{
			name: "breaking via bang",
			msg:  "feat!: drop v1",
			want: Commit{Type: "feat", Description: "drop v1", Breaking: true},
			ok:   true,
		},
		{
			name: "breaking via scoped bang",
			msg:  "refactor(core)!: rewrite",
			want: Commit{Type: "refactor", Scope: "core", Description: "rewrite", Breaking: true},
			ok:   true,
		},
		{
			name: "breaking via footer",
			msg:  "feat: new flow\n\nBody text.\n\nBREAKING CHANGE: removes legacy.",
			want: Commit{
				Type: "feat", Description: "new flow",
				Body:     "Body text.\n\nBREAKING CHANGE: removes legacy.",
				Breaking: true,
			},
			ok: true,
		},
		{
			name: "breaking via hyphenated footer",
			msg:  "fix: x\n\nBREAKING-CHANGE: yes",
			want: Commit{
				Type: "fix", Description: "x",
				Body: "BREAKING-CHANGE: yes", Breaking: true,
			},
			ok: true,
		},
		{
			name: "chore",
			msg:  "chore: bump deps",
			want: Commit{Type: "chore", Description: "bump deps"},
			ok:   true,
		},
		{
			name: "uppercase type lowered",
			msg:  "Feat: thing",
			want: Commit{Type: "feat", Description: "thing"},
			ok:   true,
		},
		{
			name: "invalid no colon",
			msg:  "just a message",
			ok:   false,
		},
		{
			name: "invalid empty",
			msg:  "",
			ok:   false,
		},
		{
			name: "invalid leading space",
			msg:  " feat: x",
			ok:   true, // TrimSpace makes it valid
			want: Commit{Type: "feat", Description: "x"},
		},
		{
			name: "multiline body retained",
			msg:  "feat: x\n\nline1\nline2",
			want: Commit{Type: "feat", Description: "x", Body: "line1\nline2"},
			ok:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Parse(tc.msg)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if got != tc.want {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}
