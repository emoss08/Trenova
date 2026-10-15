package iamports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AccessSettingsRepository interface {
	Get(ctx context.Context, tenant pagination.TenantInfo) (*iam.AccessSettings, error)
	Save(ctx context.Context, settings *iam.AccessSettings) (*iam.AccessSettings, error)
}

type UpdateAccessSettingsRequest struct {
	TenantInfo            pagination.TenantInfo
	Actor                 *permissionports.RequestActor
	RequireRoleActivation bool
	Version               int64
}

type AccessSettingsService interface {
	Get(ctx context.Context, tenant pagination.TenantInfo) (*iam.AccessSettings, error)
	Update(ctx context.Context, req *UpdateAccessSettingsRequest) (*iam.AccessSettings, error)
}

type RoleActivationPolicy interface {
	RoleActivationRequired(ctx context.Context, tenant pagination.TenantInfo) (bool, error)
}
