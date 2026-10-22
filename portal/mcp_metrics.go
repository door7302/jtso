package portal

import (
	"context"
	"fmt"
	"jtso/prometheus"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Guardrails for metric queries issued on behalf of MCP clients
const (
	maxQueryWindow   = 15 * time.Minute
	defaultWindow    = 5 * time.Minute
	maxQuerySeries   = 50
	maxSeriesPoints  = 60
	minSeriesStep    = 15 * time.Second
	maxQueryMetrics  = 5
	maxMetricMatches = 200
	queryCallTimeout = 20 * time.Second
)

var (
	promMetricRe = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	promLabelRe  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// aggregations maps the supported reductions to their PromQL over-time function
var aggregations = map[string]string{
	"last": "last_over_time",
	"avg":  "avg_over_time",
	"min":  "min_over_time",
	"max":  "max_over_time",
	"p95":  "quantile_over_time",
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type mcpListMetricsInput struct {
	Filter string `json:"filter,omitempty" jsonschema:"case-insensitive substring of the metric name, e.g. 'INTERFACES_' or 'octets_rate'"`
}

type mcpMetricList struct {
	Count     int      `json:"count"`
	Truncated bool     `json:"truncated,omitempty"`
	Metrics   []string `json:"metrics"`
}

type mcpQueryInput struct {
	Metrics     []string          `json:"metrics" jsonschema:"1 to 5 exact Prometheus metric names (MEASUREMENT_field, e.g. INTERFACES_in_octets_rate)"`
	Router      string            `json:"router,omitempty" jsonschema:"router shortname, matched on the SHORTNAME label"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"exact label filters, e.g. {\"if_name\": \"et-5/5/0\"}"`
	LabelsRegex map[string]string `json:"labels_regex,omitempty" jsonschema:"regex label filters (fully anchored), e.g. {\"if_name\": \"et-5/5/.*\"}"`
	Window      string            `json:"window,omitempty" jsonschema:"time window, Go/Prometheus duration such as 30s, 5m, 15m (default 5m, max 15m)"`
	Aggregation string            `json:"aggregation,omitempty" jsonschema:"last (default), avg, min, max, p95 or series (raw points over the window)"`
	End         string            `json:"end,omitempty" jsonschema:"RFC3339 end of the window; default now"`
}

type mcpPoint struct {
	Time  string   `json:"time"`
	Value *float64 `json:"value" jsonschema:"null when the sample is NaN/Inf"`
}

type mcpSeries struct {
	Metric string            `json:"metric"`
	Labels map[string]string `json:"labels"`
	Value  *float64          `json:"value,omitempty" jsonschema:"aggregated value (all aggregations except series)"`
	Points []mcpPoint        `json:"points,omitempty" jsonschema:"samples when aggregation is series"`
}

type mcpQueryResult struct {
	Queries     []string    `json:"queries" jsonschema:"PromQL actually executed"`
	Aggregation string      `json:"aggregation"`
	Window      string      `json:"window"`
	End         string      `json:"end"`
	Count       int         `json:"count"`
	Series      []mcpSeries `json:"series"`
	Note        string      `json:"note,omitempty"`
}

// ---------------------------------------------------------------------------
// PromQL construction
// ---------------------------------------------------------------------------

// buildSelector returns metric{...} with validated names and escaped values
func buildSelector(metric, router string, labels, labelsRegex map[string]string) (string, error) {
	if !promMetricRe.MatchString(metric) {
		return "", fmt.Errorf("invalid metric name %q", metric)
	}
	matchers := make([]string, 0, len(labels)+len(labelsRegex)+1)
	if router != "" {
		matchers = append(matchers, "SHORTNAME="+strconv.Quote(router))
	}
	add := func(m map[string]string, op string) error {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !promLabelRe.MatchString(k) || strings.HasPrefix(k, "__") {
				return fmt.Errorf("invalid label name %q", k)
			}
			if op == "=~" {
				if _, err := regexp.Compile(m[k]); err != nil {
					return fmt.Errorf("invalid regex for label %s: %v", k, err)
				}
			}
			matchers = append(matchers, k+op+strconv.Quote(m[k]))
		}
		return nil
	}
	if err := add(labels, "="); err != nil {
		return "", err
	}
	if err := add(labelsRegex, "=~"); err != nil {
		return "", err
	}
	return metric + "{" + strings.Join(matchers, ",") + "}", nil
}

func parseWindow(s string) (time.Duration, error) {
	if s == "" {
		return defaultWindow, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid window %q: use a duration like 30s, 5m or 15m", s)
	}
	if d < minSeriesStep {
		return 0, fmt.Errorf("window must be at least %s", minSeriesStep)
	}
	if d > maxQueryWindow {
		return 0, fmt.Errorf("window %s exceeds the %s limit: narrow the time frame", d, maxQueryWindow)
	}
	return d.Truncate(time.Second), nil
}

// promDuration renders a duration in seconds, always accepted by PromQL
func promDuration(d time.Duration) string {
	return strconv.FormatInt(int64(d.Seconds()), 10) + "s"
}

func aggregateExpr(agg, selector string, window time.Duration) string {
	rng := selector + "[" + promDuration(window) + "]"
	if agg == "p95" {
		return "quantile_over_time(0.95, " + rng + ")"
	}
	return aggregations[agg] + "(" + rng + ")"
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func mcpListMetrics(ctx context.Context, _ *mcp.CallToolRequest, in mcpListMetricsInput) (*mcp.CallToolResult, mcpMetricList, error) {
	ctx, cancel := context.WithTimeout(ctx, queryCallTimeout)
	defer cancel()
	names, err := prometheus.MetricNames(ctx)
	if err != nil {
		return nil, mcpMetricList{}, err
	}
	filter := strings.ToLower(strings.TrimSpace(in.Filter))
	out := mcpMetricList{Metrics: []string{}}
	for _, n := range names {
		if filter != "" && !strings.Contains(strings.ToLower(n), filter) {
			continue
		}
		if len(out.Metrics) >= maxMetricMatches {
			out.Truncated = true
			break
		}
		out.Metrics = append(out.Metrics, n)
	}
	sort.Strings(out.Metrics)
	out.Count = len(out.Metrics)
	return nil, out, nil
}

func mcpQueryMetrics(ctx context.Context, _ *mcp.CallToolRequest, in mcpQueryInput) (*mcp.CallToolResult, mcpQueryResult, error) {
	if len(in.Metrics) == 0 || len(in.Metrics) > maxQueryMetrics {
		return nil, mcpQueryResult{}, fmt.Errorf("give between 1 and %d metric names", maxQueryMetrics)
	}
	router := strings.TrimSpace(in.Router)
	if router != "" && findRouter(router) == nil {
		return nil, mcpQueryResult{}, fmt.Errorf("unknown router %s", router)
	}
	agg := strings.ToLower(strings.TrimSpace(in.Aggregation))
	if agg == "" {
		agg = "last"
	}
	if _, ok := aggregations[agg]; !ok && agg != "series" {
		return nil, mcpQueryResult{}, fmt.Errorf("unsupported aggregation %q: use last, avg, min, max, p95 or series", agg)
	}
	window, err := parseWindow(strings.TrimSpace(in.Window))
	if err != nil {
		return nil, mcpQueryResult{}, err
	}
	end := time.Now().UTC().Truncate(time.Second)
	if s := strings.TrimSpace(in.End); s != "" {
		if end, err = time.Parse(time.RFC3339, s); err != nil {
			return nil, mcpQueryResult{}, fmt.Errorf("invalid end %q: expected RFC3339, e.g. 2026-10-22T10:15:00Z", s)
		}
		end = end.UTC()
	}

	ctx, cancel := context.WithTimeout(ctx, queryCallTimeout)
	defer cancel()

	out := mcpQueryResult{Aggregation: agg, Window: window.String(), End: end.Format(time.RFC3339), Queries: []string{}, Series: []mcpSeries{}}
	for _, metric := range in.Metrics {
		metric = strings.TrimSpace(metric)
		sel, err := buildSelector(metric, router, in.Labels, in.LabelsRegex)
		if err != nil {
			return nil, mcpQueryResult{}, err
		}

		// Cardinality check first so a broad selector never pulls a large result set
		countQ := "count(last_over_time(" + sel + "[" + promDuration(window) + "]))"
		cnt, err := prometheus.Query(ctx, countQ, end)
		if err != nil {
			return nil, mcpQueryResult{}, err
		}
		n := 0
		if len(cnt) > 0 && len(cnt[0].Points) > 0 && cnt[0].Points[0].Value != nil {
			n = int(*cnt[0].Points[0].Value)
		}
		if len(out.Series)+n > maxQuerySeries {
			return nil, mcpQueryResult{}, fmt.Errorf("%s matches %d series (limit %d in total): add a router and/or label filters", sel, n, maxQuerySeries)
		}
		if n == 0 {
			continue
		}

		var series []prometheus.Series
		if agg == "series" {
			step := window / maxSeriesPoints
			if step < minSeriesStep {
				step = minSeriesStep
			}
			out.Queries = append(out.Queries, sel)
			series, err = prometheus.QueryRange(ctx, sel, end.Add(-window), end, step)
		} else {
			q := aggregateExpr(agg, sel, window)
			out.Queries = append(out.Queries, q)
			series, err = prometheus.Query(ctx, q, end)
		}
		if err != nil {
			return nil, mcpQueryResult{}, err
		}
		for _, s := range series {
			delete(s.Labels, "__name__")
			ms := mcpSeries{Metric: metric, Labels: s.Labels}
			if agg == "series" {
				for _, p := range s.Points {
					ms.Points = append(ms.Points, mcpPoint{Time: p.Time.Format(time.RFC3339), Value: p.Value})
				}
			} else if len(s.Points) > 0 {
				ms.Value = s.Points[0].Value
			}
			out.Series = append(out.Series, ms)
		}
	}
	out.Count = len(out.Series)
	if out.Count == 0 {
		out.Note = "no data: check the metric name with list_metrics and the label values (labels are case-sensitive)"
	}
	return nil, out, nil
}

func registerMetricTools(server *mcp.Server, readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_metrics",
		Description: "List metric names stored in Prometheus, optionally filtered by substring. " +
			"Names are MEASUREMENT_field where MEASUREMENT is the telegraf subscription name and field the (renamed) field, e.g. INTERFACES_in_octets_rate.",
		Annotations: readOnly,
	}, mcpListMetrics)

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_metrics",
		Description: "Read telemetry values from Prometheus over a short window (max 15m, max 50 series). " +
			"Filter by router shortname (SHORTNAME label) and labels: telegraf tags (e.g. if_name, queue_name) and enrichment tags " +
			"(device, SHORTNAME, FAMILY, MODEL, VERSION, LINKNAME, DESC...). Aggregation: last, avg, min, max, p95, or series for raw points. " +
			"Fields ending in _octets_rate are bytes per second (multiply by 8 for bit/s); _pkts_rate are packets per second.",
		Annotations: readOnly,
	}, mcpQueryMetrics)
}
