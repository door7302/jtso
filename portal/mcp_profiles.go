package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"jtso/association"
	"jtso/maker"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// profilesRoot is where the profile archives are extracted (overridden in tests)
var profilesRoot = association.ActiveProfilesPath

const maxSearchMatches = 300

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type mcpProfileConfigInput struct {
	Name    string `json:"name" jsonschema:"profile name as returned by list_profiles"`
	Family  string `json:"family" jsonschema:"platform family (mx, ptx, acx, ex, qfx, srx, crpd, cptx, vmx, vsrx, vjunos, vevo)"`
	Version string `json:"version,omitempty" jsonschema:"Junos version of a router (e.g. 23.4R1): only the template JTSO would apply to that version is returned; empty for every variant"`
}

type mcpGnmiSub struct {
	Measurement string `json:"measurement" jsonschema:"telegraf measurement name"`
	Path        string `json:"path" jsonschema:"gNMI sensor path"`
	Mode        string `json:"mode" jsonschema:"sample or on_change"`
	Interval    int    `json:"interval" jsonschema:"default sample interval in seconds (may be overridden by the user in JTSO)"`
}

type mcpNetconfSub struct {
	Measurement string   `json:"measurement"`
	RPC         string   `json:"rpc" jsonschema:"Junos RPC polled via NETCONF"`
	Interval    int      `json:"interval" jsonschema:"seconds"`
	Fields      []string `json:"fields" jsonschema:"collected xpath:type entries"`
}

type mcpPlugin struct {
	Type     string         `json:"type" jsonschema:"telegraf plugin: clone, pivot, rename, xreducer, converter, enrichment, rate, monitoring, filtering, enum, regex, strings, prometheus, kafka, file..."`
	Order    int            `json:"order,omitempty" jsonschema:"processor order in the template (lower runs first)"`
	Namepass []string       `json:"namepass,omitempty" jsonschema:"measurements the plugin applies to"`
	Settings map[string]any `json:"settings,omitempty" jsonschema:"remaining plugin parameters as written in the template"`
}

type mcpProfileConfig struct {
	Version              string          `json:"version" jsonschema:"Junos version constraint of this variant: 'all', '23.4', '>=22.2', '<<23.2' ..."`
	Config               string          `json:"config" jsonschema:"template file name inside the profile"`
	GnmiSubscriptions    []mcpGnmiSub    `json:"gnmi_subscriptions"`
	NetconfSubscriptions []mcpNetconfSub `json:"netconf_subscriptions"`
	Processors           []mcpPlugin     `json:"processors" jsonschema:"processor pipeline sorted by order"`
	Outputs              []mcpPlugin     `json:"outputs"`
	ExportedFields       []string        `json:"exported_fields" jsonschema:"fields kept for the Prometheus output (fieldpass)"`
}

type mcpProfileConfigResult struct {
	Name          string             `json:"name"`
	Family        string             `json:"family"`
	RouterVersion string             `json:"router_version,omitempty"`
	Configs       []mcpProfileConfig `json:"configs"`
}

type mcpCompareInput struct {
	Name     string `json:"name" jsonschema:"profile name"`
	FamilyA  string `json:"family_a" jsonschema:"first platform family"`
	VersionA string `json:"version_a,omitempty" jsonschema:"Junos version for side A; empty uses the 'all' variant"`
	FamilyB  string `json:"family_b" jsonschema:"second platform family (may equal family_a to compare two versions)"`
	VersionB string `json:"version_b,omitempty" jsonschema:"Junos version for side B; empty uses the 'all' variant"`
}

type mcpDiff struct {
	OnlyInA []string `json:"only_in_a"`
	OnlyInB []string `json:"only_in_b"`
}

type mcpCompareResult struct {
	Name           string  `json:"name"`
	A              string  `json:"a" jsonschema:"family/version constraint/template of side A"`
	B              string  `json:"b" jsonschema:"family/version constraint/template of side B"`
	GnmiPaths      mcpDiff `json:"gnmi_paths" jsonschema:"entries are measurement|path|mode|interval"`
	NetconfFields  mcpDiff `json:"netconf_fields" jsonschema:"entries are measurement|rpc|xpath:type"`
	Processors     mcpDiff `json:"processors" jsonschema:"entries are type|namepass|settings"`
	ExportedFields mcpDiff `json:"exported_fields"`
}

type mcpSearchInput struct {
	Query  string `json:"query" jsonschema:"case-insensitive substring searched in sensor paths, measurements, NETCONF RPCs/fields and exported fields"`
	Family string `json:"family,omitempty" jsonschema:"restrict to one platform family"`
}

type mcpSearchMatch struct {
	Profile string `json:"profile"`
	Family  string `json:"family"`
	Version string `json:"version"`
	Config  string `json:"config"`
	Kind    string `json:"kind" jsonschema:"gnmi_path, measurement, netconf_rpc, netconf_field or exported_field"`
	Value   string `json:"value"`
}

type mcpSearchResult struct {
	Count     int              `json:"count"`
	Truncated bool             `json:"truncated,omitempty"`
	Matches   []mcpSearchMatch `json:"matches"`
}

// ---------------------------------------------------------------------------
// Template loading
// ---------------------------------------------------------------------------

// profileFamilyConfigs returns the template variants a profile declares for a family
func profileFamilyConfigs(name, family string) ([]mcpTelegrafConfig, error) {
	p, ok := snapshotProfiles()[name]
	if !ok || p.Definition == nil {
		return nil, fmt.Errorf("unknown profile %s", name)
	}
	all := profileTelegraf(p.Definition)
	cfgs, ok := all[family]
	if !ok {
		fams := make([]string, 0, len(all))
		for f := range all {
			fams = append(fams, f)
		}
		sort.Strings(fams)
		return nil, fmt.Errorf("profile %s does not support family %s (supported: %s)", name, family, strings.Join(fams, ", "))
	}
	return cfgs, nil
}

// loadProfileConfig parses one telegraf template of a profile
func loadProfileConfig(profile string, c mcpTelegrafConfig) (mcpProfileConfig, error) {
	out := mcpProfileConfig{
		Version: c.Version, Config: c.Config,
		GnmiSubscriptions: []mcpGnmiSub{}, NetconfSubscriptions: []mcpNetconfSub{},
		Processors: []mcpPlugin{}, Outputs: []mcpPlugin{}, ExportedFields: []string{},
	}
	if c.Config == "" || filepath.Base(c.Config) != c.Config || c.Config == ".." {
		return out, fmt.Errorf("invalid template name %q in profile %s", c.Config, profile)
	}
	raw, err := os.ReadFile(filepath.Join(profilesRoot, profile, c.Config))
	if err != nil {
		return out, fmt.Errorf("unable to read template %s of profile %s: %v", c.Config, profile, err)
	}
	var typed maker.TelegrafConfig
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &typed); err != nil {
		return out, fmt.Errorf("invalid template %s of profile %s: %v", c.Config, profile, err)
	}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return out, fmt.Errorf("invalid template %s of profile %s: %v", c.Config, profile, err)
	}

	for _, in := range typed.GnmiList {
		for _, s := range in.Subs {
			out.GnmiSubscriptions = append(out.GnmiSubscriptions, mcpGnmiSub{Measurement: s.Name, Path: s.Path, Mode: s.Mode, Interval: s.Interval})
		}
	}
	for _, in := range typed.NetconfList {
		for _, s := range in.Subs {
			ns := mcpNetconfSub{Measurement: s.Name, RPC: s.RPC, Interval: s.Interval, Fields: make([]string, 0, len(s.Fields))}
			for _, f := range s.Fields {
				ns.Fields = append(ns.Fields, f.FieldPath+":"+f.FieldType)
			}
			out.NetconfSubscriptions = append(out.NetconfSubscriptions, ns)
		}
	}
	for _, p := range typed.PrometheusList {
		out.ExportedFields = append(out.ExportedFields, p.Fieldpass...)
	}

	// Generic walk so plugin types added to maker later are exposed without code change here
	keys := make([]string, 0, len(generic))
	for k := range generic {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "gnmi_inputs" || k == "netconf_inputs" {
			continue
		}
		var items []map[string]any
		if err := json.Unmarshal(generic[k], &items); err != nil {
			continue
		}
		isOutput := strings.HasSuffix(k, "_outputs")
		typ := strings.TrimSuffix(strings.TrimSuffix(k, "_outputs"), "_list")
		for _, item := range items {
			pl := mcpPlugin{Type: typ}
			if v, ok := item["order"].(float64); ok {
				pl.Order = int(v)
			}
			if v, ok := item["namepass"].([]any); ok {
				for _, n := range v {
					if s, ok := n.(string); ok {
						pl.Namepass = append(pl.Namepass, s)
					}
				}
			}
			delete(item, "order")
			delete(item, "namepass")
			if len(item) > 0 {
				pl.Settings = item
			}
			if isOutput {
				out.Outputs = append(out.Outputs, pl)
			} else {
				out.Processors = append(out.Processors, pl)
			}
		}
	}
	sort.SliceStable(out.Processors, func(i, j int) bool { return out.Processors[i].Order < out.Processors[j].Order })
	return out, nil
}

// resolveProfileConfig loads the single template applied to family/version ("all" variant when version is empty)
func resolveProfileConfig(name, family, version string) (mcpProfileConfig, error) {
	cfgs, err := profileFamilyConfigs(name, family)
	if err != nil {
		return mcpProfileConfig{}, err
	}
	list := make([]association.Config, 0, len(cfgs))
	for _, c := range cfgs {
		list = append(list, association.Config{Version: c.Version, Config: c.Config})
	}
	conf, constraint := association.SelectConfig(list, version)
	if conf == "" && version == "" && len(cfgs) == 1 {
		conf, constraint = cfgs[0].Config, cfgs[0].Version
	}
	if conf == "" {
		return mcpProfileConfig{}, fmt.Errorf("no template of profile %s applies to %s version %q: give a version matching one of the constraints", name, family, version)
	}
	return loadProfileConfig(name, mcpTelegrafConfig{Version: constraint, Config: conf})
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func mcpGetProfileConfig(_ context.Context, _ *mcp.CallToolRequest, in mcpProfileConfigInput) (*mcp.CallToolResult, mcpProfileConfigResult, error) {
	name := strings.TrimSpace(in.Name)
	family := strings.ToLower(strings.TrimSpace(in.Family))
	version := strings.TrimSpace(in.Version)
	out := mcpProfileConfigResult{Name: name, Family: family, RouterVersion: version, Configs: []mcpProfileConfig{}}

	if version != "" {
		c, err := resolveProfileConfig(name, family, version)
		if err != nil {
			return nil, out, err
		}
		out.Configs = append(out.Configs, c)
		return nil, out, nil
	}

	cfgs, err := profileFamilyConfigs(name, family)
	if err != nil {
		return nil, out, err
	}
	for _, c := range cfgs {
		pc, err := loadProfileConfig(name, c)
		if err != nil {
			return nil, out, err
		}
		out.Configs = append(out.Configs, pc)
	}
	return nil, out, nil
}

func diffSets(a, b []string) mcpDiff {
	inA, inB := make(map[string]bool, len(a)), make(map[string]bool, len(b))
	for _, s := range a {
		inA[s] = true
	}
	for _, s := range b {
		inB[s] = true
	}
	d := mcpDiff{OnlyInA: []string{}, OnlyInB: []string{}}
	for s := range inA {
		if !inB[s] {
			d.OnlyInA = append(d.OnlyInA, s)
		}
	}
	for s := range inB {
		if !inA[s] {
			d.OnlyInB = append(d.OnlyInB, s)
		}
	}
	sort.Strings(d.OnlyInA)
	sort.Strings(d.OnlyInB)
	return d
}

// configKeys flattens a template into comparable strings per category
func configKeys(c mcpProfileConfig) (gnmi, netconf, procs []string) {
	for _, s := range c.GnmiSubscriptions {
		gnmi = append(gnmi, fmt.Sprintf("%s|%s|%s|%ds", s.Measurement, s.Path, s.Mode, s.Interval))
	}
	for _, s := range c.NetconfSubscriptions {
		for _, f := range s.Fields {
			netconf = append(netconf, fmt.Sprintf("%s|%s|%s", s.Measurement, s.RPC, f))
		}
	}
	// order is ignored: it is renumbered when templates are merged
	for _, p := range c.Processors {
		settings, _ := json.Marshal(p.Settings)
		procs = append(procs, fmt.Sprintf("%s|%s|%s", p.Type, strings.Join(p.Namepass, ","), settings))
	}
	return
}

func mcpCompareProfileConfigs(_ context.Context, _ *mcp.CallToolRequest, in mcpCompareInput) (*mcp.CallToolResult, mcpCompareResult, error) {
	name := strings.TrimSpace(in.Name)
	famA, famB := strings.ToLower(strings.TrimSpace(in.FamilyA)), strings.ToLower(strings.TrimSpace(in.FamilyB))
	a, err := resolveProfileConfig(name, famA, strings.TrimSpace(in.VersionA))
	if err != nil {
		return nil, mcpCompareResult{}, err
	}
	b, err := resolveProfileConfig(name, famB, strings.TrimSpace(in.VersionB))
	if err != nil {
		return nil, mcpCompareResult{}, err
	}
	gA, nA, pA := configKeys(a)
	gB, nB, pB := configKeys(b)
	return nil, mcpCompareResult{
		Name:           name,
		A:              fmt.Sprintf("%s / %s / %s", famA, a.Version, a.Config),
		B:              fmt.Sprintf("%s / %s / %s", famB, b.Version, b.Config),
		GnmiPaths:      diffSets(gA, gB),
		NetconfFields:  diffSets(nA, nB),
		Processors:     diffSets(pA, pB),
		ExportedFields: diffSets(a.ExportedFields, b.ExportedFields),
	}, nil
}

func mcpSearchProfiles(_ context.Context, _ *mcp.CallToolRequest, in mcpSearchInput) (*mcp.CallToolResult, mcpSearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(in.Query))
	if q == "" {
		return nil, mcpSearchResult{}, fmt.Errorf("query is required")
	}
	famFilter := strings.ToLower(strings.TrimSpace(in.Family))
	out := mcpSearchResult{Matches: []mcpSearchMatch{}}

	profiles := snapshotProfiles()
	names := make([]string, 0, len(profiles))
	for n := range profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		all := profileTelegraf(profiles[name].Definition)
		fams := make([]string, 0, len(all))
		for f := range all {
			if famFilter == "" || f == famFilter {
				fams = append(fams, f)
			}
		}
		sort.Strings(fams)
		for _, fam := range fams {
			for _, c := range all[fam] {
				pc, err := loadProfileConfig(name, c)
				if err != nil {
					continue
				}
				add := func(kind, value string) {
					if !strings.Contains(strings.ToLower(value), q) {
						return
					}
					if len(out.Matches) >= maxSearchMatches {
						out.Truncated = true
						return
					}
					out.Matches = append(out.Matches, mcpSearchMatch{Profile: name, Family: fam, Version: c.Version, Config: c.Config, Kind: kind, Value: value})
				}
				for _, s := range pc.GnmiSubscriptions {
					add("gnmi_path", s.Path)
					add("measurement", s.Measurement)
				}
				for _, s := range pc.NetconfSubscriptions {
					add("netconf_rpc", s.RPC)
					for _, f := range s.Fields {
						add("netconf_field", f)
					}
				}
				for _, f := range pc.ExportedFields {
					add("exported_field", f)
				}
			}
		}
	}
	out.Count = len(out.Matches)
	return nil, out, nil
}

func registerProfileContentTools(server *mcp.Server, readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_profile_config",
		Description: "Content of a profile's telegraf template(s) for one platform family: gNMI subscriptions (path, mode, interval), " +
			"NETCONF RPCs and fields, processor pipeline (rename, enum, rate, monitoring, ...), outputs and the fields exported to Prometheus. " +
			"Give a Junos version to get only the template JTSO would apply to a router running it.",
		Annotations: readOnly,
	}, mcpGetProfileConfig)

	mcp.AddTool(server, &mcp.Tool{
		Name: "compare_profile_configs",
		Description: "Differences between two templates of the same profile: two families (e.g. mx vs ptx) or two versions of one family. " +
			"Returns what is only on each side for gNMI subscriptions, NETCONF fields, processors and exported fields.",
		Annotations: readOnly,
	}, mcpCompareProfileConfigs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_profiles",
		Description: "Find which profiles/families/templates collect a sensor path, measurement, NETCONF RPC/field or exported field matching a substring (e.g. 'optics', 'bgp/neighbors', 'cpu').",
		Annotations: readOnly,
	}, mcpSearchProfiles)
}
