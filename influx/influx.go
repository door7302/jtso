package influx

import (
	"fmt"
	"jtso/logger"
	"strconv"
	"strings"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
)

const (
	influxDBURL      = "http://influxdb:8086"
	influxDBDatabase = "jtsdb"
	influxRetention  = "autogen"
	DefaultRetention = "30d"
)

func RetentionDurationEqual(a, b string) (bool, error) {
	da, err := normalizeDuration(a)
	if err != nil {
		return false, err
	}

	db, err := normalizeDuration(b)
	if err != nil {
		return false, err
	}

	return da == db, nil
}

func normalizeDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func EmptyDB() error {
	// Create a new HTTP client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxDBURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish influxdb connexion: %v", err)
		return err

	}
	defer c.Close()

	// Create a new query
	q := client.Query{
		Command:  "DROP SERIES FROM /.*/",
		Database: influxDBDatabase,
	}

	// Execute the query
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		logger.Log.Infof("Influxdb %s has been successfully empty", influxDBDatabase)
		return nil
	} else {
		logger.Log.Errorf("No response from influxdb: %v", err)
		return err
	}
}

// DropMeasurement drops all data from a specific measurement in InfluxDB
func DropMeasurement(measurement string) error {
	if measurement == "" {
		return fmt.Errorf("measurement name cannot be empty")
	}

	// Create a new HTTP client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxDBURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish InfluxDB connection: %v", err)
		return err
	}
	defer c.Close()

	// Prepare the query to drop all series from the measurement
	q := client.Query{
		Command:  fmt.Sprintf("DROP SERIES FROM \"%s\"", measurement),
		Database: influxDBDatabase,
	}

	// Execute the query
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		logger.Log.Infof("InfluxDB measurement %s has been successfully cleared", measurement)
		return nil
	} else {
		if err != nil {
			logger.Log.Errorf("Failed to execute DROP SERIES query: %v", err)
			return err
		} else if response.Error() != nil {
			logger.Log.Errorf("InfluxDB response error: %v", response.Error())
			return response.Error()
		}
	}

	return nil
}

func DropRouter(r string) error {
	// Create a new HTTP client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxDBURL,
	})
	if err != nil {
		logger.Log.Errorf("Enable to establish influxdb connexion: %v", err)
		return err

	}
	defer c.Close()

	// Create a new query
	q := client.Query{
		Command:  fmt.Sprintf("DROP SERIES FROM /.*/ WHERE device='%s'", strings.ReplaceAll(r, "'", "\\'"))  ,
		Database: influxDBDatabase,
	}
	// Execute the query
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		logger.Log.Infof("Router %s has been successfully removed from Influxdb", r)
		return nil
	} else {
		logger.Log.Errorf("No response from influxdb: %v", err)
		return err
	}
}

func GetRetentionPolicyDuration() (string, error) {
	// Create a new HTTP client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxDBURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish influxdb connexion: %v", err)
		return "", err
	}
	defer c.Close()

	// Create the query
	q := client.Query{
		Command:  "SHOW RETENTION POLICIES ON " + influxDBDatabase,
		Database: influxDBDatabase,
	}

	// Execute the query
	response, err := c.Query(q)
	if err != nil {
		logger.Log.Errorf("Query error: %v", err)
		return "", err
	}
	if response.Error() != nil {
		logger.Log.Errorf("Response error: %v", response.Error())
		return "", response.Error()
	}

	// Find the autogen retention policy
	for _, row := range response.Results[0].Series {
		for _, val := range row.Values {
			rpName := val[0].(string)
			if rpName == influxRetention {
				duration := val[1].(string)
				logger.Log.Infof("Retention policy %s has duration: %s", rpName, duration)
				return duration, nil
			}
		}
	}

	logger.Log.Errorf("Retention policy %s not found", influxRetention)
	return "", fmt.Errorf("retention policy %s not found", influxRetention)
}

func AlterRetentionPolicyDuration(duration string) error {
	// Create a new HTTP client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxDBURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish influxdb connexion: %v", err)
		return err
	}
	defer c.Close()

	// Build ALTER RETENTION POLICY query (only changing duration)
	cmd := fmt.Sprintf(
		`ALTER RETENTION POLICY "%s" ON "%s" DURATION %s`,
		influxRetention, influxDBDatabase, duration,
	)

	// Create the query
	q := client.Query{
		Command:  cmd,
		Database: influxDBDatabase,
	}

	// Execute the query
	response, err := c.Query(q)
	if err != nil {
		logger.Log.Errorf("Query error: %v", err)
		return err
	}
	if response.Error() != nil {
		logger.Log.Errorf("Response error: %v", response.Error())
		return response.Error()
	}

	logger.Log.Infof("Retention policy %s duration modified successfully: Duration=%s", influxRetention, duration)
	return nil
}
