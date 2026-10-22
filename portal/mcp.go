package portal

import (
	"context"
	"fmt"
	"jtso/association"
	"jtso/config"
	"jtso/container"
	"jtso/logger"
	"jtso/netconf"
	"jtso/ondemand"
	"jtso/prometheus"
	"jtso/sqlite"
	"jtso/worker"
	"net/http"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPPath is the URL path where the Streamable HTTP MCP endpoint is mounted on the portal
const MCPPath = "/mcp"

// ---------------------------------------------------------------------------
// Tool input / output types (JSON schema is inferred from these structs)
// ---------------------------------------------------------------------------

type mcpRouter struct {
	Shortname string   `json:"shortname" jsonschema:"unique short identifier of the router"`
	Hostname  string   `json:"hostname" jsonschema:"FQDN or IP address of the router"`
	Family    string   `json:"family" jsonschema:"platform family (mx, ptx, acx, ex, qfx, srx, crpd, cptx, vmx, vsrx, vjunos, vevo)"`
	Model     string   `json:"model"`
	Version   string   `json:"version" jsonschema:"Junos version"`
	Profiles  []string `json:"profiles" jsonschema:"profiles currently associated with the router (empty when not monitored)"`
	Kafka     bool     `json:"kafka" jsonschema:"true when the router's metrics are also exported to Kafka"`
}

type mcpRouterList struct {
	Count   int         `json:"count"`
	Routers []mcpRouter `json:"routers"`
}

type mcpAddRouterInput struct {
	Hostname  string `json:"hostname" jsonschema:"FQDN or IP address of the router, reachable via NETCONF"`
	Shortname string `json:"shortname" jsonschema:"unique short identifier to give to the router"`
}

type mcpShortnameInput struct {
	Shortname string `json:"shortname" jsonschema:"short identifier of the router"`
}

type mcpRouterResult struct {
	Shortname string `json:"shortname"`
	Hostname  string `json:"hostname"`
	Family    string `json:"family"`
	Model     string `json:"model"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

type mcpMessage struct {
	Message string `json:"message"`
}

type mcpTelegrafConfig struct {
	Version string `json:"version" jsonschema:"Junos version constraint: 'all', '23.4', '>=22.1', '<<23.2' ..."`
	Config  string `json:"config" jsonschema:"telegraf configuration file name inside the profile"`
}

type mcpProfileSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Families    []string `json:"families" jsonschema:"platform families supported by the profile"`
	Dashboards  []string `json:"dashboards" jsonschema:"grafana dashboards shipped with the profile"`
	UsedBy      int      `json:"used_by" jsonschema:"number of routers currently associated with the profile"`
}

type mcpProfileList struct {
	Count    int                 `json:"count"`
	Profiles []mcpProfileSummary `json:"profiles"`
}

type mcpProfileNameInput struct {
	Name string `json:"name" jsonschema:"profile name as returned by list_profiles"`
}

type mcpProfileDetails struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Version     int                            `json:"version" jsonschema:"profile definition version"`
	Telegraf    map[string][]mcpTelegrafConfig `json:"telegraf" jsonschema:"telegraf configurations per platform family"`
	Dashboards  []string                       `json:"dashboards"`
	Routers     []string                       `json:"routers" jsonschema:"shortnames of the routers currently associated with the profile"`
}

type mcpAssociation struct {
	Shortname string   `json:"shortname"`
	Family    string   `json:"family"`
	Profiles  []string `json:"profiles"`
	Kafka     bool     `json:"kafka"`
}

type mcpAssociationList struct {
	Count        int              `json:"count"`
	Associations []mcpAssociation `json:"associations"`
}

type mcpAddAssociationInput struct {
	Shortname string   `json:"shortname" jsonschema:"short identifier of the router"`
	Profiles  []string `json:"profiles" jsonschema:"profile names to attach to the router (see list_profiles)"`
	Kafka     bool     `json:"kafka,omitempty" jsonschema:"also export this router's metrics to Kafka (requires Kafka to be enabled in settings)"`
}

type mcpContainerInput struct {
	Container string `json:"container,omitempty" jsonschema:"container name (jtso, grafana, prometheus, telegraf_mx, telegraf_ptx, ... telegraf_ondemand); empty for all"`
}

type mcpContainerLogsInput struct {
	Container string `json:"container" jsonschema:"container name (jtso, grafana, prometheus, telegraf_mx, telegraf_ptx, ... telegraf_ondemand)"`
	Tail      int    `json:"tail,omitempty" jsonschema:"number of most recent lines to return (default 100, max 1000)"`
}

type mcpContainerStatus struct {
	Name       string   `json:"name"`
	Role       string   `json:"role" jsonschema:"jtso, grafana, prometheus or telegraf"`
	State      string   `json:"state" jsonschema:"docker state: running, exited, ... or 'absent' when the container does not exist"`
	Status     string   `json:"status,omitempty" jsonschema:"docker human status, e.g. 'Up 3 hours'"`
	Image      string   `json:"image,omitempty"`
	Family     string   `json:"family,omitempty" jsonschema:"telegraf instances only: platform family or 'ondemand'"`
	Routers    *int     `json:"routers,omitempty" jsonschema:"telegraf instances only: number of routers attached to the instance"`
	RouterList []string `json:"router_list,omitempty" jsonschema:"telegraf instances only: shortnames of the attached routers"`
	Debug      *bool    `json:"debug,omitempty" jsonschema:"telegraf instances only: telegraf debug mode enabled"`
	CPUPercent *float64 `json:"cpu_percent,omitempty" jsonschema:"last sampled CPU usage (refreshed every minute)"`
	MemPercent *float64 `json:"mem_percent,omitempty" jsonschema:"last sampled memory usage (refreshed every minute)"`
}

type mcpStackStatus struct {
	TotalRouters   int                  `json:"total_routers"`
	Monitored      int                  `json:"monitored_routers" jsonschema:"routers with at least one profile associated"`
	OndemandActive bool                 `json:"ondemand_active"`
	OndemandName   string               `json:"ondemand_profile,omitempty"`
	Containers     []mcpContainerStatus `json:"containers"`
}

type mcpContainerStats struct {
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	Error      bool    `json:"error,omitempty" jsonschema:"true when the last sampling of this container failed"`
}

type mcpContainerStatsList struct {
	Note  string              `json:"note"`
	Stats []mcpContainerStats `json:"stats"`
}

type mcpContainerLogs struct {
	Container string   `json:"container"`
	Lines     int      `json:"lines"`
	Logs      []string `json:"logs"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func findRouter(shortname string) *sqlite.RtrEntry {
	for _, r := range sqlite.Routers() {
		if r.Shortname == shortname {
			return r
		}
	}
	return nil
}

func findAssociation(shortname string) *sqlite.AssoEntry {
	for _, a := range sqlite.Associations() {
		if a.Shortname == shortname {
			return a
		}
	}
	return nil
}

// associationProfiles returns the profile list of a router, filtering empty legacy entries
func associationProfiles(a *sqlite.AssoEntry) []string {
	out := make([]string, 0)
	if a == nil {
		return out
	}
	for _, p := range a.Assos {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// profileTelegraf flattens the per-family Telegraf struct into a family → configs map
func profileTelegraf(def *association.DefProfile) map[string][]mcpTelegrafConfig {
	out := make(map[string][]mcpTelegrafConfig)
	if def == nil {
		return out
	}
	families := []struct {
		name string
		cfgs []association.Config
	}{
		{"mx", def.TelCfg.MxCfg}, {"ptx", def.TelCfg.PtxCfg}, {"acx", def.TelCfg.AcxCfg},
		{"ex", def.TelCfg.ExCfg}, {"qfx", def.TelCfg.QfxCfg}, {"srx", def.TelCfg.SrxCfg},
		{"crpd", def.TelCfg.CrpdCfg}, {"cptx", def.TelCfg.CptxCfg}, {"vmx", def.TelCfg.VmxCfg},
		{"vsrx", def.TelCfg.VsrxCfg}, {"vjunos", def.TelCfg.VjunosCfg}, {"vevo", def.TelCfg.VevoCfg},
	}
	for _, f := range families {
		if len(f.cfgs) == 0 {
			continue
		}
		list := make([]mcpTelegrafConfig, 0, len(f.cfgs))
		for _, c := range f.cfgs {
			list = append(list, mcpTelegrafConfig{Version: c.Version, Config: c.Config})
		}
		out[f.name] = list
	}
	return out
}

// profileUsers returns the shortnames of the routers associated with a profile
func profileUsers(name string) []string {
	users := make([]string, 0)
	for _, a := range sqlite.Associations() {
		for _, p := range a.Assos {
			if p == name {
				users = append(users, a.Shortname)
				break
			}
		}
	}
	sort.Strings(users)
	return users
}

func snapshotProfiles() map[string]association.FileTgz {
	association.ProfileLock.Lock()
	defer association.ProfileLock.Unlock()
	out := make(map[string]association.FileTgz, len(association.ActiveProfiles))
	for k, v := range association.ActiveProfiles {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------

func mcpListRouters(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, mcpRouterList, error) {
	routers := sqlite.Routers()
	assos := make(map[string]*sqlite.AssoEntry)
	for _, a := range sqlite.Associations() {
		assos[a.Shortname] = a
	}
	out := mcpRouterList{Routers: make([]mcpRouter, 0, len(routers))}
	for _, r := range routers {
		a := assos[r.Shortname]
		out.Routers = append(out.Routers, mcpRouter{
			Shortname: r.Shortname,
			Hostname:  r.Hostname,
			Family:    r.Family,
			Model:     r.Model,
			Version:   r.Version,
			Profiles:  associationProfiles(a),
			Kafka:     a != nil && a.Kafka == "yes",
		})
	}
	sort.Slice(out.Routers, func(i, j int) bool { return out.Routers[i].Shortname < out.Routers[j].Shortname })
	out.Count = len(out.Routers)
	return nil, out, nil
}

func mcpAddRouter(_ context.Context, _ *mcp.CallToolRequest, in mcpAddRouterInput) (*mcp.CallToolResult, mcpRouterResult, error) {
	in.Hostname = strings.TrimSpace(in.Hostname)
	in.Shortname = strings.TrimSpace(in.Shortname)
	if in.Hostname == "" || in.Shortname == "" {
		return nil, mcpRouterResult{}, fmt.Errorf("hostname and shortname are required")
	}
	if findRouter(in.Shortname) != nil {
		return nil, mcpRouterResult{}, fmt.Errorf("router %s already exists", in.Shortname)
	}

	reply, err := netconf.GetFacts(in.Hostname, sqlite.ActiveCred.NetconfUser, sqlite.ActiveCred.NetconfPwd, collectCfg.cfg.Netconf.Port, 30)
	if err != nil {
		logger.Log.Errorf("MCP: unable to retrieve router %s facts: %v", in.Shortname, err)
		return nil, mcpRouterResult{}, fmt.Errorf("unable to retrieve router facts via NETCONF: %v", err)
	}
	family := findFamily(reply.Model)

	if err := sqlite.AddRouter(in.Hostname, in.Shortname, family, reply.Model, reply.Ver); err != nil {
		logger.Log.Errorf("MCP: unable to add router %s in DB: %v", in.Shortname, err)
		return nil, mcpRouterResult{}, fmt.Errorf("unable to add router in DB: %v", err)
	}
	logger.Log.Infof("MCP: router %s has been successfully added - family %s - model %s - version %s", in.Shortname, family, reply.Model, reply.Ver)
	return nil, mcpRouterResult{
		Shortname: in.Shortname, Hostname: in.Hostname, Family: family, Model: reply.Model, Version: reply.Ver,
		Message: "Router added",
	}, nil
}

func mcpRefreshRouter(_ context.Context, _ *mcp.CallToolRequest, in mcpShortnameInput) (*mcp.CallToolResult, mcpRouterResult, error) {
	r := findRouter(strings.TrimSpace(in.Shortname))
	if r == nil {
		return nil, mcpRouterResult{}, fmt.Errorf("unknown router %s", in.Shortname)
	}

	reply, err := netconf.GetFacts(r.Hostname, sqlite.ActiveCred.NetconfUser, sqlite.ActiveCred.NetconfPwd, collectCfg.cfg.Netconf.Port, 30)
	if err != nil {
		logger.Log.Errorf("MCP: unable to retrieve router %s facts: %v", r.Shortname, err)
		return nil, mcpRouterResult{}, fmt.Errorf("unable to retrieve router facts via NETCONF: %v", err)
	}
	family := findFamily(reply.Model)

	if err := sqlite.UpdateRouter(r.Shortname, family, reply.Model, reply.Ver); err != nil {
		logger.Log.Errorf("MCP: unable to update router %s in DB: %v", r.Shortname, err)
		return nil, mcpRouterResult{}, fmt.Errorf("unable to update router in DB: %v", err)
	}
	logger.Log.Infof("MCP: router %s has been successfully refreshed - family %s - model %s - version %s", r.Shortname, family, reply.Model, reply.Ver)
	return nil, mcpRouterResult{
		Shortname: r.Shortname, Hostname: r.Hostname, Family: family, Model: reply.Model, Version: reply.Ver,
		Message: "Router refreshed",
	}, nil
}

func mcpRemoveRouter(_ context.Context, _ *mcp.CallToolRequest, in mcpShortnameInput) (*mcp.CallToolResult, mcpMessage, error) {
	shortname := strings.TrimSpace(in.Shortname)
	r := findRouter(shortname)
	if r == nil {
		return nil, mcpMessage{}, fmt.Errorf("unknown router %s", shortname)
	}
	associated := findAssociation(shortname) != nil
	if !associated {
		var err error
		if associated, err = sqlite.CheckAsso(shortname); err != nil {
			return nil, mcpMessage{}, fmt.Errorf("unable to check router associations: %v", err)
		}
	}
	if associated {
		return nil, mcpMessage{}, fmt.Errorf("router %s is associated with profile(s): remove the association first", shortname)
	}
	hostname := r.Hostname
	if err := sqlite.DelRouter(shortname); err != nil {
		logger.Log.Errorf("MCP: unable to delete router %s from DB: %v", shortname, err)
		return nil, mcpMessage{}, fmt.Errorf("unable to delete router from DB: %v", err)
	}
	if err := prometheus.DropRouter(hostname); err != nil {
		logger.Log.Errorf("MCP: unable to delete router %s from Prometheus: %v", shortname, err)
		return nil, mcpMessage{Message: "Router deleted from inventory but its Prometheus series could not be dropped"}, nil
	}
	logger.Log.Infof("MCP: router %s has been successfully removed", shortname)
	return nil, mcpMessage{Message: "Router deleted"}, nil
}

func mcpListProfiles(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, mcpProfileList, error) {
	profiles := snapshotProfiles()
	out := mcpProfileList{Profiles: make([]mcpProfileSummary, 0, len(profiles))}
	for name, p := range profiles {
		s := mcpProfileSummary{Name: name, Families: make([]string, 0), Dashboards: make([]string, 0), UsedBy: len(profileUsers(name))}
		if p.Definition != nil {
			s.Description = p.Definition.Description
			s.Dashboards = append(s.Dashboards, p.Definition.GrafaCfg...)
			for f := range profileTelegraf(p.Definition) {
				s.Families = append(s.Families, f)
			}
			sort.Strings(s.Families)
		}
		out.Profiles = append(out.Profiles, s)
	}
	sort.Slice(out.Profiles, func(i, j int) bool { return out.Profiles[i].Name < out.Profiles[j].Name })
	out.Count = len(out.Profiles)
	return nil, out, nil
}

func mcpGetProfile(_ context.Context, _ *mcp.CallToolRequest, in mcpProfileNameInput) (*mcp.CallToolResult, mcpProfileDetails, error) {
	name := strings.TrimSpace(in.Name)
	p, ok := snapshotProfiles()[name]
	if !ok || p.Definition == nil {
		return nil, mcpProfileDetails{}, fmt.Errorf("unknown profile %s", name)
	}
	out := mcpProfileDetails{
		Name:        name,
		Description: p.Definition.Description,
		Version:     p.Definition.Version,
		Telegraf:    profileTelegraf(p.Definition),
		Dashboards:  append([]string{}, p.Definition.GrafaCfg...),
		Routers:     profileUsers(name),
	}
	return nil, out, nil
}

func mcpListAssociations(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, mcpAssociationList, error) {
	assos := sqlite.Associations()
	families := make(map[string]string)
	for _, r := range sqlite.Routers() {
		families[r.Shortname] = r.Family
	}
	out := mcpAssociationList{Associations: make([]mcpAssociation, 0, len(assos))}
	for _, a := range assos {
		out.Associations = append(out.Associations, mcpAssociation{
			Shortname: a.Shortname,
			Family:    families[a.Shortname],
			Profiles:  associationProfiles(a),
			Kafka:     a.Kafka == "yes",
		})
	}
	sort.Slice(out.Associations, func(i, j int) bool { return out.Associations[i].Shortname < out.Associations[j].Shortname })
	out.Count = len(out.Associations)
	return nil, out, nil
}

func mcpAddAssociation(_ context.Context, _ *mcp.CallToolRequest, in mcpAddAssociationInput) (*mcp.CallToolResult, mcpMessage, error) {
	shortname := strings.TrimSpace(in.Shortname)
	r := findRouter(shortname)
	if r == nil {
		return nil, mcpMessage{}, fmt.Errorf("unknown router %s", shortname)
	}

	profiles := make([]string, 0, len(in.Profiles))
	known := snapshotProfiles()
	for _, p := range in.Profiles {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := known[p]; !ok {
			return nil, mcpMessage{}, fmt.Errorf("unknown profile %s", p)
		}
		profiles = append(profiles, p)
	}
	if len(profiles) == 0 {
		return nil, mcpMessage{}, fmt.Errorf("at least one profile is required")
	}

	associated := findAssociation(shortname) != nil
	if !associated {
		var err error
		if associated, err = sqlite.CheckAsso(shortname); err != nil {
			return nil, mcpMessage{}, fmt.Errorf("unable to check router associations: %v", err)
		}
	}
	if associated {
		return nil, mcpMessage{}, fmt.Errorf("router %s is already associated with profile(s): remove the association first", shortname)
	}

	req := &AddProfile{Shortname: shortname, Profiles: profiles, Kafka: in.Kafka}
	valid, errString := checkCompatibility(req, r.Family, r.Version)
	if !valid {
		return nil, mcpMessage{}, fmt.Errorf("incompatibility issue:\n%s", strings.TrimSpace(strings.ReplaceAll(errString, "</br>", "\n")))
	}

	kafkaVal := "no"
	if in.Kafka {
		kafkaVal = "yes"
	}
	if err := sqlite.AddAsso(shortname, profiles, kafkaVal); err != nil {
		logger.Log.Errorf("MCP: unable to add profile(s) to router %s in DB: %v", shortname, err)
		return nil, mcpMessage{}, fmt.Errorf("unable to add profile(s) to router in DB: %v", err)
	}
	logger.Log.Infof("MCP: profile(s) of router %s has been successfully updated", shortname)

	go worker.Collect(collectCfg.cfg)
	go association.ConfigureStack(collectCfg.cfg, r.Family)

	return nil, mcpMessage{Message: fmt.Sprintf("Profile(s) %s associated with router %s; telegraf stack of family %s is being reconfigured in background", strings.Join(profiles, ", "), shortname, r.Family)}, nil
}

func mcpRemoveAssociation(_ context.Context, _ *mcp.CallToolRequest, in mcpShortnameInput) (*mcp.CallToolResult, mcpMessage, error) {
	shortname := strings.TrimSpace(in.Shortname)
	if findAssociation(shortname) == nil {
		return nil, mcpMessage{}, fmt.Errorf("router %s has no association", shortname)
	}
	if err := sqlite.DelAsso(shortname); err != nil {
		logger.Log.Errorf("MCP: unable to delete association of router %s in DB: %v", shortname, err)
		return nil, mcpMessage{}, fmt.Errorf("unable to delete association in DB: %v", err)
	}
	logger.Log.Infof("MCP: association of router %s has been successfully deleted", shortname)

	family := "all"
	if r := findRouter(shortname); r != nil {
		family = r.Family
	}
	go association.ConfigureStack(collectCfg.cfg, family)

	return nil, mcpMessage{Message: fmt.Sprintf("Association removed for router %s; telegraf stack of family %s is being reconfigured in background", shortname, family)}, nil
}

// ---------------------------------------------------------------------------
// Stack / container tools
// ---------------------------------------------------------------------------

// stackContainers lists the containers that belong to the JTS stack, keyed by name
func stackContainers() map[string]string {
	out := map[string]string{"jtso": "jtso", "grafana": "grafana", "prometheus": "prometheus"}
	for _, f := range association.Families {
		out["telegraf_"+f] = "telegraf"
	}
	return out
}

func normalizeContainerName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(strings.ToLower(name)), "/")
}

// cachedStats returns a copy of the last sampled cpu/mem per container
func cachedStats() map[string]map[string]float64 {
	out := make(map[string]map[string]float64)
	if container.Cstats == nil {
		return out
	}
	container.Cstats.StMu.Lock()
	defer container.Cstats.StMu.Unlock()
	for k, v := range container.Cstats.Stats {
		c := make(map[string]float64, len(v))
		for kk, vv := range v {
			c[kk] = vv
		}
		out[k] = c
	}
	return out
}

func mcpGetStackStatus(_ context.Context, _ *mcp.CallToolRequest, in mcpContainerInput) (*mcp.CallToolResult, mcpStackStatus, error) {
	filter := normalizeContainerName(in.Container)
	roles := stackContainers()
	if filter != "" {
		if _, ok := roles[filter]; !ok {
			return nil, mcpStackStatus{}, fmt.Errorf("unknown container %s: expected one of jtso, grafana, prometheus, telegraf_<family>", filter)
		}
	}

	summaries, err := container.ListAllContainers()
	if err != nil {
		return nil, mcpStackStatus{}, fmt.Errorf("unable to query docker: %v", err)
	}
	docker := make(map[string]struct{ state, status, image string })
	for _, s := range summaries {
		for _, n := range s.Names {
			docker[strings.TrimPrefix(n, "/")] = struct{ state, status, image string }{s.State, s.Status, s.Image}
		}
	}

	// routers attached per telegraf instance
	attached := make(map[string][]string)
	out := mcpStackStatus{Containers: make([]mcpContainerStatus, 0)}
	for _, r := range sqlite.Routers() {
		out.TotalRouters++
		if r.Profile == 1 {
			out.Monitored++
			attached[r.Family] = append(attached[r.Family], r.Shortname)
		}
	}
	if ondemand.CC.Run {
		out.OndemandActive = true
		out.OndemandName = ondemand.CC.CurrentProfile.Name
		attached["ondemand"] = append(attached["ondemand"], ondemand.CC.CurrentProfile.RtrList...)
	}

	stats := cachedStats()
	names := make([]string, 0, len(roles))
	for n := range roles {
		if filter == "" || n == filter {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	for _, n := range names {
		cs := mcpContainerStatus{Name: n, Role: roles[n], State: "absent"}
		if d, ok := docker[n]; ok {
			cs.State, cs.Status, cs.Image = d.state, d.status, d.image
		}
		if roles[n] == "telegraf" {
			fam := strings.TrimPrefix(n, "telegraf_")
			list := attached[fam]
			sort.Strings(list)
			count := len(list)
			debug := sqlite.ActiveAdmin.DebugFor(fam) == 1
			cs.Family, cs.Routers, cs.RouterList, cs.Debug = fam, &count, list, &debug
		}
		if s, ok := stats[n]; ok {
			if _, failed := s["error"]; !failed {
				cpu, mem := s["cpu"], s["mem"]
				cs.CPUPercent, cs.MemPercent = &cpu, &mem
			}
		}
		out.Containers = append(out.Containers, cs)
	}
	return nil, out, nil
}

func mcpGetContainerStats(_ context.Context, _ *mcp.CallToolRequest, in mcpContainerInput) (*mcp.CallToolResult, mcpContainerStatsList, error) {
	filter := normalizeContainerName(in.Container)
	stats := cachedStats()
	if filter != "" {
		if _, ok := stats[filter]; !ok {
			return nil, mcpContainerStatsList{}, fmt.Errorf("no statistics for container %s (unknown or not running)", filter)
		}
	}
	out := mcpContainerStatsList{
		Note:  "CPU/memory are sampled by JTSO every minute; values are percentages of host CPU and of the container memory limit.",
		Stats: make([]mcpContainerStats, 0, len(stats)),
	}
	for name, s := range stats {
		if filter != "" && name != filter {
			continue
		}
		_, failed := s["error"]
		out.Stats = append(out.Stats, mcpContainerStats{Name: name, CPUPercent: s["cpu"], MemPercent: s["mem"], Error: failed})
	}
	sort.Slice(out.Stats, func(i, j int) bool { return out.Stats[i].Name < out.Stats[j].Name })
	return nil, out, nil
}

func mcpGetContainerLogs(_ context.Context, _ *mcp.CallToolRequest, in mcpContainerLogsInput) (*mcp.CallToolResult, mcpContainerLogs, error) {
	name := normalizeContainerName(in.Container)
	if _, ok := stackContainers()[name]; !ok {
		return nil, mcpContainerLogs{}, fmt.Errorf("unknown container %s: expected one of jtso, grafana, prometheus, telegraf_<family>", name)
	}
	tail := in.Tail
	if tail <= 0 {
		tail = 100
	}
	if tail > 1000 {
		tail = 1000
	}
	logs, err := container.GetContainerLogsTail(name, tail)
	if err != nil {
		return nil, mcpContainerLogs{}, fmt.Errorf("unable to retrieve logs of %s: %v", name, err)
	}
	return nil, mcpContainerLogs{Container: name, Lines: len(logs), Logs: logs}, nil
}

// ---------------------------------------------------------------------------
// Server construction
// ---------------------------------------------------------------------------

func boolPtr(b bool) *bool { return &b }

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "jtso", Version: config.JtsoVersion}, &mcp.ServerOptions{
		Instructions: "JTSO (Junos Telemetry Stack Orchestrator) management. " +
			"Routers are identified by their shortname. A router must exist in the inventory before profiles can be associated with it, " +
			"and must have no association before it can be removed. Changing associations reconfigures the telegraf stack in the background. " +
			"A profile (list_profiles, get_profile) declares, per platform family, one or more telegraf templates selected by Junos version " +
			"('all' is the fallback, a specific constraint like '>=22.2' wins), plus grafana dashboards. " +
			"Use get_profile_config to see what a template collects (gNMI paths, NETCONF RPCs), how it is processed and what is exported, " +
			"compare_profile_configs to diff two families or versions, and search_profiles to find which profile collects a given sensor. " +
			"These tools read the installed profiles live, so answers always reflect the profiles currently deployed. " +
			"Collected values are in Prometheus as MEASUREMENT_field (see exported_fields of get_profile_config, or list_metrics); " +
			"read them with query_metrics, filtering on the router shortname and tags such as if_name.",
	})

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(false)}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_routers",
		Description: "List the router inventory with family, model, Junos version and the profiles currently associated with each router.",
		Annotations: readOnly,
	}, mcpListRouters)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_router",
		Description: "Add a router to the inventory. JTSO connects to the device via NETCONF to discover its model, family and version.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, mcpAddRouter)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "refresh_router",
		Description: "Re-query a router via NETCONF and update its model, family and version in the inventory (use after a software upgrade).",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(false), IdempotentHint: true},
	}, mcpRefreshRouter)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_router",
		Description: "Remove a router from the inventory and drop its time series from Prometheus. Fails if the router still has a profile association.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, mcpRemoveRouter)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_profiles",
		Description: "List the monitoring profiles available on this JTSO instance with their description, supported platform families and usage count.",
		Annotations: readOnly,
	}, mcpListProfiles)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_profile",
		Description: "Show the details of a profile: description, telegraf configurations per family and version constraint, grafana dashboards and routers using it.",
		Annotations: readOnly,
	}, mcpGetProfile)

	registerProfileContentTools(server, readOnly)
	registerMetricTools(server, readOnly)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_associations",
		Description: "List the current router → profile(s) associations.",
		Annotations: readOnly,
	}, mcpListAssociations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_association",
		Description: "Associate one or more profiles with a router. Profiles must be compatible with the router family/version. The router must not already have an association.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, mcpAddAssociation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_association",
		Description: "Remove all profile associations of a router, which stops its telemetry collection.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, mcpRemoveAssociation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stack_status",
		Description: "State of the JTS stack containers (jtso, grafana, prometheus, telegraf_<family>): docker state, routers attached to each telegraf instance, debug mode, last CPU/memory sample. Optionally filter on one container.",
		Annotations: readOnly,
	}, mcpGetStackStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_container_stats",
		Description: "CPU and memory usage (percent) of the stack containers from the last JTSO sampling. Optionally filter on one container.",
		Annotations: readOnly,
	}, mcpGetContainerStats)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_container_logs",
		Description: "Latest log lines (stdout+stderr) of a stack container, e.g. telegraf_mx to troubleshoot a collection or jtso for orchestrator logs.",
		Annotations: readOnly,
	}, mcpGetContainerLogs)

	return server
}

// newMCPHandler returns the Streamable HTTP handler serving the MCP endpoint
func newMCPHandler() http.Handler {
	server := newMCPServer()
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
}
