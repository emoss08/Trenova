package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetHomePreferenceRequest struct {
	TenantInfo pagination.TenantInfo
}

type GetHomeLayoutPresetRequest struct {
	TenantInfo pagination.TenantInfo
	PresetID   pulid.ID
}

type ListHomeLayoutPresetsRequest struct {
	TenantInfo pagination.TenantInfo
}

type DeleteHomeLayoutPresetRequest struct {
	TenantInfo pagination.TenantInfo
	PresetID   pulid.ID
}

// HomeRoleUserAssignment is one live role membership together with the role's
// core responsibility, which is everything preset audience math needs: a preset
// reaches a user through an explicit role ID or through the responsibility the
// role carries.
type HomeRoleUserAssignment struct {
	RoleID             pulid.ID
	UserID             pulid.ID
	CoreResponsibility permission.CoreResponsibility
}

type HomeLayoutRepository interface {
	GetPreference(
		ctx context.Context,
		req *GetHomePreferenceRequest,
	) (*homelayout.Preference, bool, error)
	CreatePreference(
		ctx context.Context,
		entity *homelayout.Preference,
	) (*homelayout.Preference, error)
	UpdatePreference(
		ctx context.Context,
		entity *homelayout.Preference,
	) (*homelayout.Preference, error)

	ListPresets(
		ctx context.Context,
		req *ListHomeLayoutPresetsRequest,
	) ([]*homelayout.Preset, error)
	GetPreset(
		ctx context.Context,
		req *GetHomeLayoutPresetRequest,
	) (*homelayout.Preset, error)
	CreatePreset(ctx context.Context, entity *homelayout.Preset) (*homelayout.Preset, error)
	UpdatePreset(ctx context.Context, entity *homelayout.Preset) (*homelayout.Preset, error)
	DeletePreset(ctx context.Context, req *DeleteHomeLayoutPresetRequest) error
	ListRoleUserAssignments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]HomeRoleUserAssignment, error)
}
