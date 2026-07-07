package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentUIPolicy struct {
	AllowMoveRemovals      bool  `json:"allowMoveRemovals"`
	CheckForDuplicateBOLs  bool  `json:"checkForDuplicateBols"`
	CheckHazmatSegregation bool  `json:"checkHazmatSegregation"`
	MaxShipmentWeightLimit int32 `json:"maxShipmentWeightLimit"`
}

type ShipmentBillingReadinessPolicy struct {
	ShipmentBillingRequirementEnforcement tenant.EnforcementLevel            `json:"shipmentBillingRequirementEnforcement"`
	RateValidationEnforcement             tenant.EnforcementLevel            `json:"rateValidationEnforcement"`
	BillingExceptionDisposition           tenant.BillingExceptionDisposition `json:"billingExceptionDisposition"`
	NotifyOnBillingExceptions             bool                               `json:"notifyOnBillingExceptions"`
	ReadyToBillAssignmentMode             tenant.ReadyToBillAssignmentMode   `json:"readyToBillAssignmentMode"`
	BillingQueueTransferMode              tenant.BillingQueueTransferMode    `json:"billingQueueTransferMode"`
}

type ShipmentBillingValidation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ShipmentBillingWarning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

type ShipmentServiceFailureBillingContext struct {
	HasUnresolved     bool     `json:"hasUnresolved"`
	UnresolvedCount   int      `json:"unresolvedCount"`
	ServiceFailureIDs []string `json:"serviceFailureIds"`
}

type ShipmentBillingRequirement struct {
	DocumentTypeID   string   `json:"documentTypeId"`
	DocumentTypeCode string   `json:"documentTypeCode"`
	DocumentTypeName string   `json:"documentTypeName"`
	Satisfied        bool     `json:"satisfied"`
	DocumentCount    int      `json:"documentCount"`
	DocumentIDs      []string `json:"documentIds"`
}

type ShipmentBillingReadiness struct {
	ShipmentID                   string                               `json:"shipmentId"`
	ShipmentStatus               shipment.Status                      `json:"shipmentStatus"`
	Policy                       ShipmentBillingReadinessPolicy       `json:"policy"`
	Requirements                 []ShipmentBillingRequirement         `json:"requirements"`
	MissingRequirements          []ShipmentBillingRequirement         `json:"missingRequirements"`
	ValidationFailures           []ShipmentBillingValidation          `json:"validationFailures"`
	Warnings                     []ShipmentBillingWarning             `json:"warnings"`
	ServiceFailureContext        ShipmentServiceFailureBillingContext `json:"serviceFailureContext"`
	CanMarkReadyToInvoice        bool                                 `json:"canMarkReadyToInvoice"`
	ShouldAutoMarkReadyToInvoice bool                                 `json:"shouldAutoMarkReadyToInvoice"`
	ShouldAutoTransferToBilling  bool                                 `json:"shouldAutoTransferToBilling"`
}

type DistanceMoveResult struct {
	MoveID              pulid.ID `json:"moveId,omitempty"`
	MoveIndex           int      `json:"moveIndex"`
	Distance            float64  `json:"distance"`
	Source              string   `json:"source"`
	Provider            string   `json:"provider,omitempty"`
	RoutingType         string   `json:"routingType,omitempty"`
	DataVersion         string   `json:"dataVersion,omitempty"`
	DistanceUnits       string   `json:"distanceUnits,omitempty"`
	DistanceProfileID   string   `json:"distanceProfileId,omitempty"`
	DistanceProfileName string   `json:"distanceProfileName,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
	CalculatedAt        int64    `json:"calculatedAt"`
}

type DistanceCalculationResponse struct {
	ShipmentID    pulid.ID             `json:"shipmentId,omitempty"`
	TotalDistance float64              `json:"totalDistance"`
	Moves         []DistanceMoveResult `json:"moves"`
}

type DistanceCalculationService interface {
	ResolveForShipment(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*DistanceCalculationResponse, error)
	RecalculateShipment(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*DistanceCalculationResponse, error)
}

type TransferShipmentToBillingRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	BillType   billingqueue.BillType `json:"billType"`
}

type BulkTransferShipmentToBillingRequest struct {
	ShipmentIDs []pulid.ID            `json:"shipmentIds"`
	BillType    billingqueue.BillType `json:"billType"`
}

type BulkTransferToBillingResult struct {
	ShipmentID pulid.ID                       `json:"shipmentId"`
	Success    bool                           `json:"success"`
	Item       *billingqueue.BillingQueueItem `json:"item,omitempty"`
	Error      string                         `json:"error,omitempty"`
}

type BulkTransferToBillingResponse struct {
	Results      []BulkTransferToBillingResult `json:"results"`
	TotalCount   int                           `json:"totalCount"`
	SuccessCount int                           `json:"successCount"`
	ErrorCount   int                           `json:"errorCount"`
}

type ShipmentMutationObserver interface {
	AfterShipmentUpdate(
		ctx context.Context,
		original *shipment.Shipment,
		updated *shipment.Shipment,
		actor *RequestActor,
	) error
}

type ShipmentService interface {
	List(
		ctx context.Context,
		req *repositories.ListShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Get(
		ctx context.Context,
		req *repositories.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
	GetByIDs(
		ctx context.Context,
		req *repositories.GetShipmentsByIDsRequest,
	) ([]*shipment.Shipment, error)
	SelectOptions(
		ctx context.Context,
		req *repositories.ShipmentSelectOptionsRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	GetUIPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentUIPolicy, error)
	GetBillingReadiness(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentBillingReadiness, error)
	GetPreviousRates(
		ctx context.Context,
		req *repositories.GetPreviousRatesRequest,
	) (*pagination.ListResult[*repositories.PreviousRateSummary], error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	GetUnassigned(
		ctx context.Context,
		req *repositories.GetUnassignedShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	Cancel(
		ctx context.Context,
		req *repositories.CancelShipmentRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	Uncancel(
		ctx context.Context,
		req *repositories.UncancelShipmentRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	TransferOwnership(
		ctx context.Context,
		req *repositories.TransferOwnershipRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		req *repositories.DuplicateBOLCheckRequest,
	) error
	CheckHazmatSegregation(
		ctx context.Context,
		req *repositories.CheckHazmatSegregationRequest,
	) error
	CalculateLoadingOptimization(
		ctx context.Context,
		req *repositories.LoadingOptimizationRequest,
	) (*repositories.LoadingOptimizationResult, error)
	GetDelayedShipments(
		ctx context.Context,
		req *repositories.GetDelayedShipmentsRequest,
	) ([]*shipment.Shipment, error)
	DelayShipments(
		ctx context.Context,
		req *repositories.DelayShipmentsRequest,
		actor *RequestActor,
	) ([]*shipment.Shipment, error)
	GetAutoCancelableShipments(
		ctx context.Context,
		req *repositories.GetAutoCancelableShipmentsRequest,
	) ([]*shipment.Shipment, error)
	AutoCancelShipments(
		ctx context.Context,
		req *repositories.AutoCancelShipmentsRequest,
		actor *RequestActor,
	) ([]*shipment.Shipment, error)
	Duplicate(
		ctx context.Context,
		req *repositories.BulkDuplicateShipmentRequest,
	) (*repositories.ShipmentDuplicateWorkflowResponse, error)
	CalculateTotals(
		ctx context.Context,
		entity *shipment.Shipment,
		userID pulid.ID,
	) (*repositories.ShipmentTotalsResponse, error)
	CalculateDistance(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*DistanceCalculationResponse, error)
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*DistanceCalculationResponse, error)
	AutoMarkReadyToInvoiceIfEligible(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*shipment.Shipment, error)
	TransferToBilling(
		ctx context.Context,
		req *TransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *BulkTransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*BulkTransferToBillingResponse, error)
}
