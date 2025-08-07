/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
		"proNumber":                true,
		"status":                   true,
		"bol":                      true,
		"customer.name":            true,
		"originLocation.name":      true,
		"destinationLocation.name": true,
		"originDate":               true,
		"destinationDate":          true,
		"consolidationGroupId":     true,
	},
	SortableFields: map[string]bool{
		"proNumber":                true,
		"status":                   true,
		"bol":                      true,
		"customer.name":            true,
		"originLocation.name":      true,
		"destinationLocation.name": true,
		"originDate":               true,
		"destinationDate":          true,
		"createdAt":                true,
		"consolidationGroupId":     true,
	},
	FieldMap: map[string]string{
		"proNumber":            "pro_number",
		"status":               "status",
		"bol":                  "bol",
		"createdAt":            "created_at",
		"consolidationGroupId": "consolidation_group_id",
	},
	EnumMap: map[string]bool{
		"status": true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{
		"customer.name": {
			DatabaseField: "cust.name",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "customers",
					Alias:     "cust",
					Condition: "sp.customer_id = cust.id",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
		"originLocation.name": {
			DatabaseField: "orig_loc.name",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "shipment_moves",
					Alias:     "sm_orig",
					Condition: "sp.id = sm_orig.shipment_id",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "stops",
					Alias:     "stop_orig",
					Condition: "sm_orig.id = stop_orig.shipment_move_id AND stop_orig.type = 'Pickup' AND stop_orig.sequence = 0",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "locations",
					Alias:     "orig_loc",
					Condition: "stop_orig.location_id = orig_loc.id",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
		"destinationLocation.name": {
			DatabaseField: "dest_loc.name",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "shipment_moves",
					Alias:     "sm_dest",
					Condition: "sp.id = sm_dest.shipment_id",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "stops",
					Alias:     "stop_dest",
					Condition: "sm_dest.id = stop_dest.shipment_move_id AND stop_dest.type = 'Delivery'",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "locations",
					Alias:     "dest_loc",
					Condition: "stop_dest.location_id = dest_loc.id",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
		"originDate": {
			DatabaseField: "stop_orig_date.planned_arrival",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "shipment_moves",
					Alias:     "sm_orig_date",
					Condition: "sp.id = sm_orig_date.shipment_id",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "stops",
					Alias:     "stop_orig_date",
					Condition: "sm_orig_date.id = stop_orig_date.shipment_move_id AND stop_orig_date.type = 'Pickup' AND stop_orig_date.sequence = 0",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
		"destinationDate": {
			DatabaseField: "stop_dest_date.planned_arrival",
			RequiredJoins: []ports.JoinDefinition{
				{
					Table:     "shipment_moves",
					Alias:     "sm_dest_date",
					Condition: "sp.id = sm_dest_date.shipment_id",
					JoinType:  ports.JoinTypeLeft,
				},
				{
					Table:     "stops",
					Alias:     "stop_dest_date",
					Condition: "sm_dest_date.id = stop_dest_date.shipment_move_id AND stop_dest_date.type = 'Delivery'",
					JoinType:  ports.JoinTypeLeft,
				},
			},
			IsEnum: false,
		},
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
	GetOpts *GetShipmentByIDOptions `json:"getOpts" query:"getOpts"`
	Status  shipment.Status         `json:"status"  query:"status"`
}

type CancelShipmentRequest struct {
	ShipmentID   pulid.ID `json:"shipmentId"   query:"shipmentId"`
	OrgID        pulid.ID `json:"orgId"        query:"orgId"`
	BuID         pulid.ID `json:"buId"         query:"buId"`
	CanceledByID pulid.ID `json:"canceledById" query:"canceledById"`
	CanceledAt   int64    `json:"canceledAt"   query:"canceledAt"`
	CancelReason string   `json:"cancelReason" query:"cancelReason"`
}

type UnCancelShipmentRequest struct {
	ShipmentID         pulid.ID `json:"shipmentId"         query:"shipmentId"`
	OrgID              pulid.ID `json:"orgId"              query:"orgId"`
	BuID               pulid.ID `json:"buId"               query:"buId"`
	UserID             pulid.ID `json:"userId"             query:"userId"`
	UpdateAppointments bool     `json:"updateAppointments" query:"updateAppointments" default:"false"`
}

type TransferOwnershipRequest struct {
	ShipmentID pulid.ID `json:"shipmentId" query:"shipmentId"`
	OrgID      pulid.ID `json:"orgId"      query:"orgId"`
	BuID       pulid.ID `json:"buId"       query:"buId"`
	UserID     pulid.ID `json:"userId"     query:"userId"`
	OwnerID    pulid.ID `json:"ownerId"    query:"ownerId"`
}

type DuplicateShipmentRequest struct {
	ShipmentID               pulid.ID `json:"shipmentId"               query:"shipmentId"`
	OrgID                    pulid.ID `json:"orgId"                    query:"orgId"`
	BuID                     pulid.ID `json:"buId"                     query:"buId"`
	UserID                   pulid.ID `json:"userId"                   query:"userId"`
	Count                    int      `json:"count"                    query:"count"                    default:"1"`
	OverrideDates            bool     `json:"overrideDates"            query:"overrideDates"            default:"false"`
	IncludeCommodities       bool     `json:"includeCommodities"       query:"includeCommodities"       default:"false"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges" query:"includeAdditionalCharges" default:"false"`
	IncludeComments          bool     `json:"includeComments"          query:"includeComments"          default:"false"`
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

type GetPreviousRatesRequest struct {
	UserID                pulid.ID `json:"userId"`
	OrgID                 pulid.ID `json:"orgId"`
	BuID                  pulid.ID `json:"buId"`
	OriginLocationID      pulid.ID `json:"originLocationId"`
	DestinationLocationID pulid.ID `json:"destinationLocationId"`
	ShipmentTypeID        pulid.ID `json:"shipmentTypeId"`
	ServiceTypeID         pulid.ID `json:"serviceTypeId"`

	// * Optional Customer filter
	CustomerID *pulid.ID `json:"customerId" query:"customerId"`

	// * Optional Exclude Shipment ID
	ExcludeShipmentID *pulid.ID `json:"excludeShipmentId" query:"excludeShipmentId"`
}

type DelayShipmentRequest struct{}

type ShipmentRepository interface {
	List(
		ctx context.Context,
		opts *ListShipmentOptions,
	) (*ports.ListResult[*shipment.Shipment], error)
	GetPreviousRates(
		ctx context.Context,
		req *GetPreviousRatesRequest,
	) (*ports.ListResult[*shipment.Shipment], error)
	GetAll(ctx context.Context) (*ports.ListResult[*shipment.Shipment], error)
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*ports.ListResult[*shipment.Shipment], error)
	GetByDateRange(
		ctx context.Context,
		req *GetShipmentsByDateRangeRequest,
	) (*ports.ListResult[*shipment.Shipment], error)
	GetByID(ctx context.Context, opts *GetShipmentByIDOptions) (*shipment.Shipment, error)
	Create(ctx context.Context, t *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error)
	Update(ctx context.Context, t *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error)
	UpdateStatus(ctx context.Context, opts *UpdateShipmentStatusRequest) (*shipment.Shipment, error)
	Cancel(ctx context.Context, req *CancelShipmentRequest) (*shipment.Shipment, error)
	TransferOwnership(
		ctx context.Context,
		req *TransferOwnershipRequest,
	) (*shipment.Shipment, error)
	UnCancel(ctx context.Context, req *UnCancelShipmentRequest) (*shipment.Shipment, error)
	BulkDuplicate(ctx context.Context, req *DuplicateShipmentRequest) ([]*shipment.Shipment, error)
	GetDelayedShipments(ctx context.Context) ([]*shipment.Shipment, error)
	DelayShipments(ctx context.Context) ([]*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		currentBOL string,
		orgID pulid.ID,
		buID pulid.ID,
		excludeID *pulid.ID,
	) ([]DuplicateBOLsResult, error)
	CalculateShipmentTotals(
		ctx context.Context,
		shp *shipment.Shipment,
		userID pulid.ID,
	) (*ShipmentTotalsResponse, error)
}
