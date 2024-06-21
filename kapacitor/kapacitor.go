package kapacitor

import (
	"jtso/logger"
	"os"
	"strconv"

	client "github.com/influxdata/kapacitor/client/v1"
)

const (
	kapacitorURL = "http://kapacitor:9092"
)

var ActiveTick map[string]client.Task

func init() {
	ActiveTick = make(map[string]client.Task)
}

func CleanKapa() error {
	// Create a new Kapacitor client
	cli, err := client.New(client.Config{
		URL: kapacitorURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish kapacitor connexion: %v", err)
		return err
	}

	all_tasks, err := cli.ListTasks(&client.ListTasksOptions{})
	if err != nil {
		logger.Log.Errorf("Unable to List all current kapacitor tasks: %v", err)
		return err
	}
	for _, i := range all_tasks {
		err = cli.DeleteTask(i.Link)
		if err != nil {
			logger.Log.Errorf("Unable to delete the taskname for script %s: %v", i.ID, err)
			continue
		}
		logger.Log.Infof("Taskname for script %s has been successfully removed", i.ID)
	}
	return nil
}

func StartTick(t []string) error {
	i := len(ActiveTick)

	// Create a new Kapacitor client
	cli, err := client.New(client.Config{
		URL: kapacitorURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish kapacitor connexion: %v", err)
		return err
	}

	for _, v := range t {
		taskName := "jts_tick_" + strconv.Itoa(i)

		// Read the contents of the TICK script file
		tickScriptContent, err := os.ReadFile(v)
		if err != nil {
			logger.Log.Errorf("Unable to read tick file %s: %v", v, err)
			return err
		}

		// Create a new task using the TICK script content
		ticket := client.CreateTaskOptions{
			Type:       client.StreamTask,
			DBRPs:      []client.DBRP{{Database: "jtsdb", RetentionPolicy: "autogen"}},
			TICKscript: string(tickScriptContent),
			Status:     client.Enabled,
			ID:         taskName,
		}

		// Create the task in Kapacitor
		ActiveTick[v], err = cli.CreateTask(ticket)
		if err != nil {
			logger.Log.Errorf("Unable to create the tick task %s: %v", v, err)
			// Close the Kapacitor client
			return err
		}
		logger.Log.Infof("Tick Script %s has been successfully installed and enabled", v)
		i++
	}
	return nil
}

func DeleteTick(t []string) error {
	// Create a new Kapacitor client
	cli, err := client.New(client.Config{
		URL: kapacitorURL,
	})
	if err != nil {
		logger.Log.Errorf("Unable to establish kapacitor connexion: %v", err)
		return err
	}

	for _, v := range t {
		taskName, ok := ActiveTick[v]
		if ok {
			delete(ActiveTick, v)
			// Delete the task
			err = cli.DeleteTask(taskName.Link)
			if err != nil {
				logger.Log.Errorf("Unable to delete the taskname for script %s: %v", v, err)
				continue
			}
			logger.Log.Infof("Taskname for script %s has been successfully removed", v)
		} else {
			logger.Log.Errorf("Unable to find the taskname for script %s: %v", v, err)
			continue
		}
	}
	return nil
}
