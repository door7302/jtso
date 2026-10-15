package portal

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"jtso/association"
	"jtso/container"
	"jtso/sqlite"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestMCPClient(t *testing.T) *mcp.ClientSession {
	t.Helper()
	srv := httptest.NewServer(newMCPHandler())
	t.Cleanup(srv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: srv.URL}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestMCPToolsRegistered(t *testing.T) {
	session := newTestMCPClient(t)
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	want := map[string]bool{
		"list_routers": false, "add_router": false, "refresh_router": false, "remove_router": false,
		"list_profiles": false, "get_profile": false,
		"list_associations": false, "add_association": false, "remove_association": false,
		"get_stack_status": false, "get_container_stats": false, "get_container_logs": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %s not registered", name)
		}
	}
}

func TestMCPListRoutersAndProfiles(t *testing.T) {
	sqlite.RtrList = []*sqlite.RtrEntry{
		{Hostname: "r1.example.com", Shortname: "r1", Family: "mx", Model: "MX480", Version: "23.4R1", Profile: 1},
		{Hostname: "r2.example.com", Shortname: "r2", Family: "ptx", Model: "PTX10001", Version: "22.4R3", Profile: 0},
	}
	sqlite.AssoList = []*sqlite.AssoEntry{{Shortname: "r1", Assos: []string{"bgp", "interfaces"}, Kafka: "yes"}}
	association.ActiveProfiles = map[string]association.FileTgz{
		"bgp": {Definition: &association.DefProfile{Description: "BGP", TelCfg: association.Telegraf{MxCfg: []association.Config{{Version: "all", Config: "mx.json"}}}}},
	}
	t.Cleanup(func() {
		sqlite.RtrList, sqlite.AssoList = nil, nil
		association.ActiveProfiles = map[string]association.FileTgz{}
	})

	session := newTestMCPClient(t)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_routers"})
	if err != nil || res.IsError {
		t.Fatalf("list_routers failed: err=%v res=%+v", err, res)
	}
	var routers mcpRouterList
	raw, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(raw, &routers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if routers.Count != 2 || routers.Routers[0].Shortname != "r1" || !routers.Routers[0].Kafka || len(routers.Routers[0].Profiles) != 2 {
		t.Errorf("unexpected routers: %+v", routers)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get_profile", Arguments: map[string]any{"name": "bgp"}})
	if err != nil || res.IsError {
		t.Fatalf("get_profile failed: err=%v res=%+v", err, res)
	}
	var prof mcpProfileDetails
	raw, _ = json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(raw, &prof)
	if prof.Description != "BGP" || len(prof.Telegraf["mx"]) != 1 || len(prof.Routers) != 1 || prof.Routers[0] != "r1" {
		t.Errorf("unexpected profile: %+v", prof)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "remove_router", Arguments: map[string]any{"shortname": "r1"}})
	if err != nil {
		t.Fatalf("remove_router protocol error: %v", err)
	}
	if !res.IsError {
		t.Errorf("remove_router of an associated router should be a tool error")
	}
}

func TestMCPContainerStatsAndLogs(t *testing.T) {
	container.Init(1)
	container.Cstats.Stats = map[string]map[string]float64{
		"telegraf_mx": {"cpu": 12.5, "mem": 3.2},
		"grafana":     {"error": 1},
	}
	session := newTestMCPClient(t)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_container_stats", Arguments: map[string]any{"container": "telegraf_mx"}})
	if err != nil || res.IsError {
		t.Fatalf("get_container_stats failed: err=%v res=%+v", err, res)
	}
	var stats mcpContainerStatsList
	raw, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(raw, &stats)
	if len(stats.Stats) != 1 || stats.Stats[0].CPUPercent != 12.5 {
		t.Errorf("unexpected stats: %+v", stats)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get_container_logs", Arguments: map[string]any{"container": "nginx"}})
	if err != nil {
		t.Fatalf("get_container_logs protocol error: %v", err)
	}
	if !res.IsError {
		t.Errorf("logs of a container outside the stack must be refused")
	}
}
