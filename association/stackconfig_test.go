package association

import (
	"reflect"
	"testing"
)

func TestGetLeaf(t *testing.T) {
	cases := map[string]string{
		"a/b/c-d":  "c_d",
		"single":   "single",
		"x-y":      "x_y",
		"/root/in": "in",
	}
	for in, want := range cases {
		if got := getLeaf(in); got != want {
			t.Errorf("getLeaf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetLastTwoNodes(t *testing.T) {
	cases := map[string]string{
		"a/b/c-d": "b_c_d",
		"single":  "single",
		"x/y":     "x_y",
	}
	for in, want := range cases {
		if got := getLastTwoNodes(in); got != want {
			t.Errorf("getLastTwoNodes(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateFileName(t *testing.T) {
	for _, ok := range []string{"profile1", "my-profile_2"} {
		if err := validateFileName(ok); err != nil {
			t.Errorf("validateFileName(%q) unexpected error: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "../x", "a/b", `a\b`, "a..b"} {
		if err := validateFileName(bad); err == nil {
			t.Errorf("validateFileName(%q) expected error", bad)
		}
	}
}

func TestSplitBrokers(t *testing.T) {
	got := splitBrokers(" k1:9092, k2:9092 ,,k3:9092")
	want := []string{"k1:9092", "k2:9092", "k3:9092"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitBrokers = %v, want %v", got, want)
	}
}

func TestCheckVersion(t *testing.T) {
	cases := []struct {
		search, router string
		want           bool
	}{
		{"23.4", "23.4R1.10", true},
		{"22.4", "23.4R1.10", false},
		{">=23.2", "23.4R1.10", true},
		{"<<23.2", "23.4R1.10", false},
		{"ab", "23.4R1.10", false},
	}
	for _, c := range cases {
		if got := CheckVersion(c.search, c.router); got != c.want {
			t.Errorf("CheckVersion(%q, %q) = %v, want %v", c.search, c.router, got, c.want)
		}
	}
}
