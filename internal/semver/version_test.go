package semver

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in      string
		want    Version
		wantErr bool
	}{
		{"1.2.3", Version{1, 2, 3, ""}, false},
		{"v1.2.3", Version{1, 2, 3, ""}, false},
		{"v0.0.0", Version{0, 0, 0, ""}, false},
		{"v1.2.3-rc.1", Version{1, 2, 3, "rc.1"}, false},
		{"v10.20.30-beta.4", Version{10, 20, 30, "beta.4"}, false},
		{"", Version{}, true},
		{"v", Version{}, true},
		{"1.2", Version{}, true},
		{"1.2.3.4", Version{}, true},
		{"a.b.c", Version{}, true},
		{"v1.2.3-", Version{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3, ""}, "1.2.3"},
		{Version{0, 1, 0, ""}, "0.1.0"},
		{Version{1, 0, 0, "rc.1"}, "1.0.0-rc.1"},
	}
	for _, tc := range cases {
		if got := tc.v.String(); got != tc.want {
			t.Errorf("String(%+v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"1.2.0", "1.1.9", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc.1", 1},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1},
		{"1.0.0-rc.2", "1.0.0-rc.10", -1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha", 1},
		{"1.0.0-rc.1", "1.0.0-rc.1", 0},
		{"1.0.0-1", "1.0.0-alpha", -1},
	}
	for _, tc := range cases {
		a, _ := Parse(tc.a)
		b, _ := Parse(tc.b)
		if got := Compare(a, b); got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
