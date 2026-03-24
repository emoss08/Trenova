package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type ShipmentOptions struct {
	ExpandShipmentDetails bool   `form:"expandShipmentDetails" json:"expandShipmentDetails" query:"expandShipmentDetails"`
	Status                string `form:"status"                json:"status"                query:"status"`
}

type ListShipmentsRequest struct {
	Filter          *pagination.QueryOptions `json:"filter"`
	ShipmentOptions ShipmentOptions          `json:"shipmentOptions"`
}

type GetShipmentByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	ShipmentOptions
}

type CancelShipmentRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	ShipmentID   pulid.ID              `json:"shipmentId"`
	CanceledByID pulid.ID              `json:"-"`
	CanceledAt   int64                 `json:"-"`
	CancelReason string                `json:"cancelReason"`
}

func (r *CancelShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type UncancelShipmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

func (r *UncancelShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type TransferOwnershipRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	OwnerID    pulid.ID              `json:"ownerId"`
}

func (r *TransferOwnershipRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.OwnerID, validation.Required.Error("Owner ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type DuplicateBOLCheckRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	BOL        string                `json:"bol"`
	ShipmentID *pulid.ID             `json:"shipmentId,omitempty"`
}

func (r *DuplicateBOLCheckRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.BOL, validation.Required.Error("BOL is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type BulkDuplicateShipmentRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	ShipmentID    pulid.ID              `json:"shipmentId"`
	Count         int                   `json:"count"`
	OverrideDates bool                  `json:"overrideDates"`
}

func (r *BulkDuplicateShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}

	if r.TenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}

	if r.TenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}

	if r.TenantInfo.UserID.IsNil() {
		multiErr.Add("userId", errortypes.ErrRequired, "User ID is required")
	}

	switch {
	case r.Count == 0:
		multiErr.Add("count", errortypes.ErrRequired, "Count is required")
	case r.Count < 1:
		multiErr.Add("count", errortypes.ErrInvalid, "Count must be at least 1")
	case r.Count > 20:
		multiErr.Add("count", errortypes.ErrInvalid, "Count must be at most 20")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type GetDelayedShipmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

func (r *GetDelayedShipmentsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type DelayShipmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

func (r *DelayShipmentsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type GetAutoCancelableShipmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

func (r *GetAutoCancelableShipmentsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type AutoCancelShipmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

func (r *AutoCancelShipmentsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type GetPreviousRatesRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	OriginLocationID      pulid.ID              `json:"originLocationId"`
	DestinationLocationID pulid.ID              `json:"destinationLocationId"`
	ShipmentTypeID        pulid.ID              `json:"shipmentTypeId"`
	ServiceTypeID         pulid.ID              `json:"serviceTypeId"`
	CustomerID            *pulid.ID             `json:"customerId,omitempty"`
	ExcludeShipmentID     *pulid.ID             `json:"excludeShipmentId,omitempty"`
}

func (r *GetPreviousRatesRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.OriginLocationID.IsNil() {
		multiErr.Add("originLocationId", errortypes.ErrRequired, "Origin location ID is required")
	}

	if r.DestinationLocationID.IsNil() {
		multiErr.Add(
			"destinationLocationId",
			errortypes.ErrRequired,
			"Destination location ID is required",
		)
	}

	if r.ShipmentTypeID.IsNil() {
		multiErr.Add("shipmentTypeId", errortypes.ErrRequired, "Shipment type ID is required")
	}

	if r.ServiceTypeID.IsNil() {
		multiErr.Add("serviceTypeId", errortypes.ErrRequired, "Service type ID is required")
	}

	if r.TenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}

	if r.TenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type ShipmentTotalsResponse struct {
	BaseCharge        decimal.Decimal `json:"baseCharge"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`
}

type ShipmentDuplicateWorkflowResponse struct {
	WorkflowID  string `json:"workflowId"`
	RunID       string `json:"runId"`
	TaskQueue   string `json:"taskQueue"`
	Status      string `json:"status"`
	SubmittedAt int64  `json:"submittedAt"`
}

type DuplicateBOLResult struct {
	ID        pulid.ID `bun:"id"`
	ProNumber string   `bun:"pro_number"`
}

type PreviousRateSummary struct {
	ShipmentID          pulid.ID        `json:"shipmentId"          bun:"shipment_id"`
	ProNumber           string          `json:"proNumber"           bun:"pro_number"`
	CustomerID          pulid.ID        `json:"customerId"          bun:"customer_id"`
	ServiceTypeID       pulid.ID        `json:"serviceTypeId"       bun:"service_type_id"`
	ShipmentTypeID      pulid.ID        `json:"shipmentTypeId"      bun:"shipment_type_id"`
	FormulaTemplateID   pulid.ID        `json:"formulaTemplateId"   bun:"formula_template_id"`
	FreightChargeAmount decimal.Decimal `json:"freightChargeAmount" bun:"freight_charge_amount"`
	OtherChargeAmount   decimal.Decimal `json:"otherChargeAmount"   bun:"other_charge_amount"`
	TotalChargeAmount   decimal.Decimal `json:"totalChargeAmount"   bun:"total_charge_amount"`
	RatingUnit          int64           `json:"ratingUnit"          bun:"rating_unit"`
	Pieces              *int64          `json:"pieces"              bun:"pieces"`
	Weight              *int64          `json:"weight"              bun:"weight"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at"`
}

type GetShipmentsByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ShipmentIDs []pulid.ID            `json:"shipmentIds"`
}

type ShipmentRepository interface {
	List(
		ctx context.Context,
		req *ListShipmentsRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
	GetByIDs(
		ctx context.Context,
		req *GetShipmentsByIDsRequest,
	) ([]*shipment.Shipment, error)
	GetPreviousRates(
		ctx context.Context,
		req *GetPreviousRatesRequest,
	) (*pagination.ListResult[*PreviousRateSummary], error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
	UpdateDerivedState(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
	Cancel(
		ctx context.Context,
		req *CancelShipmentRequest,
	) (*shipment.Shipment, error)
	Uncancel(
		ctx context.Context,
		req *UncancelShipmentRequest,
	) (*shipment.Shipment, error)
	TransferOwnership(
		ctx context.Context,
		req *TransferOwnershipRequest,
	) (*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		req *DuplicateBOLCheckRequest,
	) ([]*DuplicateBOLResult, error)
	BulkDuplicate(
		ctx context.Context,
		req *BulkDuplicateShipmentRequest,
	) ([]*shipment.Shipment, error)
	GetDelayedShipments(
		ctx context.Context,
		req *GetDelayedShipmentsRequest,
		thresholdMinutes int16,
	) ([]*shipment.Shipment, error)
	DelayShipments(
		ctx context.Context,
		req *DelayShipmentsRequest,
		thresholdMinutes int16,
	) ([]*shipment.Shipment, error)
	GetAutoCancelableShipments(
		ctx context.Context,
		req *GetAutoCancelableShipmentsRequest,
		thresholdDays int8,
	) ([]*shipment.Shipment, error)
	AutoCancelShipments(
		ctx context.Context,
		req *AutoCancelShipmentsRequest,
		thresholdDays int8,
	) ([]*shipment.Shipment, error)
	AutoDelayShipments(ctx context.Context) ([]*shipment.Shipment, error)
	RunAutoCancelShipments(ctx context.Context) ([]*shipment.Shipment, error)
}
