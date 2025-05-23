package config

import (
	"fmt"
	"os"

	"jtso/logger"

	"github.com/spf13/viper"
)

// this is config displayed on the main page
const JTSO_VERSION string = "1.0.17"

type PortalConfig struct {
	Https          bool
	ServerCrt      string
	ServerKey      string
	Port           int
	BrowserTimeout int
}

type GrafanaConfig struct {
	Port int
}

type KapacitorConfig struct {
	BootTimeout int
}

type ChronografConfig struct {
	Port int
}

type NetconfConfig struct {
	Port       int
	RpcTimeout int
}

type GnmiConfig struct {
	Port int
}

type EnricherConfig struct {
	Folder   string
	Interval int
	Workers  int
}

type ConfigContainer struct {
	Kapacitor  *KapacitorConfig
	Chronograf *ChronografConfig
	Grafana    *GrafanaConfig
	Enricher   *EnricherConfig
	Portal     *PortalConfig
	Netconf    *NetconfConfig
	Gnmi       *GnmiConfig
}

func NewConfigContainer(f string) *ConfigContainer {
	viper.SetConfigFile(f)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Log.Errorf("Fatal error config file %v", err)
		fmt.Println("Fatal error config file: default \n", err)
		os.Exit(1)
	}

	logger.Log.Info("Read configuration file")

	// Ser default value for grafana
	viper.SetDefault("modules.grafana.port", 8080)

	// Ser default value for chronograf
	viper.SetDefault("modules.chronograf.port", 8081)

	// Ser default value for kapacitor
	viper.SetDefault("modules.kapacitor.timeout", 15)

	// Ser default value for portal
	viper.SetDefault("modules.portal.https", false)
	viper.SetDefault("modules.portal.server_crt", "")
	viper.SetDefault("modules.portal.server_key", "")
	viper.SetDefault("modules.portal.port", 8082)
	viper.SetDefault("modules.portal.browsertimeout", 40)

	// Ser default value for enricher
	viper.SetDefault("modules.enricher.folder", "/var/metadata/")
	viper.SetDefault("modules.enricher.interval", 240)
	viper.SetDefault("modules.enricher.workers", 4)

	// Set default value for Netconf
	viper.SetDefault("protocols.netconf.port", 830)
	viper.SetDefault("protocols.netconf.rpc_timeout", 60)

	// Set default value for gnmi
	viper.SetDefault("protocols.gnmi.port", 9339)

	return &ConfigContainer{
		Grafana: &GrafanaConfig{
			Port: viper.GetInt("modules.grafana.port"),
		},
		Chronograf: &ChronografConfig{
			Port: viper.GetInt("modules.chronograf.port"),
		},
		Kapacitor: &KapacitorConfig{
			BootTimeout: viper.GetInt("modules.kapacitor.timeout"),
		},
		Portal: &PortalConfig{
			Port:           viper.GetInt("modules.portal.port"),
			Https:          viper.GetBool("modules.portal.https"),
			ServerCrt:      viper.GetString("modules.portal.server_crt"),
			ServerKey:      viper.GetString("modules.portal.server_key"),
			BrowserTimeout: viper.GetInt("modules.portal.browsertimeout"),
		},
		Enricher: &EnricherConfig{
			Folder:   viper.GetString("modules.enricher.folder"),
			Interval: viper.GetInt("modules.enricher.interval"),
			Workers:  viper.GetInt("modules.enricher.workers"),
		},
		Netconf: &NetconfConfig{
			Port:       viper.GetInt("protocols.netconf.port"),
			RpcTimeout: viper.GetInt("protocols.netconf.rpc_timeout"),
		},
		Gnmi: &GnmiConfig{
			Port: viper.GetInt("protocols.gnmi.port"),
		},
	}
}
