package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
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

// ClearHomeLayoutOrgDefaultRequest drops the org-default flag from every preset
// except the one taking it over, which is how the single-default invariant is
// kept without a race between two administrators saving at once.
type ClearHomeLayoutOrgDefaultRequest struct {
	TenantInfo pagination.TenantInfo
	ExceptID   pulid.ID
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
	ClearOrgDefault(ctx context.Context, req *ClearHomeLayoutOrgDefaultRequest) error
	CountUsersForRoles(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		roleIDs []pulid.ID,
	) (int, error)
}
