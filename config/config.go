package config

import (
	"fmt"
	"os"
	"time"

	"jtso/logger"

	"github.com/spf13/viper"
)

type PortalConfig struct {
	Port int
}

type InstanceConfig struct {
	Name string
	Rtrs []string
}

type NetconfConfig struct {
	Port       int
	RpcTimeout int
	User       string
	Pwd        string
}

type GnmiConfig struct {
	Port       int
	User       string
	Pwd        string
	UseTls     bool
	SkipVerify bool
}

type EnricherConfig struct {
	Folder   string
	Interval time.Duration
	Workers  int
	Port     int
}
type ConfigContainer struct {
	Instances []*InstanceConfig
	Enricher  *EnricherConfig
	Portal    *PortalConfig
	Netconf   *NetconfConfig
	Gnmi      *GnmiConfig
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

	// Set default value forthe 3 instances
	viper.SetDefault("modules.telegraf_instances.instance1.name", "")
	viper.SetDefault("modules.telegraf_instances.instance2.name", "")
	viper.SetDefault("modules.telegraf_instances.instance3.name", "")
	viper.SetDefault("modules.telegraf_instances.instance1.routers", []string{})
	viper.SetDefault("modules.telegraf_instances.instance2.routers", []string{})
	viper.SetDefault("modules.telegraf_instances.instance3.routers", []string{})

	// Ser default value for portal
	viper.SetDefault("modules.portal.port", 8080)

	// Ser default value for enricher
	viper.SetDefault("modules.enricher.folder", "./")
	viper.SetDefault("modules.enricher.interval", 720*time.Minute)
	viper.SetDefault("modules.enricher.workers", 4)
	viper.SetDefault("modules.enricher.port", 10000)

	// Set default value for Netconf
	viper.SetDefault("protocols.netconf.port", 830)
	viper.SetDefault("protocols.netconf.rpc_timeout", 60)
	viper.SetDefault("protocols.netconf.username", "lab")
	viper.SetDefault("protocols.netconf.password", "lab123")

	// Set default value for gnmi
	viper.SetDefault("protocols.gnmi.port", 9339)
	viper.SetDefault("protocols.gnmi.username", "lab")
	viper.SetDefault("protocols.gnmi.password", "lab123")
	viper.SetDefault("protocols.gnmi.use_tls", false)
	viper.SetDefault("protocols.gnmi.skip_verify", true)

	inst := make([]*InstanceConfig, 3)
	inst[0] = &InstanceConfig{
		Name: viper.GetString("modules.telegraf_instances.instance1.name"),
		Rtrs: viper.GetStringSlice("modules.telegraf_instances.instance1.routers"),
	}
	inst[1] = &InstanceConfig{
		Name: viper.GetString("modules.telegraf_instances.instance2.name"),
		Rtrs: viper.GetStringSlice("modules.telegraf_instances.instance2.routers"),
	}
	inst[2] = &InstanceConfig{
		Name: viper.GetString("modules.telegraf_instances.instance3.name"),
		Rtrs: viper.GetStringSlice("modules.telegraf_instances.instance3.routers"),
	}

	return &ConfigContainer{
		Instances: inst,
		Portal: &PortalConfig{
			Port: viper.GetInt("modules.portal.port"),
		},
		Enricher: &EnricherConfig{
			Folder:   viper.GetString("modules.enricher.folder"),
			Interval: viper.GetDuration("modules.enricher.interval") * time.Minute,
			Workers:  viper.GetInt("modules.enricher.workers"),
			Port:     viper.GetInt("modules.enricher.port"),
		},
		Netconf: &NetconfConfig{
			Port:       viper.GetInt("protocols.netconf.port"),
			RpcTimeout: viper.GetInt("protocols.netconf.rpc_timeout"),
			User:       viper.GetString("protocols.netconf.username"),
			Pwd:        viper.GetString("protocols.netconf.password"),
		},
		Gnmi: &GnmiConfig{
			Port:       viper.GetInt("protocols.gnmi.port"),
			User:       viper.GetString("protocols.gnmi.username"),
			Pwd:        viper.GetString("protocols.gnmi.password"),
			UseTls:     viper.GetBool("protocols.gnmi.use_tls"),
			SkipVerify: viper.GetBool("protocols.gnmi.skip_verify"),
		},
	}
}
