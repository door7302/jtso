package container

import (
	"context"
	"jtso/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func ListContainers() []types.Container {
	// Open Docker API
	var containers []types.Container
	containers = make([]types.Container, 0)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
	}
	defer cli.Close()

	containers, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logger.Log.Errorf("Unable to list container state: %v", err)
	}
	logger.Log.Info(" List of containers has been retrieved")
	return containers
}

func RestartContainer(name string) {
	timeout := 30

	// Open Docker API
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Log.Errorf("Unable to open Docker session: %v", err)
		return
	}
	defer cli.Close()

	// Restart container
	err = cli.ContainerRestart(context.Background(), name, container.StopOptions{Signal: "SIGTERM", Timeout: &timeout})
	if err != nil {
		logger.Log.Errorf("Unable to restart %s container: %v", name, err)
		return
	}
	logger.Log.Infof("%s container has been restarted", name)

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
	version := "N/A"
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
	vers, ok := imageInspect.Config.Labels["version"]
	if !ok {
		logger.Log.Errorf("Unable to retrieve Docker %s version", name)
		return "N/A"
		
	}

	logger.Log.Infof("%s container version is %s", name, version)	
	return version

}
