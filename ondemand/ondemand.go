package ondemand

import (
	"encoding/json"
	"fmt"
	"jtso/logger"
	"os"
	"path/filepath"
	"strings"
)

type ProcessorConvert struct {
	Enable bool   `json:"enable"`
	Type   string `json:"type"`
}

type ProcessorRate struct {
	Enable bool `json:"enable"`
	Factor int  `json:"factor"`
}

type ProcessorAlarm struct {
	Enable    bool    `json:"enable"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Threshold float32 `json:"threshold"`
	Operator  string  `json:"operator"`
}

type (
	FieldEntry struct {
		Name        string           `json:"name"`
		Monitor     bool             `json:"monitor"`
		Rate        ProcessorRate    `json:"rate"`
		Convert     ProcessorConvert `json:"convert"`
		Alarming    ProcessorAlarm   `json:"alarming"`
		InheritTags []string         `json:"inherit_tags"`
	}

	Entry struct {
		Path     string       `json:"path"`
		Interval int          `json:"interval"`
		Aliases  []string     `json:"aliases"`
		Fields   []FieldEntry `json:"fields"`
	}

	RunningProfile struct {
		Name    string   `json:"name"`
		RtrList []string `json:"routers"`
		Entries []Entry  `json:"entries"`
	}

	CurrentContext struct {
		Run            bool           `json:"run"`
		CurrentProfile RunningProfile `json:"currentProfile"`
	}
)

var CC CurrentContext

const (
	PATH_ONDEMAND string = "/var/ondemand/"
)

func init() {
	CC = CurrentContext{
		Run: false,
		CurrentProfile: RunningProfile{
			Name:    "no-name",
			RtrList: make([]string, 0),
			Entries: make([]Entry, 0),
		},
	}
}

func Load(f string) (error, RunningProfile) {
	var profile RunningProfile

	logger.Log.Infof("Load Ondemand configuration %s", f)

	filePath := filepath.Join(PATH_ONDEMAND, f)
	// Prevent path traversal
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(PATH_ONDEMAND)) {
		return fmt.Errorf("invalid file path: path traversal detected"), RunningProfile{}
	}

	data, err := os.ReadFile(cleanPath + ".json")
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
	// Prevent path traversal
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(PATH_ONDEMAND)) {
		return fmt.Errorf("invalid file path: path traversal detected")
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(cleanPath+".json", data, 0644)
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
