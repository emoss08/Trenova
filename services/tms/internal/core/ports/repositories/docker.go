package repositories

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
)

type DockerCacheRepository interface {
	GetDiskUsage(ctx context.Context) (types.DiskUsage, error)
	SetDiskUsage(ctx context.Context, du types.DiskUsage) error
	InvalidateDiskUsage(ctx context.Context) error
	GetSystemInfo(ctx context.Context) (system.Info, error)
	SetSystemInfo(ctx context.Context, info system.Info) error
}
