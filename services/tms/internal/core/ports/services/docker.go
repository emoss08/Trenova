/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package services

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/emoss08/trenova/internal/infrastructure/external/dockerhub"
	"github.com/emoss08/trenova/shared/pulid"
)

// DockerOperationRequest contains common fields for Docker operations
type DockerOperationRequest struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

// DockerService defines the interface for Docker management operations
type DockerService interface {
	// Container operations
	ListContainers(ctx context.Context, req *DockerOperationRequest, all bool) ([]container.Summary, error)
	InspectContainer(ctx context.Context, req *DockerOperationRequest, containerID string) (container.InspectResponse, error)
	StartContainer(ctx context.Context, req *DockerOperationRequest, containerID string) error
	StopContainer(ctx context.Context, req *DockerOperationRequest, containerID string, timeout *int) error
	RestartContainer(ctx context.Context, req *DockerOperationRequest, containerID string, timeout *int) error
	RemoveContainer(ctx context.Context, req *DockerOperationRequest, containerID string, force bool) error
	GetContainerLogs(
		ctx context.Context,
		req *DockerOperationRequest,
		containerID string,
		tail string,
		follow bool,
	) (io.ReadCloser, error)
	GetContainerStats(
		ctx context.Context,
		req *DockerOperationRequest,
		containerID string,
	) (*dockerhub.ContainerStatsResponse, error)
	StreamContainerStats(
		ctx context.Context,
		req *DockerOperationRequest,
		containerID string,
		statsChan chan<- *dockerhub.ContainerStatsResponse,
	) error

	// Image operations
	ListImages(ctx context.Context, req *DockerOperationRequest) ([]image.Summary, error)
	PullImage(ctx context.Context, req *DockerOperationRequest, imageName string) (string, error)
	RemoveImage(ctx context.Context, req *DockerOperationRequest, imageID string, force bool) error

	// Volume operations
	ListVolumes(ctx context.Context, req *DockerOperationRequest) (*EnhancedVolumeListResponse, error)
	CreateVolume(
		ctx context.Context,
		req *DockerOperationRequest,
		name string,
		driver string,
		labels map[string]string,
	) (volume.Volume, error)
	RemoveVolume(ctx context.Context, req *DockerOperationRequest, volumeID string, force bool) error

	// Network operations
	ListNetworks(ctx context.Context, req *DockerOperationRequest) ([]network.Inspect, error)
	InspectNetwork(ctx context.Context, req *DockerOperationRequest, networkID string) (network.Inspect, error)

	// System operations
	GetSystemInfo(ctx context.Context, req *DockerOperationRequest) (system.Info, error)
	GetDiskUsage(ctx context.Context, req *DockerOperationRequest) (types.DiskUsage, error)
	PruneContainers(ctx context.Context, req *DockerOperationRequest) (container.PruneReport, error)
	PruneImages(ctx context.Context, req *DockerOperationRequest) (image.PruneReport, error)
	PruneVolumes(ctx context.Context, req *DockerOperationRequest) (volume.PruneReport, error)
	PruneSystem(ctx context.Context, req *DockerOperationRequest) (*SystemPruneReport, error)
}

// SystemPruneReport represents the combined results of system pruning
type SystemPruneReport struct {
	ContainersDeleted []string `json:"containersDeleted"`
	SpaceReclaimed    uint64   `json:"spaceReclaimed"`
	ImagesDeleted     []string `json:"imagesDeleted"`
	VolumesDeleted    []string `json:"volumesDeleted"`
}

// EnhancedVolume represents a Docker volume with additional size information
type EnhancedVolume struct {
	volume.Volume
	Size int64 `json:"size"`
}

// EnhancedVolumeListResponse represents a list of volumes with size information
type EnhancedVolumeListResponse struct {
	Volumes  []*EnhancedVolume `json:"volumes"`
	Warnings []string          `json:"warnings,omitempty"`
}
