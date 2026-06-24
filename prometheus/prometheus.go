package prometheus

import (
	"fmt"
	"io"
	"jtso/logger"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	prometheusURL = "http://prometheus:9090"
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
