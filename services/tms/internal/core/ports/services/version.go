package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/system"
)

type VersionService interface {
	GetVersionInfo(ctx context.Context) (*system.VersionInfo, error)
	GetUpdateStatus(ctx context.Context) (*system.UpdateStatus, error)
	CheckForUpdates(ctx context.Context) (*system.UpdateStatus, error)
}
