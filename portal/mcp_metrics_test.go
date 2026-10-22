package portal

import (
	"testing"
	"time"
)

func TestBuildSelector(t *testing.T) {
	sel, err := buildSelector("INTERFACES_in_octets_rate", "rtr1",
		map[string]string{"if_name": "et-5/5/0"}, map[string]string{"LINKNAME": `.*CORE"x.*`})
	if err != nil {
		t.Fatal(err)
	}
	want := `INTERFACES_in_octets_rate{SHORTNAME="rtr1",if_name="et-5/5/0",LINKNAME=~".*CORE\"x.*"}`
	if sel != want {
		t.Errorf("got %s want %s", sel, want)
	}

	for _, c := range []struct {
		metric string
		labels map[string]string
	}{
		{"up or vector(1)", nil},
		{"INTERFACES_in_octets_rate", map[string]string{"if_name}": "x"}},
		{"INTERFACES_in_octets_rate", map[string]string{"__name__": "x"}},
	} {
		if _, err := buildSelector(c.metric, "", c.labels, nil); err == nil {
			t.Errorf("expected rejection for %q %v", c.metric, c.labels)
		}
	}
	if _, err := buildSelector("m", "", nil, map[string]string{"a": "("}); err == nil {
		t.Errorf("invalid regex must be rejected")
	}
}

func TestParseWindow(t *testing.T) {
	if d, err := parseWindow(""); err != nil || d != defaultWindow {
		t.Errorf("default: %v %v", d, err)
	}
	if d, err := parseWindow("15m"); err != nil || d != 15*time.Minute {
		t.Errorf("15m: %v %v", d, err)
	}
	for _, s := range []string{"16m", "1h", "5s", "abc"} {
		if _, err := parseWindow(s); err == nil {
			t.Errorf("%s should be rejected", s)
		}
	}
	if q := aggregateExpr("p95", "m{}", 5*time.Minute); q != "quantile_over_time(0.95, m{}[300s])" {
		t.Errorf("unexpected expr %s", q)
	}
}
