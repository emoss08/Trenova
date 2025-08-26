/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/external/dockerhub"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	DockerClient        *dockerhub.Client
	Logger              *logger.Logger
	Cache               repositories.DockerCacheRepository
	PermissionService   services.PermissionService
	AuditService        services.AuditService
	NotificationService services.NotificationService
	Config              *config.Manager
}

type Service struct {
	client     *dockerhub.Client
	l          *zerolog.Logger
	ps         services.PermissionService
	cache      repositories.DockerCacheRepository
	as         services.AuditService
	ns         services.NotificationService
	config     *config.Manager
	errBuilder oops.OopsErrorBuilder
}

func NewService(p ServiceParams) services.DockerService {
	log := p.Logger.With().
		Str("service", "docker").
		Logger()

	return &Service{
		client:     p.DockerClient,
		l:          &log,
		cache:      p.Cache,
		ps:         p.PermissionService,
		as:         p.AuditService,
		ns:         p.NotificationService,
		config:     p.Config,
		errBuilder: oops.In("docker_service"),
	}
}

func (s *Service) ListContainers(
	ctx context.Context,
	req *services.DockerOperationRequest,
	all bool,
) ([]container.Summary, error) {
	log := s.l.With().
		Str("operation", "ListContainers").
		Bool("all", all).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	containers, err := s.client.ListContainers(ctx, all)
	if err != nil {
		log.Error().Err(err).Msg("failed to list containers")
		return nil, s.errBuilder.
			With("operation", "list_containers").
			With("all", all).
			Wrap(err)
	}

	log.Debug().Int("count", len(containers)).Msg("containers listed successfully")
	return containers, nil
}

func (s *Service) InspectContainer(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
) (container.InspectResponse, error) {
	log := s.l.With().
		Str("operation", "InspectContainer").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return container.InspectResponse{}, err
	}

	resp, err := s.client.InspectContainer(ctx, containerID)
	if err != nil {
		log.Error().Err(err).Msg("failed to inspect container")
		return container.InspectResponse{}, s.errBuilder.
			With("operation", "inspect_container").
			With("container_id", containerID).
			Wrap(err)
	}

	log.Debug().Msg("container inspected successfully")
	return resp, nil
}

func (s *Service) StartContainer(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
) error {
	log := s.l.With().
		Str("operation", "StartContainer").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionManage,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.StartContainer(ctx, containerID); err != nil {
		log.Error().Err(err).Msg("failed to start container")
		return s.errBuilder.
			With("operation", "start_container").
			With("container_id", containerID).
			Wrap(err)
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"container_started", containerID, map[string]any{"action": "start"}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("container started successfully")
	return nil
}

func (s *Service) StopContainer(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
	timeout *int,
) error {
	log := s.l.With().
		Str("operation", "StopContainer").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionManage,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.StopContainer(ctx, containerID, timeout); err != nil {
		log.Error().Err(err).Msg("failed to stop container")
		return s.errBuilder.
			With("operation", "stop_container").
			With("container_id", containerID).
			Wrap(err)
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"container_stopped", containerID, map[string]any{"action": "stop"}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("container stopped successfully")
	return nil
}

func (s *Service) RestartContainer(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
	timeout *int,
) error {
	log := s.l.With().
		Str("operation", "RestartContainer").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionManage,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.RestartContainer(ctx, containerID, timeout); err != nil {
		log.Error().Err(err).Msg("failed to restart container")
		return s.errBuilder.
			With("operation", "restart_container").
			With("container_id", containerID).
			Wrap(err)
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"container_restarted", containerID, map[string]any{"action": "restart"}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("container restarted successfully")
	return nil
}

func (s *Service) RemoveContainer(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
	force bool,
) error {
	log := s.l.With().
		Str("operation", "RemoveContainer").
		Str("container_id", containerID).
		Bool("force", force).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.RemoveContainer(ctx, containerID, force); err != nil {
		log.Error().Err(err).Msg("failed to remove container")
		return s.errBuilder.
			With("operation", "remove_container").
			With("container_id", containerID).
			With("force", force).
			Wrap(err)
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"container_removed", containerID, map[string]any{"action": "remove", "force": force}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("container removed successfully")
	return nil
}

func (s *Service) GetContainerLogs(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
	tail string,
	follow bool,
) (io.ReadCloser, error) {
	log := s.l.With().
		Str("operation", "GetContainerLogs").
		Str("container_id", containerID).
		Str("tail", tail).
		Bool("follow", follow).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionAudit,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	logs, err := s.client.GetContainerLogs(ctx, containerID, tail, follow)
	if err != nil {
		log.Error().Err(err).Msg("failed to get container logs")
		return nil, s.errBuilder.
			With("operation", "get_container_logs").
			With("container_id", containerID).
			Wrap(err)
	}

	log.Debug().Msg("container logs retrieved successfully")
	return logs, nil
}

func (s *Service) GetContainerStats(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
) (*dockerhub.ContainerStatsResponse, error) {
	log := s.l.With().
		Str("operation", "GetContainerStats").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	stats, err := s.client.GetContainerStats(ctx, containerID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get container stats")
		return nil, s.errBuilder.
			With("operation", "get_container_stats").
			With("container_id", containerID).
			Wrap(err)
	}

	parsedStats, err := s.client.ParseContainerStats(containerID, stats)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse container stats")
		return nil, s.errBuilder.
			With("operation", "parse_container_stats").
			With("container_id", containerID).
			Wrap(err)
	}

	return parsedStats, nil
}

func (s *Service) StreamContainerStats(
	ctx context.Context,
	req *services.DockerOperationRequest,
	containerID string,
	statsChan chan<- *dockerhub.ContainerStatsResponse,
) error {
	log := s.l.With().
		Str("operation", "StreamContainerStats").
		Str("container_id", containerID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	interval := 2 * time.Second
	if s.config != nil && s.config.Docker() != nil && s.config.Docker().StatsInterval > 0 {
		interval = s.config.Docker().StatsInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Debug().Dur("interval", interval).Msg("starting container stats stream")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("container stats stream stopped by context")
			return ctx.Err()
		case <-ticker.C:
			stats, err := s.GetContainerStats(ctx, req, containerID)
			if err != nil {
				log.Error().Err(err).Msg("failed to get container stats during stream")
				continue
			}

			select {
			case statsChan <- stats:
			case <-ctx.Done():
				log.Debug().Msg("container stats stream stopped by context")
				return ctx.Err()
			}
		}
	}
}

func (s *Service) ListImages(
	ctx context.Context,
	req *services.DockerOperationRequest,
) ([]image.Summary, error) {
	log := s.l.With().
		Str("operation", "ListImages").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	images, err := s.client.ListImages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list images")
		return nil, s.errBuilder.
			With("operation", "list_images").
			Wrap(err)
	}

	return images, nil
}

func (s *Service) PullImage(
	ctx context.Context,
	req *services.DockerOperationRequest,
	imageName string,
) (string, error) {
	log := s.l.With().
		Str("operation", "PullImage").
		Str("image", imageName).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionManage,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return "", err
	}

	reader, err := s.client.PullImage(ctx, imageName)
	if err != nil {
		log.Error().Err(err).Msg("failed to pull image")
		return "", s.errBuilder.
			With("operation", "pull_image").
			With("image", imageName).
			Wrap(err)
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var event map[string]any
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return "", s.errBuilder.
				With("operation", "pull_image").
				With("image", imageName).
				Wrap(err)
		}
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"image_pulled", imageName, map[string]any{"image": imageName}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return fmt.Sprintf("Image %s pulled successfully", imageName), nil
}

func (s *Service) RemoveImage(
	ctx context.Context,
	req *services.DockerOperationRequest,
	imageID string,
	force bool,
) error {
	log := s.l.With().
		Str("operation", "RemoveImage").
		Str("image_id", imageID).
		Bool("force", force).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"image_removed", imageID, map[string]any{"image_id": imageID, "force": force}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return nil
}

func (s *Service) ListVolumes(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (*services.VolumeListResponse, error) {
	log := s.l.With().
		Str("operation", "ListVolumes").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	volumes, err := s.client.ListVolumes(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list volumes")
		return nil, s.errBuilder.
			With("operation", "list_volumes").
			Wrap(err)
	}

	var diskUsage types.DiskUsage
	diskUsage, err = s.cache.GetDiskUsage(ctx)
	if err != nil {
		diskUsage, err = s.client.GetDiskUsage(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to get disk usage for volume sizes")
		}
	}

	volumeSizes := make(map[string]int64)
	for _, v := range diskUsage.Volumes {
		if v.Name != "" {
			volumeSizes[v.Name] = v.UsageData.Size
		}
	}

	enhancedVolumes := make([]*services.Volume, 0, len(volumes.Volumes))
	for _, v := range volumes.Volumes {
		enhancedVol := &services.Volume{
			Volume: *v,
			Size:   volumeSizes[v.Name],
		}
		enhancedVolumes = append(enhancedVolumes, enhancedVol)
	}

	return &services.VolumeListResponse{
		Volumes:  enhancedVolumes,
		Warnings: volumes.Warnings,
	}, nil
}

func (s *Service) CreateVolume(
	ctx context.Context,
	req *services.DockerOperationRequest,
	name string,
	driver string,
	labels map[string]string,
) (volume.Volume, error) {
	log := s.l.With().
		Str("operation", "CreateVolume").
		Str("name", name).
		Str("driver", driver).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionManage,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return volume.Volume{}, err
	}

	vol, err := s.client.CreateVolume(ctx, name, driver, labels)
	if err != nil {
		log.Error().Err(err).Msg("failed to create volume")
		return volume.Volume{}, s.errBuilder.
			With("operation", "create_volume").
			With("name", name).
			With("driver", driver).
			Wrap(err)
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"volume_created", name, map[string]any{"name": name, "driver": driver}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("volume created successfully")
	return vol, nil
}

func (s *Service) RemoveVolume(
	ctx context.Context,
	req *services.DockerOperationRequest,
	volumeID string,
	force bool,
) error {
	log := s.l.With().
		Str("operation", "RemoveVolume").
		Str("volume_id", volumeID).
		Bool("force", force).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.RemoveVolume(ctx, volumeID, force); err != nil {
		log.Error().Err(err).Msg("failed to remove volume")
		return s.errBuilder.
			With("operation", "remove_volume").
			With("volume_id", volumeID).
			With("force", force).
			Wrap(err)
	}

	// Invalidate disk usage cache
	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	// Audit log
	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"volume_removed", volumeID, map[string]any{"volume_id": volumeID, "force": force}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	log.Info().Msg("volume removed successfully")
	return nil
}

func (s *Service) ListNetworks(
	ctx context.Context,
	req *services.DockerOperationRequest,
) ([]network.Inspect, error) {
	log := s.l.With().
		Str("operation", "ListNetworks").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	networkList, err := s.client.ListNetworks(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list networks")
		return nil, s.errBuilder.
			With("operation", "list_networks").
			Wrap(err)
	}

	networks := make([]network.Inspect, 0, len(networkList))
	for _, net := range networkList {
		netDetail, err := s.client.InspectNetwork(ctx, net.ID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("network_id", net.ID).
				Str("network_name", net.Name).
				Msg("failed to inspect network, skipping")
			continue
		}
		networks = append(networks, netDetail)
	}

	return networks, nil
}

func (s *Service) InspectNetwork(
	ctx context.Context,
	req *services.DockerOperationRequest,
	networkID string,
) (network.Inspect, error) {
	log := s.l.With().
		Str("operation", "InspectNetwork").
		Str("network_id", networkID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return network.Inspect{}, err
	}

	net, err := s.client.InspectNetwork(ctx, networkID)
	if err != nil {
		log.Error().Err(err).Msg("failed to inspect network")
		return network.Inspect{}, s.errBuilder.
			With("operation", "inspect_network").
			With("network_id", networkID).
			Wrap(err)
	}

	return net, nil
}

func (s *Service) RemoveNetwork(
	ctx context.Context,
	req *services.DockerOperationRequest,
	networkID string,
) error {
	log := s.l.With().
		Str("operation", "RemoveNetwork").
		Str("network_id", networkID).
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return err
	}

	if err := s.client.RemoveNetwork(ctx, networkID); err != nil {
		log.Error().Err(err).Msg("failed to remove network")
		return s.errBuilder.
			With("operation", "remove_network").
			With("network_id", networkID).
			Wrap(err)
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"network_removed", networkID, map[string]any{"network_id": networkID}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return nil
}

func (s *Service) GetSystemInfo(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (system.Info, error) {
	log := s.l.With().
		Str("operation", "GetSystemInfo").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return system.Info{}, err
	}

	info, err := s.cache.GetSystemInfo(ctx)
	if err != nil {
		info, err = s.client.GetSystemInfo(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to get system info")
			return system.Info{}, s.errBuilder.
				With("operation", "get_system_info").
				Wrap(err)
		}

		if err := s.cache.SetSystemInfo(ctx, info); err != nil {
			log.Warn().Err(err).Msg("failed to cache system info")
		}
	}

	return info, nil
}

func (s *Service) GetDiskUsage(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (du types.DiskUsage, err error) {
	log := s.l.With().
		Str("operation", "GetDiskUsage").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionRead,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return types.DiskUsage{}, err
	}

	du, err = s.cache.GetDiskUsage(ctx)
	if err != nil {
		log.Debug().Msg("disk usage not found in cache, fetching from client")
		du, err = s.client.GetDiskUsage(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to get disk usage")
			return types.DiskUsage{}, s.errBuilder.
				With("operation", "get_disk_usage").
				Wrap(err)
		}

		if err := s.cache.SetDiskUsage(ctx, du); err != nil {
			log.Warn().Err(err).Msg("failed to cache disk usage")
		}
	}

	return du, nil
}

func (s *Service) PruneContainers(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (container.PruneReport, error) {
	log := s.l.With().
		Str("operation", "PruneContainers").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return container.PruneReport{}, err
	}

	report, err := s.client.PruneContainers(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune containers")
		return container.PruneReport{}, s.errBuilder.
			With("operation", "prune_containers").
			Wrap(err)
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"containers_pruned", "", map[string]any{"containers_deleted": len(report.ContainersDeleted), "space_reclaimed": report.SpaceReclaimed}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return report, nil
}

func (s *Service) PruneImages(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (image.PruneReport, error) {
	log := s.l.With().
		Str("operation", "PruneImages").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return image.PruneReport{}, err
	}

	report, err := s.client.PruneImages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune images")
		return image.PruneReport{}, s.errBuilder.
			With("operation", "prune_images").
			Wrap(err)
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"images_pruned", "", map[string]any{"images_deleted": len(report.ImagesDeleted), "space_reclaimed": report.SpaceReclaimed}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return report, nil
}

func (s *Service) PruneVolumes(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (volume.PruneReport, error) {
	log := s.l.With().
		Str("operation", "PruneVolumes").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return volume.PruneReport{}, err
	}

	report, err := s.client.PruneVolumes(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune volumes")
		return volume.PruneReport{}, s.errBuilder.
			With("operation", "prune_volumes").
			Wrap(err)
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"volumes_pruned", "", map[string]any{"volumes_deleted": len(report.VolumesDeleted), "space_reclaimed": report.SpaceReclaimed}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return report, nil
}

func (s *Service) PruneSystem(
	ctx context.Context,
	req *services.DockerOperationRequest,
) (*services.SystemPruneReport, error) {
	log := s.l.With().
		Str("operation", "PruneSystem").
		Logger()

	if err := s.checkDockerPermission(ctx, checkDockerPermissionParams{
		userID: req.UserID,
		orgID:  req.OrganizationID,
		buID:   req.BusinessUnitID,
		action: permission.ActionDelete,
	}); err != nil {
		log.Error().Err(err).Msg("permission check failed")
		return nil, err
	}

	var totalSpace uint64
	var containersDeleted []string
	var imagesDeleted []string
	var volumesDeleted []string

	containerReport, err := s.client.PruneContainers(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune containers during system prune")
	} else {
		totalSpace += containerReport.SpaceReclaimed
		containersDeleted = containerReport.ContainersDeleted
	}

	imageReport, err := s.client.PruneImages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune images during system prune")
	} else {
		totalSpace += imageReport.SpaceReclaimed
		for _, img := range imageReport.ImagesDeleted {
			if img.Deleted != "" {
				imagesDeleted = append(imagesDeleted, img.Deleted)
			}
		}
	}

	volumeReport, err := s.client.PruneVolumes(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to prune volumes during system prune")
	} else {
		totalSpace += volumeReport.SpaceReclaimed
		for _, vol := range volumeReport.VolumesDeleted {
			volumesDeleted = append(volumesDeleted, vol)
		}
	}

	if err := s.cache.InvalidateDiskUsage(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate disk usage cache")
	}

	if err := s.auditDockerOperation(req.UserID, req.OrganizationID, req.BusinessUnitID,
		"system_pruned", "", map[string]any{
			"containers_deleted": len(containersDeleted),
			"images_deleted":     len(imagesDeleted),
			"volumes_deleted":    len(volumesDeleted),
			"space_reclaimed":    totalSpace,
		}); err != nil {
		log.Warn().Err(err).Msg("failed to create audit log")
	}

	return &services.SystemPruneReport{
		ContainersDeleted: containersDeleted,
		SpaceReclaimed:    totalSpace,
		ImagesDeleted:     imagesDeleted,
		VolumesDeleted:    volumesDeleted,
	}, nil
}

type checkDockerPermissionParams struct {
	userID pulid.ID
	orgID  pulid.ID
	buID   pulid.ID
	action permission.Action
}

func (s *Service) checkDockerPermission(
	ctx context.Context,
	params checkDockerPermissionParams,
) error {
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         params.userID,
				Resource:       permission.ResourceDocker,
				Action:         params.action,
				BusinessUnitID: params.buID,
				OrganizationID: params.orgID,
			},
		},
	)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			fmt.Sprintf("You do not have permission to %s Docker resources", params.action),
		)
	}

	return nil
}

func (s *Service) auditDockerOperation(
	userID pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
	operation string,
	resourceID string,
	metadata map[string]any,
) error {
	var action permission.Action
	switch operation {
	case "container_started", "container_stopped", "container_restarted":
		action = permission.ActionManage
	case "container_removed", "image_removed", "volume_removed", "network_removed":
		action = permission.ActionDelete
	case "image_pulled", "volume_created":
		action = permission.ActionManage
	case "containers_pruned", "images_pruned", "volumes_pruned", "system_pruned":
		action = permission.ActionDelete
	default:
		action = permission.ActionManage
	}

	var dockerResourceID pulid.ID
	if resourceID != "" && len(resourceID) >= 8 {
		dockerResourceID = pulid.MustNew(fmt.Sprintf("dock_%s", resourceID[:8]))
	} else if resourceID != "" {
		dockerResourceID = pulid.MustNew(fmt.Sprintf("dock_%s", resourceID))
	} else {
		dockerResourceID = pulid.MustNew("dock")
	}

	err := s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocker,
			ResourceID:     dockerResourceID.String(),
			Action:         action,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			CurrentState:   jsonutils.MustToJSON(metadata),
		},
		audit.WithComment(fmt.Sprintf("Docker operation: %s", operation)),
	)
	if err != nil {
		s.l.Warn().Err(err).Str("operation", operation).Msg("failed to create audit log")
	}

	return nil
}
