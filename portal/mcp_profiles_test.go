package portal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"jtso/association"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testMxTemplate = `{
  "gnmi_inputs": [{"subscriptions": [
    {"name": "COMPONENTS", "path": "/components/component", "mode": "sample", "interval": 60},
    {"name": "JUNOS_CMERROR", "path": "/junos/system/cmerror/counters", "mode": "sample", "interval": 30}
  ]}],
  "netconf_inputs": [{"subscriptions": [{"name": "JUNOS_ALARMS", "rpc": "<get-alarm-information></get-alarm-information>",
    "fields": [{"field_path": "/alarm-information/alarm-detail/alarm-class", "field_type": "string"}], "interval": 30}]}],
  "rename_list": [{"order": 300, "namepass": ["COMPONENTS"], "entries": [{"type": 1, "from": "a", "to": "b"}]}],
  "clone_list": [{"order": 100, "namepass": ["COMPONENTS"], "override": "HW_NPU"}],
  "prometheus_outputs": [{"fieldpass": ["cpu", "temperature"]}]
}`

const testPtxTemplate = `{
  "gnmi_inputs": [{"subscriptions": [
    {"name": "COMPONENTS", "path": "/components/component", "mode": "sample", "interval": 60}
  ]}],
  "prometheus_outputs": [{"fieldpass": ["cpu"]}]
}`

func setupTestProfiles(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "health"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"mx_health.json": testMxTemplate, "ptx_health.json": testPtxTemplate} {
		if err := os.WriteFile(filepath.Join(dir, "health", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	oldRoot := profilesRoot
	profilesRoot = dir
	association.ActiveProfiles = map[string]association.FileTgz{
		"health": {Definition: &association.DefProfile{Description: "Health", TelCfg: association.Telegraf{
			MxCfg:  []association.Config{{Version: "all", Config: "mx_health.json"}},
			PtxCfg: []association.Config{{Version: ">=22.2", Config: "ptx_health.json"}},
		}}},
	}
	t.Cleanup(func() {
		profilesRoot = oldRoot
		association.ActiveProfiles = map[string]association.FileTgz{}
	})
}

func callStructured(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s protocol error: %v", tool, err)
	}
	if !res.IsError && out != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s decode: %v", tool, err)
		}
	}
	return res
}

func TestMCPGetProfileConfig(t *testing.T) {
	setupTestProfiles(t)
	session := newTestMCPClient(t)

	var cfg mcpProfileConfigResult
	res := callStructured(t, session, "get_profile_config", map[string]any{"name": "health", "family": "mx"}, &cfg)
	if res.IsError {
		t.Fatalf("get_profile_config failed: %+v", res.Content)
	}
	if len(cfg.Configs) != 1 {
		t.Fatalf("expected 1 config, got %+v", cfg)
	}
	c := cfg.Configs[0]
	if len(c.GnmiSubscriptions) != 2 || len(c.NetconfSubscriptions) != 1 || len(c.ExportedFields) != 2 {
		t.Errorf("unexpected content: %+v", c)
	}
	if len(c.Processors) != 2 || c.Processors[0].Type != "clone" || c.Processors[1].Type != "rename" {
		t.Errorf("processors not ordered: %+v", c.Processors)
	}

	if res := callStructured(t, session, "get_profile_config", map[string]any{"name": "health", "family": "ptx", "version": "21.4R1"}, nil); !res.IsError {
		t.Errorf("ptx 21.4 has no matching template, expected a tool error")
	}
	if res := callStructured(t, session, "get_profile_config", map[string]any{"name": "health", "family": "acx"}, nil); !res.IsError {
		t.Errorf("unsupported family must be a tool error")
	}
}

func TestMCPCompareAndSearchProfiles(t *testing.T) {
	setupTestProfiles(t)
	session := newTestMCPClient(t)

	var diff mcpCompareResult
	res := callStructured(t, session, "compare_profile_configs",
		map[string]any{"name": "health", "family_a": "mx", "family_b": "ptx", "version_b": "23.4R1"}, &diff)
	if res.IsError {
		t.Fatalf("compare failed: %+v", res.Content)
	}
	if len(diff.GnmiPaths.OnlyInA) != 1 || len(diff.GnmiPaths.OnlyInB) != 0 || len(diff.NetconfFields.OnlyInA) != 1 ||
		len(diff.ExportedFields.OnlyInA) != 1 || diff.ExportedFields.OnlyInA[0] != "temperature" {
		t.Errorf("unexpected diff: %+v", diff)
	}

	var found mcpSearchResult
	res = callStructured(t, session, "search_profiles", map[string]any{"query": "cmerror"}, &found)
	if res.IsError {
		t.Fatalf("search failed: %+v", res.Content)
	}
	if found.Count != 2 || found.Matches[0].Family != "mx" {
		t.Errorf("unexpected search result: %+v", found)
	}
}

func TestSelectConfig(t *testing.T) {
	list := []association.Config{{Version: ">=22.2", Config: "new.json"}, {Version: "all", Config: "default.json"}}
	if c, v := association.SelectConfig(list, "23.4R1"); c != "new.json" || v != ">=22.2" {
		t.Errorf("got %s %s", c, v)
	}
	if c, v := association.SelectConfig(list, "21.4R1"); c != "default.json" || v != "all" {
		t.Errorf("got %s %s", c, v)
	}
}
