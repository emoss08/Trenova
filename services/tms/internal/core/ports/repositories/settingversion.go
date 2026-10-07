package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetSettingVersionAtRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       settingversion.Kind
	SubjectID  pulid.ID
	Version    int64
}

type SettingVersionRepository interface {
	Create(ctx context.Context, version *settingversion.SettingVersion) error
	// LatestAt returns the newest save at or before a version, with its
	// author, or nil when none was kept.
	LatestAt(ctx context.Context, req *GetSettingVersionAtRequest) (*settingversion.SettingVersion, error)
}
