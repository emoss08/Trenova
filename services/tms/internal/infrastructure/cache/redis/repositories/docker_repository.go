/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	dockerDiskUsageKey  = "docker:disk_usage"
	dockerSystemInfoKey = "docker:system_info"
	dockerNetworksKey   = "docker:networks"
	dockerVolumesKey    = "docker:volumes"
	dockerImagesKey     = "docker:images"
	dockerContainersKey = "docker:containers"
	dockerStatsKey      = "docker:stats"
	dockerSystemInfoTTL = 30 * time.Second
	dockerDiskUsageTTL  = 60 * time.Second
)

type DockerRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type dockerRepository struct {
	cache *redis.Client
	l     *zerolog.Logger
}

func NewDockerRepository(p DockerRepositoryParams) repositories.DockerCacheRepository {
	log := p.Logger.With().
		Str("repository", "docker").
		Logger()

	return &dockerRepository{
		cache: p.Cache,
		l:     &log,
	}
}

func (dr *dockerRepository) GetDiskUsage(ctx context.Context) (types.DiskUsage, error) {
	var du types.DiskUsage
	if err := dr.cache.GetJSON(ctx, "$", dockerDiskUsageKey, &du); err != nil {
		return types.DiskUsage{}, err
	}

	return du, nil
}

func (dr *dockerRepository) SetDiskUsage(ctx context.Context, du types.DiskUsage) error {
	return dr.cache.SetJSON(ctx, "$", dockerDiskUsageKey, du, dockerDiskUsageTTL)
}

func (dr *dockerRepository) InvalidateDiskUsage(ctx context.Context) error {
	return dr.cache.Del(ctx, dockerDiskUsageKey)
}

func (dr *dockerRepository) GetSystemInfo(ctx context.Context) (system.Info, error) {
	var info system.Info
	if err := dr.cache.GetJSON(ctx, "$", dockerSystemInfoKey, &info); err != nil {
		return system.Info{}, err
	}

	return info, nil
}

func (dr *dockerRepository) SetSystemInfo(ctx context.Context, info system.Info) error {
	return dr.cache.SetJSON(ctx, "$", dockerSystemInfoKey, info, dockerSystemInfoTTL)
}
