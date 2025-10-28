package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type ShipmentOptions struct {
	ExpandShipmentDetails bool   `form:"expandShipmentDetails" json:"expandShipmentDetails"`
	Status                string `form:"status"                json:"status"`
}

type ListShipmentRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
	ShipmentOptions
}

type GetShipmentByIDRequest struct {
	ID              pulid.ID        `json:"id"              form:"id"`
	OrgID           pulid.ID        `json:"orgId"           form:"orgId"`
	BuID            pulid.ID        `json:"buId"            form:"buId"`
	UserID          pulid.ID        `json:"userId"          form:"userId"`
	ShipmentOptions ShipmentOptions `json:"shipmentOptions" form:"shipmentOptions"`
}

type GetShipmentsByIDsRequest struct {
	IDs             []pulid.ID      `json:"ids"             form:"ids"`
	OrgID           pulid.ID        `json:"orgId"           form:"orgId"`
	BuID            pulid.ID        `json:"buId"            form:"buId"`
	ShipmentOptions ShipmentOptions `json:"shipmentOptions" form:"shipmentOptions"`
}

type UpdateShipmentStatusRequest struct {
	GetOpts *GetShipmentByIDRequest `json:"getOpts" form:"getOpts"`
	Status  shipment.Status         `json:"status"  form:"status"`
}

type CancelShipmentRequest struct {
	ShipmentID   pulid.ID `json:"shipmentId"   form:"shipmentId"`
	OrgID        pulid.ID `json:"orgId"        form:"orgId"`
	BuID         pulid.ID `json:"buId"         form:"buId"`
	CanceledByID pulid.ID `json:"canceledById" form:"canceledById"`
	CanceledAt   int64    `json:"canceledAt"   form:"canceledAt"`
	CancelReason string   `json:"cancelReason" form:"cancelReason"`
}

type UnCancelShipmentRequest struct {
	ShipmentID         pulid.ID `json:"shipmentId"         form:"shipmentId"`
	OrgID              pulid.ID `json:"orgId"              form:"orgId"`
	BuID               pulid.ID `json:"buId"               form:"buId"`
	UserID             pulid.ID `json:"userId"             form:"userId"`
	UpdateAppointments bool     `json:"updateAppointments" form:"updateAppointments" default:"false"`
}

type TransferOwnershipRequest struct {
	ShipmentID pulid.ID `json:"shipmentId" form:"shipmentId"`
	OrgID      pulid.ID `json:"orgId"      form:"orgId"`
	BuID       pulid.ID `json:"buId"       form:"buId"`
	UserID     pulid.ID `json:"userId"     form:"userId"`
	OwnerID    pulid.ID `json:"ownerId"    form:"ownerId"`
}

type DuplicateShipmentRequest struct {
	ShipmentID               pulid.ID `json:"shipmentId"               form:"shipmentId"`
	OrgID                    pulid.ID `json:"orgId"                    form:"orgId"`
	BuID                     pulid.ID `json:"buId"                     form:"buId"`
	UserID                   pulid.ID `json:"userId"                   form:"userId"`
	Count                    int      `json:"count"                    form:"count"                    default:"1"`
	OverrideDates            bool     `json:"overrideDates"            form:"overrideDates"            default:"false"`
	IncludeCommodities       bool     `json:"includeCommodities"       form:"includeCommodities"       default:"false"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges" form:"includeAdditionalCharges" default:"false"`
	IncludeComments          bool     `json:"includeComments"          form:"includeComments"          default:"false"`
}

func (dr *DuplicateShipmentRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		dr,
		validation.Field(&dr.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&dr.UserID, validation.Required.Error("User ID is required")),
		validation.Field(&dr.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&dr.BuID, validation.Required.Error("Business Unit ID is required")),
		validation.Field(&dr.Count, validation.Required.Error("Count is required")),
		validation.Field(
			&dr.Count,
			validation.Min(1).Error("Count must be at least 1"),
			validation.Max(20).Error("Count must be at most 20"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type DuplicateBolsRequest struct {
	CurrentBOL string    `json:"currentBOL"`
	OrgID      pulid.ID  `json:"orgId"`
	BuID       pulid.ID  `json:"buId"`
	ExcludeID  *pulid.ID `json:"excludeId"`
}

func (dr *DuplicateBolsRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		dr,
		validation.Field(&dr.CurrentBOL, validation.Required.Error("Current BOL is required")),
		validation.Field(&dr.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&dr.BuID, validation.Required.Error("Business Unit ID is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type DuplicateBOLsResult struct {
	ID        pulid.ID `bun:"id"`
	ProNumber string   `bun:"pro_number"`
}

type ShipmentTotalsResponse struct {
	BaseCharge        decimal.Decimal `json:"baseCharge"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`
}

type GetShipmentsByDateRangeRequest struct {
	OrgID      pulid.ID  `json:"orgId"`
	StartDate  int64     `json:"startDate"`  // Unix timestamp
	EndDate    int64     `json:"endDate"`    // Unix timestamp
	CustomerID *pulid.ID `json:"customerId"` // Optional customer filter
}

type GetPreviousRatesRequest struct {
	UserID                pulid.ID  `json:"userId"`
	OrgID                 pulid.ID  `json:"orgId"`
	BuID                  pulid.ID  `json:"buId"`
	OriginLocationID      pulid.ID  `json:"originLocationId"`
	DestinationLocationID pulid.ID  `json:"destinationLocationId"`
	ShipmentTypeID        pulid.ID  `json:"shipmentTypeId"`
	ServiceTypeID         pulid.ID  `json:"serviceTypeId"`
	CustomerID            *pulid.ID `json:"customerId"            form:"customerId"`
	ExcludeShipmentID     *pulid.ID `json:"excludeShipmentId"     form:"excludeShipmentId"`
}

type BulkCancelShipmentsByCreatedAtRequest struct {
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
	CreatedAt int64    `json:"createdAt"`
}

type ShipmentRepository interface {
	List(
		ctx context.Context,
		opts *ListShipmentRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	GetByID(ctx context.Context, req *GetShipmentByIDRequest) (*shipment.Shipment, error)
	GetByIDs(ctx context.Context, req *GetShipmentsByIDsRequest) ([]*shipment.Shipment, error)
	GetPreviousRates(
		ctx context.Context,
		req *GetPreviousRatesRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	// GetAll(ctx context.Context) (*pagination.ListResult[*shipment.Shipment], error)
	GetByOrgID(
		ctx context.Context,
		orgID pulid.ID,
	) (*pagination.ListResult[*shipment.Shipment], error)
	// GetByDateRange(
	// 	ctx context.Context,
	// 	req *GetShipmentsByDateRangeRequest,
	// ) (*pagination.ListResult[*shipment.Shipment], error)
	GetDelayedShipments(ctx context.Context) ([]*shipment.Shipment, error)
	Create(ctx context.Context, t *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error)
	Update(ctx context.Context, t *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error)
	// UpdateStatus(ctx context.Context, opts *UpdateShipmentStatusRequest) (*shipment.Shipment, error)
	Cancel(ctx context.Context, req *CancelShipmentRequest) (*shipment.Shipment, error)
	UnCancel(ctx context.Context, req *UnCancelShipmentRequest) (*shipment.Shipment, error)
	TransferOwnership(
		ctx context.Context,
		req *TransferOwnershipRequest,
	) (*shipment.Shipment, error)
	BulkDuplicate(ctx context.Context, req *DuplicateShipmentRequest) ([]*shipment.Shipment, error)
	DelayShipments(ctx context.Context) ([]*shipment.Shipment, error)
	// BulkCancelShipmentsByCreatedAt(
	// 	ctx context.Context,
	// 	req *BulkCancelShipmentsByCreatedAtRequest,
	// ) ([]*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		req *DuplicateBolsRequest,
	) ([]*DuplicateBOLsResult, error)
	CalculateTotals(
		ctx context.Context,
		shp *shipment.Shipment,
		userID pulid.ID,
	) (*ShipmentTotalsResponse, error)
}
