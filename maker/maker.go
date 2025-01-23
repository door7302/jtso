package maker

import (
	"encoding/json"
	"jtso/logger"
	"os"
)

const TELEGRAF_ROOT_PATH string = "/var/shared/telegraf/"

func GenerateTemplate(filename string) {
	var config TelegrafConfig

	// Marshal the struct to JSON with no values (default values)
	outputJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logger.Log.Errorf("Unable to generate JSON template config: %v", err)
		return
	}

	// Create or open the file
	file, err := os.Create(TELEGRAF_ROOT_PATH + filename + ".json")
	if err != nil {
		logger.Log.Errorf("Error creating JSON template file: %v", err)
		return
	}
	defer file.Close()

	// Write the JSON data to the file
	_, err = file.Write(outputJSON)
	if err != nil {
		logger.Log.Errorf("Error writing to JSON template file: %v", err)
		return
	}

	// Optional: Print success message
	logger.Log.Infof("Successfully generate JSON template to %s.json\n", filename)
}
