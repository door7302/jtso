package influx

import (
	"jtso/logger"

	client "github.com/influxdata/influxdb1-client/v2"
)

const (
	influxDBURL      = "http://influxdb:8086"
	influxDBDatabase = "jtsdb"
)

func EmptyDB() error {
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
		Command:  "DROP SERIES FROM /.*/ WHERE device='" + r + "'",
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
