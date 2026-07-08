package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TractorRelationIncludes struct {
	IncludeTenantDetails          bool     `json:"includeTenantDetails"`
	IncludeEquipmentDetails       bool     `json:"includeEquipmentDetails"`
	IncludeFleetDetails           bool     `json:"includeFleetDetails"`
	IncludeWorkerDetails          bool     `json:"includeWorkerDetails"`
	IncludeBusinessUnit           bool     `json:"includeBusinessUnit"`
	IncludeOrganization           bool     `json:"includeOrganization"`
	IncludeEquipmentType          bool     `json:"includeEquipmentType"`
	IncludeEquipmentManufacturer  bool     `json:"includeEquipmentManufacturer"`
	IncludeFleetCode              bool     `json:"includeFleetCode"`
	IncludeState                  bool     `json:"includeState"`
	IncludePrimaryWorker          bool     `json:"includePrimaryWorker"`
	IncludePrimaryWorkerState     bool     `json:"includePrimaryWorkerState"`
	IncludePrimaryWorkerFleet     bool     `json:"includePrimaryWorkerFleet"`
	IncludePrimaryWorkerManager   bool     `json:"includePrimaryWorkerManager"`
	IncludeSecondaryWorker        bool     `json:"includeSecondaryWorker"`
	IncludeSecondaryWorkerState   bool     `json:"includeSecondaryWorkerState"`
	IncludeSecondaryWorkerFleet   bool     `json:"includeSecondaryWorkerFleet"`
	IncludeSecondaryWorkerManager bool     `json:"includeSecondaryWorkerManager"`
	IncludeLastKnownLocation      bool     `json:"includeLastKnownLocation"`
	IncludeCustomFields           bool     `json:"includeCustomFields"`
	TractorColumns                []string `json:"-"`
	EquipmentTypeColumns          []string `json:"-"`
	EquipmentManufacturerColumns  []string `json:"-"`
	FleetCodeColumns              []string `json:"-"`
	PrimaryWorkerColumns          []string `json:"-"`
	SecondaryWorkerColumns        []string `json:"-"`
}

func FullTractorRelationIncludes() TractorRelationIncludes {
	return TractorRelationIncludes{
		IncludeTenantDetails:          true,
		IncludeEquipmentDetails:       true,
		IncludeFleetDetails:           true,
		IncludeWorkerDetails:          true,
		IncludeBusinessUnit:           true,
		IncludeOrganization:           true,
		IncludeEquipmentType:          true,
		IncludeEquipmentManufacturer:  true,
		IncludeFleetCode:              true,
		IncludeState:                  true,
		IncludePrimaryWorker:          true,
		IncludePrimaryWorkerState:     true,
		IncludePrimaryWorkerFleet:     true,
		IncludePrimaryWorkerManager:   true,
		IncludeSecondaryWorker:        true,
		IncludeSecondaryWorkerState:   true,
		IncludeSecondaryWorkerFleet:   true,
		IncludeSecondaryWorkerManager: true,
		IncludeLastKnownLocation:      true,
		IncludeCustomFields:           true,
	}
}

type ListTractorsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
	TractorRelationIncludes
	Status string `json:"status"`
}

type GetTractorByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	TractorRelationIncludes
}

type BulkUpdateTractorStatusRequest struct {
	TenantInfo pagination.TenantInfo       `json:"-"`
	TractorIDs []pulid.ID                  `json:"tractorIds"`
	Status     domaintypes.EquipmentStatus `json:"status"`
}

type GetTractorsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TractorIDs []pulid.ID            `json:"tractorIds"`
	TractorRelationIncludes
}

type TractorSelectOptionsRequest struct {
	SelectOptionsRequest *pagination.SelectQueryRequest `json:"-"`
}

type LocateTractorRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	TractorID     pulid.ID              `json:"tractorId"`
	NewLocationID pulid.ID              `json:"newLocationId"`
}

func (r *LocateTractorRequest) Validate() *errortypes.MultiError {
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
	if r.TractorID.IsNil() {
		me.Add("tractorId", errortypes.ErrRequired, "Tractor ID is required")
	}
	if r.NewLocationID.IsNil() {
		me.Add("newLocationId", errortypes.ErrRequired, "New location ID is required")
	}
	if me.HasErrors() {
		return me
	}

	return nil
}

type TractorRepository interface {
	List(
		ctx context.Context,
		req *ListTractorsRequest,
	) (*pagination.CursorListResult[*tractor.Tractor], error)
	GetByID(
		ctx context.Context,
		req GetTractorByIDRequest,
	) (*tractor.Tractor, error)
	Create(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*tractor.Tractor, error)
	Update(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*tractor.Tractor, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateTractorStatusRequest,
	) ([]*tractor.Tractor, error)
	SelectOptions(
		ctx context.Context,
		req *TractorSelectOptionsRequest,
	) (*pagination.ListResult[*tractor.Tractor], error)
	GetByIDs(
		ctx context.Context,
		req GetTractorsByIDsRequest,
	) ([]*tractor.Tractor, error)
}
