package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
)

var ShipmentFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"proNumber": true,
		"status":    true,
	},
	SortableFields: map[string]bool{
		"proNumber": true,
		"status":    true,
	},
	FieldMap: map[string]string{
		"proNumber": "pro_number",
		"status":    "status",
	},
	EnumMap: map[string]bool{
		"status": true,
	},
}

type ShipmentOptions struct {
	ExpandShipmentDetails bool   `query:"expandShipmentDetails"`
	Status                string `query:"status"`
}

func BuildShipmentListOptions(
	filter *ports.QueryOptions,
	additionalOpts *ListShipmentOptions,
) *ListShipmentOptions {
	return &ListShipmentOptions{
		Filter:          filter,
		ShipmentOptions: additionalOpts.ShipmentOptions,
	}
}

type ListShipmentOptions struct {
	Filter          *ports.QueryOptions `json:"filter"          query:"filter"`
	ShipmentOptions `json:"shipmentOptions" query:"shipmentOptions"`
}

type GetShipmentByIDOptions struct {
	ID              pulid.ID        `json:"id"              query:"id"`
	OrgID           pulid.ID        `json:"orgId"           query:"orgId"`
	BuID            pulid.ID        `json:"buId"            query:"buId"`
	UserID          pulid.ID        `json:"userId"          query:"userId"`
	ShipmentOptions ShipmentOptions `json:"shipmentOptions" query:"shipmentOptions"`
}

type UpdateShipmentStatusRequest struct {
	// Fetch the shipment
	GetOpts *GetShipmentByIDOptions `json:"getOpts" query:"getOpts"`

	// The status of the shipment
	Status shipment.Status `json:"status" query:"status"`
}

type CancelShipmentRequest struct {
	ShipmentID   pulid.ID `json:"shipmentId"   query:"shipmentId"`
	OrgID        pulid.ID `json:"orgId"        query:"orgId"`
	BuID         pulid.ID `json:"buId"         query:"buId"`
	CanceledByID pulid.ID `json:"canceledById" query:"canceledById"`
	CanceledAt   int64    `json:"canceledAt"   query:"canceledAt"`
	CancelReason string   `json:"cancelReason" query:"cancelReason"`
}

type DuplicateShipmentRequest struct {
	// The ID of the shipment to duplicate
	ShipmentID pulid.ID `json:"shipmentId" query:"shipmentId"`

	// The ID of the organization
	OrgID pulid.ID `json:"orgId" query:"orgId"`

	// The ID of the business unit
	BuID pulid.ID `json:"buId" query:"buId"`

	// The ID of the user who is duplicating the shipment
	UserID pulid.ID `json:"userId" query:"userId"`

	// The number of shipments to duplicate
	Count int `json:"count" default:"1" query:"count"`

	// Optional parameter to override the dates of the new shipment
	OverrideDates bool `json:"overrideDates" query:"overrideDates" default:"false"`

	// Optional parameter to include commodities in the new shipment
	IncludeCommodities bool `json:"includeCommodities" query:"includeCommodities" default:"false"`

	// Optional parameter to include additional charges in the new shipment
	IncludeAdditionalCharges bool `json:"includeAdditionalCharges" query:"includeAdditionalCharges" default:"false"`
}

func (dr *DuplicateShipmentRequest) Validate(ctx context.Context) *errors.MultiError {
	me := errors.NewMultiError()

	err := validation.ValidateStructWithContext(
		ctx,
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
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

// DuplicateBOLsResult represents the minimal data needed when checking for duplicate BOLs
type DuplicateBOLsResult struct {
	ID        pulid.ID `bun:"id"`
	ProNumber string   `bun:"pro_number"`
}

type ShipmentTotalsResponse struct {
	BaseCharge        decimal.Decimal `json:"baseCharge"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`
}

// GetShipmentsByDateRangeRequest represents request parameters for fetching shipments by date range
type GetShipmentsByDateRangeRequest struct {
	OrgID      pulid.ID  `json:"orgId"`
	StartDate  int64     `json:"startDate"`  // Unix timestamp
	EndDate    int64     `json:"endDate"`    // Unix timestamp
	CustomerID *pulid.ID `json:"customerId"` // Optional customer filter
}

type ShipmentRepository interface {
	List(
		ctx context.Context,
		opts *ListShipmentOptions,
	) (*ports.ListResult[*shipment.Shipment], error)
	GetAll(ctx context.Context) (*ports.ListResult[*shipment.Shipment], error)
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*ports.ListResult[*shipment.Shipment], error)
	GetByDateRange(
		ctx context.Context,
		req *GetShipmentsByDateRangeRequest,
	) (*ports.ListResult[*shipment.Shipment], error)
	GetByID(ctx context.Context, opts *GetShipmentByIDOptions) (*shipment.Shipment, error)
	Create(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
	Update(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
	UpdateStatus(ctx context.Context, opts *UpdateShipmentStatusRequest) (*shipment.Shipment, error)
	Cancel(ctx context.Context, req *CancelShipmentRequest) (*shipment.Shipment, error)
	BulkDuplicate(ctx context.Context, req *DuplicateShipmentRequest) ([]*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		currentBOL string,
		orgID pulid.ID,
		buID pulid.ID,
		excludeID *pulid.ID,
	) ([]DuplicateBOLsResult, error)
	CalculateShipmentTotals(shp *shipment.Shipment) (*ShipmentTotalsResponse, error)
}
