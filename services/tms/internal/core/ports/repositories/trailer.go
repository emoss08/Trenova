package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrailerRelationIncludes struct {
	IncludeTenantDetails         bool     `json:"includeTenantDetails"`
	IncludeEquipmentDetails      bool     `json:"includeEquipmentDetails"`
	IncludeFleetDetails          bool     `json:"includeFleetDetails"`
	IncludeFleetManager          bool     `json:"includeFleetManager"`
	IncludeRegistrationDetails   bool     `json:"includeRegistrationDetails"`
	IncludeBusinessUnit          bool     `json:"includeBusinessUnit"`
	IncludeOrganization          bool     `json:"includeOrganization"`
	IncludeEquipmentType         bool     `json:"includeEquipmentType"`
	IncludeEquipmentManufacturer bool     `json:"includeEquipmentManufacturer"`
	IncludeFleetCode             bool     `json:"includeFleetCode"`
	IncludeRegistrationState     bool     `json:"includeRegistrationState"`
	IncludeLastKnownLocation     bool     `json:"includeLastKnownLocation"`
	IncludeCustomFields          bool     `json:"includeCustomFields"`
	TrailerColumns               []string `json:"-"`
	EquipmentTypeColumns         []string `json:"-"`
	EquipmentManufacturerColumns []string `json:"-"`
	FleetCodeColumns             []string `json:"-"`
}

func FullTrailerRelationIncludes() TrailerRelationIncludes {
	return TrailerRelationIncludes{
		IncludeTenantDetails:         true,
		IncludeEquipmentDetails:      true,
		IncludeFleetDetails:          true,
		IncludeFleetManager:          true,
		IncludeRegistrationDetails:   true,
		IncludeBusinessUnit:          true,
		IncludeOrganization:          true,
		IncludeEquipmentType:         true,
		IncludeEquipmentManufacturer: true,
		IncludeFleetCode:             true,
		IncludeRegistrationState:     true,
		IncludeLastKnownLocation:     true,
		IncludeCustomFields:          true,
	}
}

type ListTrailersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
	TrailerRelationIncludes
	Status string `json:"status"`
}

type GetTrailerByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	TrailerRelationIncludes
}

type BulkUpdateTrailerStatusRequest struct {
	TenantInfo pagination.TenantInfo       `json:"-"`
	TrailerIDs []pulid.ID                  `json:"trailerIds"`
	Status     domaintypes.EquipmentStatus `json:"status"`
}

type GetTrailersByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TrailerIDs []pulid.ID            `json:"trailerIds"`
	TrailerRelationIncludes
}

type LocateTrailerRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	TrailerID     pulid.ID              `json:"trailerId"`
	NewLocationID pulid.ID              `json:"newLocationId"`
}

func (r *LocateTrailerRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()

	if r == nil {
		me.Add("", errortypes.ErrInvalid, "Request is required")
		return me
	}

	if r.TenantInfo.OrgID.IsNil() {
		me.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if r.TenantInfo.BuID.IsNil() {
		me.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if r.TrailerID.IsNil() {
		me.Add("trailerId", errortypes.ErrRequired, "Trailer ID is required")
	}
	if r.NewLocationID.IsNil() {
		me.Add("newLocationId", errortypes.ErrRequired, "New location ID is required")
	}
	if me.HasErrors() {
		return me
	}

	return nil
}

type TrailerRepository interface {
	List(
		ctx context.Context,
		req *ListTrailersRequest,
	) (*pagination.CursorListResult[*trailer.Trailer], error)
	GetByID(
		ctx context.Context,
		req GetTrailerByIDRequest,
	) (*trailer.Trailer, error)
	Create(
		ctx context.Context,
		entity *trailer.Trailer,
	) (*trailer.Trailer, error)
	Update(
		ctx context.Context,
		entity *trailer.Trailer,
	) (*trailer.Trailer, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateTrailerStatusRequest,
	) ([]*trailer.Trailer, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*trailer.Trailer], error)
	GetByIDs(
		ctx context.Context,
		req GetTrailersByIDsRequest,
	) ([]*trailer.Trailer, error)
}
