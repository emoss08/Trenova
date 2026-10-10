package sidebarpreferenceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/sidebarpreference"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetSidebarPreferenceRequest struct {
	TenantInfo pagination.TenantInfo
}

type SidebarPreferenceRepository interface {
	Get(
		ctx context.Context,
		req *GetSidebarPreferenceRequest,
	) (*sidebarpreference.SidebarPreference, bool, error)
	Create(
		ctx context.Context,
		entity *sidebarpreference.SidebarPreference,
	) (*sidebarpreference.SidebarPreference, error)
	Update(
		ctx context.Context,
		entity *sidebarpreference.SidebarPreference,
	) (*sidebarpreference.SidebarPreference, error)
}
