package ondemand

import (
	"encoding/json"
	"fmt"
	"jtso/logger"
	"os"
	"path/filepath"
	"strings"
)

type (
	FieldEntry struct {
		Name        string   `json:"name"`
		Monitor     bool     `json:"monitor"`
		Rate        bool     `json:"rate"`
		Convert     bool     `json:"convert"`
		InheritTags []string `json:"inherit_tags"`
	}

	Entry struct {
		Path     string       `json:"path"`
		Interval int          `json:"interval"`
		Aliases  []string     `json:"aliases"`
		Fields   []FieldEntry `json:"fields"`
		Tags     []string     `json:"tags"`
	}

	RunningProfile struct {
		Name    string   `json:"name"`
		RtrList []string `json:"routers"`
		Entries []Entry  `json:"entries"`
	}

	CurrentContext struct {
		Run            bool           `json:"run"`
		CurrentConfig  string         `json:"currentConfig"`
		CurrentProfile RunningProfile `json:"currentProfile"`
	}
)

const (
	PATH_ONDEMAND string = "/var/ondemand/"
)

func Load(f string) (error, RunningProfile) {
	var profile RunningProfile

	logger.Log.Infof("Load Ondemand configuration %s", f)

	filePath := filepath.Join(PATH_ONDEMAND, f)

	data, err := os.ReadFile(filePath + ".json")
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err), RunningProfile{}
	}

	// Unmarshal JSON into RunningProfile struct
	err = json.Unmarshal(data, &profile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err), RunningProfile{}
	}

	return nil, profile
}

func Save(f string, profile RunningProfile) error {
	logger.Log.Infof("Save Ondemand configuration %s", f)

	filePath := filepath.Join(PATH_ONDEMAND, f)

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(filePath+".json", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func ListConfigs() ([]string, error) {
	var configs []string

	// Read directory entries
	entries, err := os.ReadDir(PATH_ONDEMAND)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", PATH_ONDEMAND, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			// Remove the .json extension
			configName := strings.TrimSuffix(name, ".json")
			configs = append(configs, configName)
		}
	}

	return configs, nil
}
