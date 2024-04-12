package config

import (
	"fmt"
	"os"
	"time"

	"jtso/logger"

	"github.com/spf13/viper"
)

const JTSO_VERSION string = "1.0.1"

type PortalConfig struct {
	Https     bool
	ServerCrt string
	ServerKey string
	Port      int
}

type GrafanaConfig struct {
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
	Interval time.Duration
	Workers  int
}
type ConfigContainer struct {
	Grafana  *GrafanaConfig
	Enricher *EnricherConfig
	Portal   *PortalConfig
	Netconf  *NetconfConfig
	Gnmi     *GnmiConfig
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

	// Ser default value for portal
	viper.SetDefault("modules.grafana.port", 8080)

	// Ser default value for portal
	viper.SetDefault("modules.portal.https", false)
	viper.SetDefault("modules.portal.server_crt", "")
	viper.SetDefault("modules.portal.server_key", "")
	viper.SetDefault("modules.portal.port", 8081)

	// Ser default value for enricher
	viper.SetDefault("modules.enricher.folder", "/var/metadata/")
	viper.SetDefault("modules.enricher.interval", 720*time.Minute)
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
		Portal: &PortalConfig{
			Port:      viper.GetInt("modules.portal.port"),
			Https:     viper.GetBool("modules.portal.https"),
			ServerCrt: viper.GetString("modules.portal.server_crt"),
			ServerKey: viper.GetString("modules.portal.server_key"),
		},
		Enricher: &EnricherConfig{
			Folder:   viper.GetString("modules.enricher.folder"),
			Interval: viper.GetDuration("modules.enricher.interval") * time.Minute,
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
