package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListServiceFailuresRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	ShipmentID pulid.ID                 `json:"shipmentId"`
}

type ListServiceFailureConnectionRequest struct {
	Filter                *pagination.QueryOptions `json:"filter"`
	Cursor                pagination.CursorInfo    `json:"-"`
	ServiceFailureColumns []string                 `json:"-"`
	ShipmentID            *pulid.ID                `json:"-"`
}

type GetServiceFailureByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetServiceFailureByShipmentRequest struct {
	ID         pulid.ID              `json:"id"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ServiceFailureActiveStopRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentID     pulid.ID              `json:"shipmentId"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	StopID         pulid.ID              `json:"stopId"`
	Type           servicefailure.Type   `json:"type"`
}

type ServiceFailuresByShipmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

func (r *ListServiceFailuresRequest) EnsureFilter() {
	if r.Filter == nil {
		r.Filter = &pagination.QueryOptions{}
	}
}

func (r *GetServiceFailureByIDRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure request is required")
		return multiErr
	}
	if r.ID.IsNil() {
		multiErr.Add("id", errortypes.ErrRequired, "Service failure ID is required")
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *GetServiceFailureByShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure request is required")
		return multiErr
	}
	if r.ID.IsNil() {
		multiErr.Add("id", errortypes.ErrRequired, "Service failure ID is required")
	}
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *ServiceFailureActiveStopRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure stop request is required")
		return multiErr
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move ID is required")
	}
	if r.StopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")
	}
	if !r.Type.IsValid() {
		multiErr.Add("type", errortypes.ErrInvalid, "Service failure type is invalid")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *ServiceFailuresByShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failures by shipment request is required")
		return multiErr
	}
	validateTenantInfo(multiErr, r.TenantInfo)
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type ServiceFailureRepository interface {
	List(
		ctx context.Context,
		req *ListServiceFailuresRequest,
	) (*pagination.ListResult[*servicefailure.ServiceFailure], error)
	ListConnection(
		ctx context.Context,
		req *ListServiceFailureConnectionRequest,
	) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error)
	GetByID(
		ctx context.Context,
		req *GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	GetByShipment(
		ctx context.Context,
		req *GetServiceFailureByShipmentRequest,
	) (*servicefailure.ServiceFailure, error)
	Create(
		ctx context.Context,
		entity *servicefailure.ServiceFailure,
	) (*servicefailure.ServiceFailure, error)
	Update(
		ctx context.Context,
		entity *servicefailure.ServiceFailure,
	) (*servicefailure.ServiceFailure, error)
	UpdateDetectionSnapshot(
		ctx context.Context,
		entity *servicefailure.ServiceFailure,
	) (*servicefailure.ServiceFailure, error)
	FindUnresolvedByStop(
		ctx context.Context,
		req *ServiceFailureActiveStopRequest,
	) (*servicefailure.ServiceFailure, error)
	ListUnresolvedByShipment(
		ctx context.Context,
		req *ServiceFailuresByShipmentRequest,
	) ([]*servicefailure.ServiceFailure, error)
	CountUnresolvedByShipment(ctx context.Context, req *ServiceFailuresByShipmentRequest) (int, error)
}
