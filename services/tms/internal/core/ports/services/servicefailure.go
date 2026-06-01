package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateManualServiceFailureRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	ShipmentID            pulid.ID              `json:"shipmentId"`
	ShipmentMoveID        pulid.ID              `json:"shipmentMoveId"`
	StopID                pulid.ID              `json:"stopId"`
	ReasonCodeID          pulid.ID              `json:"reasonCodeId"`
	Type                  servicefailure.Type   `json:"type"`
	Notes                 string                `json:"notes"`
	InternalNotes         string                `json:"internalNotes"`
	X12StatusCodeOverride string                `json:"x12StatusCodeOverride"`
	X12ReasonCodeOverride string                `json:"x12ReasonCodeOverride"`
	X12ExceptionCode      string                `json:"x12ExceptionCode"`
	ScheduledCutoff       *int64                `json:"scheduledCutoff"`
	ActualArrival         *int64                `json:"actualArrival"`
	GracePeriodMinutes    *int                  `json:"gracePeriodMinutes"`
	LateMinutes           *int64                `json:"lateMinutes"`
}

type EvaluateShipmentServiceFailuresRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	Force      bool                  `json:"force"`
}

type EvaluateStopServiceFailuresRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentID     pulid.ID              `json:"shipmentId"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	StopID         pulid.ID              `json:"stopId"`
	Force          bool                  `json:"force"`
}

type BulkEvaluateServiceFailuresRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	ShipmentIDs []pulid.ID            `json:"shipmentIds"`
	Force       bool                  `json:"force"`
}

type UpdateServiceFailureRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	ID                    pulid.ID              `json:"id"`
	ShipmentID            pulid.ID              `json:"shipmentId"`
	ReasonCodeID          pulid.ID              `json:"reasonCodeId"`
	ClearReasonCode       bool                  `json:"clearReasonCode"`
	Notes                 string                `json:"notes"`
	InternalNotes         string                `json:"internalNotes"`
	X12StatusCodeOverride string                `json:"x12StatusCodeOverride"`
	X12ReasonCodeOverride string                `json:"x12ReasonCodeOverride"`
	X12ExceptionCode      string                `json:"x12ExceptionCode"`
	Version               int64                 `json:"version"`
}

type ServiceFailureLifecycleRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	ID           pulid.ID              `json:"id"`
	ShipmentID   pulid.ID              `json:"shipmentId"`
	ReasonCodeID pulid.ID              `json:"reasonCodeId"`
	Notes        string                `json:"notes"`
	Version      int64                 `json:"version"`
}

type BuildServiceFailureEDIPayloadRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	ServiceFailureID pulid.ID              `json:"serviceFailureId"`
}

type ServiceFailureEvaluationResult struct {
	CreatedIDs   []pulid.ID                  `json:"createdIds"`
	UpdatedIDs   []pulid.ID                  `json:"updatedIds"`
	SkippedStops []ServiceFailureSkippedStop `json:"skippedStops"`
	Skipped      int                         `json:"skipped"`
}

type ServiceFailureSkippedStop struct {
	ShipmentID   pulid.ID          `json:"shipmentId"`
	StopID       pulid.ID          `json:"stopId,omitempty"`
	StopSequence int64             `json:"stopSequence,omitempty"`
	StopType     shipment.StopType `json:"stopType,omitempty"`
	Reason       string            `json:"reason"`
}

type ServiceFailureEDIPayloadResult struct {
	Payload     edi.DocumentPayload `json:"payload"`
	Diagnostics []edix12.Diagnostic `json:"diagnostics"`
}

func (r *CreateManualServiceFailureRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Manual service failure request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move ID is required")
	}
	if r.StopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")
	}
	if r.ReasonCodeID.IsNil() {
		multiErr.Add("reasonCodeId", errortypes.ErrRequired, "Reason code is required")
	}
	if !r.Type.IsValid() {
		multiErr.Add("type", errortypes.ErrInvalid, "Service failure type is invalid")
	}
	if r.ScheduledCutoff != nil && *r.ScheduledCutoff <= 0 {
		multiErr.Add("scheduledCutoff", errortypes.ErrInvalid, "Scheduled cutoff must be greater than zero")
	}
	if r.ActualArrival != nil && *r.ActualArrival <= 0 {
		multiErr.Add("actualArrival", errortypes.ErrInvalid, "Actual arrival must be greater than zero")
	}
	if r.GracePeriodMinutes != nil && *r.GracePeriodMinutes <= 0 {
		multiErr.Add("gracePeriodMinutes", errortypes.ErrInvalid, "Grace period must be greater than zero")
	}
	if r.LateMinutes != nil && *r.LateMinutes <= 0 {
		multiErr.Add("lateMinutes", errortypes.ErrInvalid, "Late minutes must be greater than zero")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *EvaluateShipmentServiceFailuresRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure evaluation request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *EvaluateStopServiceFailuresRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure stop evaluation request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if r.StopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *BulkEvaluateServiceFailuresRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Bulk service failure evaluation request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if len(r.ShipmentIDs) == 0 {
		multiErr.Add("shipmentIds", errortypes.ErrRequired, "Shipment IDs are required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *UpdateServiceFailureRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure update request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ID.IsNil() {
		multiErr.Add("id", errortypes.ErrRequired, "Service failure ID is required")
	}
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if r.Version < 0 {
		multiErr.Add("version", errortypes.ErrInvalid, "Version is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *ServiceFailureLifecycleRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure lifecycle request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ID.IsNil() {
		multiErr.Add("id", errortypes.ErrRequired, "Service failure ID is required")
	}
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if r.Version < 0 {
		multiErr.Add("version", errortypes.ErrInvalid, "Version is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *BuildServiceFailureEDIPayloadRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure EDI payload request is required")
		return multiErr
	}
	validateServiceTenantInfo(multiErr, r.TenantInfo)
	if r.ServiceFailureID.IsNil() {
		multiErr.Add("serviceFailureId", errortypes.ErrRequired, "Service failure ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type ServiceFailureEvaluator interface {
	EvaluateShipment(
		ctx context.Context,
		req *EvaluateShipmentServiceFailuresRequest,
		actor *RequestActor,
	) (*ServiceFailureEvaluationResult, error)
	EvaluateStop(
		ctx context.Context,
		req *EvaluateStopServiceFailuresRequest,
		actor *RequestActor,
	) (*ServiceFailureEvaluationResult, error)
}

type ServiceFailureService interface {
	ServiceFailureEvaluator
	List(
		ctx context.Context,
		req *repositories.ListServiceFailuresRequest,
	) (*pagination.ListResult[*servicefailure.ServiceFailure], error)
	GetByID(
		ctx context.Context,
		req *repositories.GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	GetByShipment(
		ctx context.Context,
		req *repositories.GetServiceFailureByShipmentRequest,
	) (*servicefailure.ServiceFailure, error)
	CreateManual(
		ctx context.Context,
		req *CreateManualServiceFailureRequest,
		actor *RequestActor,
	) (*servicefailure.ServiceFailure, error)
	BulkEvaluate(
		ctx context.Context,
		req *BulkEvaluateServiceFailuresRequest,
		actor *RequestActor,
	) (*ServiceFailureEvaluationResult, error)
	Update(
		ctx context.Context,
		req *UpdateServiceFailureRequest,
		actor *RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Review(
		ctx context.Context,
		req *ServiceFailureLifecycleRequest,
		actor *RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Resolve(
		ctx context.Context,
		req *ServiceFailureLifecycleRequest,
		actor *RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Void(
		ctx context.Context,
		req *ServiceFailureLifecycleRequest,
		actor *RequestActor,
	) (*servicefailure.ServiceFailure, error)
}

type ServiceFailureReasonCodeService interface {
	List(
		ctx context.Context,
		req *repositories.ListServiceFailureReasonCodesRequest,
	) (*pagination.ListResult[*servicefailure.ReasonCode], error)
	Get(
		ctx context.Context,
		req repositories.GetServiceFailureReasonCodeByIDRequest,
	) (*servicefailure.ReasonCode, error)
	SelectOptions(
		ctx context.Context,
		req *repositories.ServiceFailureReasonCodeSelectOptionsRequest,
	) (*pagination.ListResult[*servicefailure.ReasonCode], error)
	Create(ctx context.Context, entity *servicefailure.ReasonCode, actor *RequestActor) (*servicefailure.ReasonCode, error)
	Update(ctx context.Context, entity *servicefailure.ReasonCode, actor *RequestActor) (*servicefailure.ReasonCode, error)
	Archive(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		actor *RequestActor,
	) (*servicefailure.ReasonCode, error)
	Activate(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
		actor *RequestActor,
	) (*servicefailure.ReasonCode, error)
	Reorder(
		ctx context.Context,
		req *repositories.ReorderServiceFailureReasonCodesRequest,
		actor *RequestActor,
	) ([]*servicefailure.ReasonCode, error)
}

func validateServiceTenantInfo(multiErr *errortypes.MultiError, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if tenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}
}
