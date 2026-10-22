package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jtso/logger"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// prometheusURL is a variable so tests can point it to a fake server
var prometheusURL = "http://prometheus:9090"

const (
	// Admin TSDB API endpoints (require --web.enable-admin-api on the Prometheus server)
	deleteSeriesPath    = "/api/v1/admin/tsdb/delete_series"
	cleanTombstonesPath = "/api/v1/admin/tsdb/clean_tombstones"
)

// httpClient is reused for all calls to the Prometheus Admin API
var httpClient = &http.Client{Timeout: 20 * time.Second}

// deleteSeries asks Prometheus to delete every series matching the given selectors.
// Once deleted, clean_tombstones is triggered to actually reclaim the data on disk
// (delete_series only marks the data for deletion).
func deleteSeries(matchers []string) error {
	// Build the form values - one match[] entry per selector
	form := url.Values{}
	for _, m := range matchers {
		form.Add("match[]", m)
	}

	resp, err := httpClient.PostForm(prometheusURL+deleteSeriesPath, form)
	if err != nil {
		logger.Log.Errorf("Unable to establish prometheus connexion: %v", err)
		return err
	}
	defer resp.Body.Close()

	// The Admin API returns 204 No Content on success
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("delete_series failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		logger.Log.Errorf("No valid response from prometheus: %v", err)
		return err
	}

	// Reclaim the disk space immediately so the data is really gone
	return cleanTombstones()
}

// cleanTombstones removes the deletion tombstones, effectively purging the data
// previously marked for deletion by delete_series.
func cleanTombstones() error {
	resp, err := httpClient.Post(prometheusURL+cleanTombstonesPath, "application/x-www-form-urlencoded", nil)
	if err != nil {
		logger.Log.Errorf("Unable to establish prometheus connexion: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("clean_tombstones failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		logger.Log.Errorf("No valid response from prometheus: %v", err)
		return err
	}

	return nil
}

// EmptyDB deletes all series stored in Prometheus.
func EmptyDB() error {
	// {__name__=~".+"} matches every series whatever its metric name
	if err := deleteSeries([]string{`{__name__=~".+"}`}); err != nil {
		return err
	}
	logger.Log.Infof("Prometheus TSDB has been successfully empty")
	return nil
}

// DropMeasurement deletes all series belonging to a given measurement.
// With the telegraf prometheus_client output the measurement name is used as a
// prefix of the metric name, so we match every metric name starting with it.
func DropMeasurement(measurement string) error {
	if measurement == "" {
		return fmt.Errorf("measurement name cannot be empty")
	}

	matcher := fmt.Sprintf(`{__name__=~"%s.*"}`, measurement)
	if err := deleteSeries([]string{matcher}); err != nil {
		return err
	}

	logger.Log.Infof("Prometheus measurement %s has been successfully cleared", measurement)
	return nil
}

// DropRouter deletes all series tagged with the given device.
func DropRouter(r string) error {
	if r == "" {
		return fmt.Errorf("router name cannot be empty")
	}

	matcher := fmt.Sprintf(`{device="%s"}`, r)
	if err := deleteSeries([]string{matcher}); err != nil {
		return err
	}

	logger.Log.Infof("Router %s has been successfully removed from Prometheus", r)
	return nil
}

// ---------------------------------------------------------------------------
// Read API
// ---------------------------------------------------------------------------

// queryTimeout bounds the evaluation time on the Prometheus side
const queryTimeout = "10s"

// Point is one sample; Value is nil for NaN/Inf, which JSON cannot carry
type Point struct {
	Time  time.Time
	Value *float64
}

// Series is a labelled result: one point for instant queries, several for range queries
type Series struct {
	Labels map[string]string
	Points []Point
}

type apiResponse struct {
	Status    string          `json:"status"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
}

type apiResult struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Value  []any             `json:"value"`
		Values [][]any           `json:"values"`
	} `json:"result"`
}

func apiGet(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, prometheusURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to reach prometheus: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("invalid prometheus response (status %d)", resp.StatusCode)
	}
	if r.Status != "success" {
		return nil, fmt.Errorf("prometheus %s: %s", r.ErrorType, r.Error)
	}
	return r.Data, nil
}

func parsePoint(v []any) (Point, bool) {
	if len(v) != 2 {
		return Point{}, false
	}
	ts, ok1 := v[0].(float64)
	s, ok2 := v[1].(string)
	if !ok1 || !ok2 {
		return Point{}, false
	}
	p := Point{Time: time.Unix(0, int64(ts*1e9)).UTC()}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		p.Value = &f
	}
	return p, true
}

func parseResult(data json.RawMessage) ([]Series, error) {
	var r apiResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.ResultType != "vector" && r.ResultType != "matrix" {
		return nil, fmt.Errorf("unexpected result type %s", r.ResultType)
	}
	out := make([]Series, 0, len(r.Result))
	for _, res := range r.Result {
		s := Series{Labels: res.Metric, Points: []Point{}}
		if res.Value != nil {
			if p, ok := parsePoint(res.Value); ok {
				s.Points = append(s.Points, p)
			}
		}
		for _, v := range res.Values {
			if p, ok := parsePoint(v); ok {
				s.Points = append(s.Points, p)
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// Query evaluates an instant PromQL query at the given time (now when zero)
func Query(ctx context.Context, promql string, at time.Time) ([]Series, error) {
	params := url.Values{"query": {promql}, "timeout": {queryTimeout}}
	if !at.IsZero() {
		params.Set("time", strconv.FormatInt(at.Unix(), 10))
	}
	data, err := apiGet(ctx, "/api/v1/query", params)
	if err != nil {
		return nil, err
	}
	return parseResult(data)
}

// QueryRange evaluates a PromQL range query
func QueryRange(ctx context.Context, promql string, start, end time.Time, step time.Duration) ([]Series, error) {
	params := url.Values{
		"query":   {promql},
		"start":   {strconv.FormatInt(start.Unix(), 10)},
		"end":     {strconv.FormatInt(end.Unix(), 10)},
		"step":    {strconv.FormatInt(int64(step.Seconds()), 10)},
		"timeout": {queryTimeout},
	}
	data, err := apiGet(ctx, "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}
	return parseResult(data)
}

// MetricNames returns every metric name known by Prometheus
func MetricNames(ctx context.Context) ([]string, error) {
	data, err := apiGet(ctx, "/api/v1/label/__name__/values", url.Values{})
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	return names, nil
}
