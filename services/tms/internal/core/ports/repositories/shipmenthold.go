package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ListShipmentHoldsRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	ShipmentID pulid.ID                 `json:"shipmentId"`
}

type GetShipmentHoldByIDRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	HoldID     pulid.ID              `json:"holdId"`
}

type CreateShipmentHoldRequest struct {
	TenantInfo         pagination.TenantInfo      `json:"-"`
	ShipmentID         pulid.ID                   `json:"shipmentId"`
	HoldReasonID       pulid.ID                   `json:"holdReasonId"`
	Notes              string                     `json:"notes"`
	Severity           *holdreason.HoldSeverity   `json:"severity,omitempty"`
	BlocksDispatch     *bool                      `json:"blocksDispatch,omitempty"`
	BlocksDelivery     *bool                      `json:"blocksDelivery,omitempty"`
	BlocksBilling      *bool                      `json:"blocksBilling,omitempty"`
	VisibleToCustomer  *bool                      `json:"visibleToCustomer,omitempty"`
	StartedAt          *int64                     `json:"startedAt,omitempty"`
}

type UpdateShipmentHoldRequest struct {
	TenantInfo         pagination.TenantInfo    `json:"-"`
	ShipmentID         pulid.ID                 `json:"shipmentId"`
	HoldID             pulid.ID                 `json:"holdId"`
	Severity           holdreason.HoldSeverity  `json:"severity"`
	Notes              string                   `json:"notes"`
	BlocksDispatch     bool                     `json:"blocksDispatch"`
	BlocksDelivery     bool                     `json:"blocksDelivery"`
	BlocksBilling      bool                     `json:"blocksBilling"`
	VisibleToCustomer  bool                     `json:"visibleToCustomer"`
	StartedAt          int64                    `json:"startedAt"`
	Version            int64                    `json:"version"`
}

type ReleaseShipmentHoldRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	HoldID     pulid.ID              `json:"holdId"`
}

type ActiveShipmentHoldRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

type ShipmentHoldRepository interface {
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentHoldsRequest,
	) (*pagination.ListResult[*shipment.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Create(
		ctx context.Context,
		entity *shipment.ShipmentHold,
	) (*shipment.ShipmentHold, error)
	Update(
		ctx context.Context,
		entity *shipment.ShipmentHold,
	) (*shipment.ShipmentHold, error)
	Release(
		ctx context.Context,
		entity *shipment.ShipmentHold,
	) (*shipment.ShipmentHold, error)
	HasActiveDispatchHold(
		ctx context.Context,
		req *ActiveShipmentHoldRequest,
	) (bool, error)
	HasActiveDeliveryHold(
		ctx context.Context,
		req *ActiveShipmentHoldRequest,
	) (bool, error)
}

func (r *GetShipmentHoldByIDRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.HoldID, validation.Required.Error("Hold ID is required")),
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

func (r *CreateShipmentHoldRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.HoldReasonID, validation.Required.Error("Hold reason ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
		validation.Field(&r.StartedAt, validation.By(func(value any) error {
			startedAt, _ := value.(*int64)
			if startedAt != nil && *startedAt <= 0 {
				return errors.New("Started At must be greater than zero")
			}
			return nil
		})),
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

func (r *UpdateShipmentHoldRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.HoldID, validation.Required.Error("Hold ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
		validation.Field(&r.StartedAt, validation.Min(int64(1)).Error("Started At must be greater than zero")),
		validation.Field(&r.Version, validation.Min(int64(0)).Error("Version is required")),
		validation.Field(&r.Severity,
			validation.Required.Error("Severity is required"),
			validation.In(
				holdreason.HoldSeverityInformational,
				holdreason.HoldSeverityAdvisory,
				holdreason.HoldSeverityBlocking,
			).Error("Invalid hold severity"),
		),
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

func (r *ReleaseShipmentHoldRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.HoldID, validation.Required.Error("Hold ID is required")),
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

func (r *ActiveShipmentHoldRequest) Validate() *errortypes.MultiError {
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
