package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/googlemapsconfig"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetGoogleMapsConfigRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type GoogleMapsConfigRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*googlemapsconfig.GoogleMapsConfig, error)
	GetAPIKeyByOrgID(ctx context.Context, orgID pulid.ID) (string, error)
	Update(ctx context.Context, gmc *googlemapsconfig.GoogleMapsConfig) (*googlemapsconfig.GoogleMapsConfig, error)
}
