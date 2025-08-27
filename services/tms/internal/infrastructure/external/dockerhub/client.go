package dockerhub

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In
}

type Client struct {
	cli *client.Client
}

func NewClient(p ClientParams) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, oops.In("docker_hub_client").
			Time(time.Now()).
			Wrap(err)
	}

	return &Client{
		cli: cli,
	}, nil
}

// Container Operations

// ListContainers returns a list of containers
func (c *Client) ListContainers(ctx context.Context, all bool) ([]types.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, oops.In("docker_list_containers").
			Time(time.Now()).
			Wrap(err)
	}
	return containers, nil
}

// InspectContainer returns detailed information about a container
func (c *Client) InspectContainer(
	ctx context.Context,
	containerID string,
) (types.ContainerJSON, error) {
	container, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, oops.In("docker_inspect_container").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return container, nil
}

// StartContainer starts a stopped container
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return oops.In("docker_start_container").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return nil
}

// StopContainer stops a running container
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout *int) error {
	stopOptions := container.StopOptions{}
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := c.cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return oops.In("docker_stop_container").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return nil
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, containerID string, timeout *int) error {
	stopOptions := container.StopOptions{}
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := c.cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return oops.In("docker_restart_container").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}); err != nil {
		return oops.In("docker_remove_container").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return nil
}

// GetContainerLogs retrieves logs from a container
func (c *Client) GetContainerLogs(
	ctx context.Context,
	containerID string,
	tail string,
	follow bool,
) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	}

	logs, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, oops.In("docker_container_logs").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return logs, nil
}

// GetContainerStats retrieves real-time stats for a container
func (c *Client) GetContainerStats(
	ctx context.Context,
	containerID string,
) (container.StatsResponseReader, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return container.StatsResponseReader{}, oops.In("docker_container_stats").
			Time(time.Now()).
			With("container_id", containerID).
			Wrap(err)
	}
	return stats, nil
}

// Image Operations

// ListImages returns a list of Docker images
func (c *Client) ListImages(ctx context.Context) ([]image.Summary, error) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, oops.In("docker_list_images").
			Time(time.Now()).
			Wrap(err)
	}
	return images, nil
}

// PullImage pulls a Docker image from a registry
func (c *Client) PullImage(ctx context.Context, imageName string) (io.ReadCloser, error) {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return nil, oops.In("docker_pull_image").
			Time(time.Now()).
			With("image", imageName).
			Wrap(err)
	}
	return reader, nil
}

// RemoveImage removes a Docker image
func (c *Client) RemoveImage(
	ctx context.Context,
	imageID string,
	force bool,
) ([]image.DeleteResponse, error) {
	removed, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	if err != nil {
		return nil, oops.In("docker_remove_image").
			Time(time.Now()).
			With("image_id", imageID).
			Wrap(err)
	}
	return removed, nil
}

// Volume Operations

// ListVolumes returns a list of Docker volumes
func (c *Client) ListVolumes(ctx context.Context) (volume.ListResponse, error) {
	volumes, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return volume.ListResponse{}, oops.In("docker_list_volumes").
			Time(time.Now()).
			Wrap(err)
	}
	return volumes, nil
}

// CreateVolume creates a new Docker volume
func (c *Client) CreateVolume(
	ctx context.Context,
	name string,
	driver string,
	labels map[string]string,
) (volume.Volume, error) {
	vol, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: driver,
		Labels: labels,
	})
	if err != nil {
		return volume.Volume{}, oops.In("docker_create_volume").
			Time(time.Now()).
			With("name", name).
			Wrap(err)
	}
	return vol, nil
}

// RemoveVolume removes a Docker volume
func (c *Client) RemoveVolume(ctx context.Context, volumeID string, force bool) error {
	if err := c.cli.VolumeRemove(ctx, volumeID, force); err != nil {
		return oops.In("docker_remove_volume").
			Time(time.Now()).
			With("volume_id", volumeID).
			Wrap(err)
	}
	return nil
}

// ListNetworks returns a list of Docker networks
func (c *Client) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, oops.In("docker_list_networks").
			Time(time.Now()).
			Wrap(err)
	}
	return networks, nil
}

// InspectNetwork returns detailed information about a network
func (c *Client) InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	net, err := c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{
		Verbose: true,
	})
	if err != nil {
		return network.Inspect{}, oops.In("docker_inspect_network").
			Time(time.Now()).
			With("network_id", networkID).
			Wrap(err)
	}
	return net, nil
}

// RemoveNetwork removes a Docker network
func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	if err := c.cli.NetworkRemove(ctx, networkID); err != nil {
		return oops.In("docker_remove_network").
			Time(time.Now()).
			With("network_id", networkID).
			Wrap(err)
	}

	return nil
}

// GetSystemInfo returns Docker system information
func (c *Client) GetSystemInfo(ctx context.Context) (system.Info, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return system.Info{}, oops.In("docker_system_info").
			Time(time.Now()).
			Wrap(err)
	}
	return info, nil
}

// GetDiskUsage returns Docker disk usage information
func (c *Client) GetDiskUsage(ctx context.Context) (types.DiskUsage, error) {
	du, err := c.cli.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return types.DiskUsage{}, oops.In("docker_disk_usage").
			Time(time.Now()).
			Wrap(err)
	}
	return du, nil
}

// PruneContainers removes stopped containers
func (c *Client) PruneContainers(ctx context.Context) (container.PruneReport, error) {
	report, err := c.cli.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		return container.PruneReport{}, oops.In("docker_prune_containers").
			Time(time.Now()).
			Wrap(err)
	}
	return report, nil
}

// PruneImages removes unused images
func (c *Client) PruneImages(ctx context.Context) (image.PruneReport, error) {
	report, err := c.cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return image.PruneReport{}, oops.In("docker_prune_images").
			Time(time.Now()).
			Wrap(err)
	}
	return report, nil
}

// PruneVolumes removes unused volumes
func (c *Client) PruneVolumes(ctx context.Context) (volume.PruneReport, error) {
	report, err := c.cli.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return volume.PruneReport{}, oops.In("docker_prune_volumes").
			Time(time.Now()).
			Wrap(err)
	}
	return report, nil
}

// ContainerStatsResponse represents formatted container stats
type ContainerStatsResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CPUPercent  float64   `json:"cpuPercent"`
	MemUsage    uint64    `json:"memUsage"`
	MemLimit    uint64    `json:"memLimit"`
	MemPercent  float64   `json:"memPercent"`
	NetInput    uint64    `json:"netInput"`
	NetOutput   uint64    `json:"netOutput"`
	BlockInput  uint64    `json:"blockInput"`
	BlockOutput uint64    `json:"blockOutput"`
	PidsCurrent uint64    `json:"pidsCurrent"`
	Timestamp   time.Time `json:"timestamp"`
}

// ParseContainerStats parses raw container stats into a formatted response
func (c *Client) ParseContainerStats(
	containerID string,
	stats container.StatsResponseReader,
) (*ContainerStatsResponse, error) {
	var v *container.StatsResponse
	dec := json.NewDecoder(stats.Body)
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	// Calculate CPU percentage
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(
			len(v.CPUStats.CPUUsage.PercpuUsage),
		) * 100.0
	}

	// Calculate memory percentage
	memPercent := 0.0
	if v.MemoryStats.Limit > 0 {
		memPercent = (float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit)) * 100.0
	}

	// Calculate network I/O
	var netInput, netOutput uint64
	for _, net := range v.Networks {
		netInput += net.RxBytes
		netOutput += net.TxBytes
	}

	// Calculate block I/O
	var blockInput, blockOutput uint64
	for _, bioEntry := range v.BlkioStats.IoServiceBytesRecursive {
		switch bioEntry.Op {
		case "read":
			blockInput += bioEntry.Value
		case "write":
			blockOutput += bioEntry.Value
		}
	}

	return &ContainerStatsResponse{
		ID:          containerID,
		Name:        v.Name,
		CPUPercent:  cpuPercent,
		MemUsage:    v.MemoryStats.Usage,
		MemLimit:    v.MemoryStats.Limit,
		MemPercent:  memPercent,
		NetInput:    netInput,
		NetOutput:   netOutput,
		BlockInput:  blockInput,
		BlockOutput: blockOutput,
		PidsCurrent: v.PidsStats.Current,
		Timestamp:   time.Now(),
	}, nil
}
