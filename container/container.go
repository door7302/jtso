package container

import (
	"context"
	"encoding/json"
	"jtso/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func calculateCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

func GetContainerStats() (map[string]map[string]float64, error) {
	statsMap := make(map[string]map[string]float64)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Error creating Docker client: %v", err)
		return nil, err
	}

	// Get the list of running containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logger.Log.Errorf("Error listing containers: %v", err)
		return nil, err
	}

	for _, container := range containers {
		// Retrieve stats for each container
		stats, err := cli.ContainerStats(context.Background(), container.ID, false)
		if err != nil {
			logger.Log.Errorf("Error getting stats for container %s: %v", container.ID, err)
			continue
		}
		defer stats.Body.Close()

		var stat types.StatsJSON
		if err := json.NewDecoder(stats.Body).Decode(&stat); err != nil {
			logger.Log.Errorf("Error decoding stats for container %s: %v", container.ID, err)
			continue
		}

		// Calculate CPU percentage
		cpuPercent := calculateCPUPercent(stat)

		// Calculate memory usage percentage
		memUsage := stat.MemoryStats.Usage
		memLimit := stat.MemoryStats.Limit
		memPercent := float64(memUsage) / float64(memLimit) * 100.0

		// Add stats to the map
		statsMap[container.Names[0][1:]] = map[string]float64{
			"cpu": cpuPercent,
			"mem": memPercent,
		}
	}

	return statsMap, nil
}

func ListContainers() []types.Container {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logger.Log.Errorf("Unable to list container state: %v", err)
	}
	logger.Log.Info("List of containers has been retrieved")
	return containers
}

func RestartContainer(name string) error {
	timeout := 30

	// Open Docker API
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
		return err
	}
	defer cli.Close()

	// Restart container
	err = cli.ContainerRestart(context.Background(), name, container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
	if err != nil {
		logger.Log.Errorf("Unable to restart %s container: %v", name, err)
		return err
	}
	logger.Log.Infof("%s container has been restarted", name)
	return nil

}

func StopContainer(name string) {
	timeout := 30

	// Open Docker API
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
		return
	}
	defer cli.Close()

	err = cli.ContainerStop(context.Background(), name, container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
	if err != nil {
		logger.Log.Errorf("Unable to stop %s container: %v", name, err)
		return
	}
	logger.Log.Infof("%s container has been stopped - no more router attached", name)

}

func GetVersionLabel(name string) string {

	// Open Docker API
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
		return "N/A"
	}
	defer cli.Close()

	// Get the image details using the Docker API
	imageInspect, _, err := cli.ImageInspectWithRaw(context.Background(), name)
	if err != nil {
		logger.Log.Errorf("Unable to retrieve Docker %s inspect data: %v", name, err)
		return "N/A"

	}

	// Extract the version label from imageInspect.Config.Labels
	version, ok := imageInspect.Config.Labels["version"]
	if !ok {
		logger.Log.Errorf("Unable to retrieve Docker %s version", name)
		return "N/A"

	}

	logger.Log.Debugf("%s container version is %s", name, version)
	return version

}
