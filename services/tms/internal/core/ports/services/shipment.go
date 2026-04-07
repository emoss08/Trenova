package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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
	EnforceCustomerBillingReq bool `json:"enforceCustomerBillingReq"`
	AutoMarkReadyToBill       bool `json:"autoMarkReadyToBill"`
}

type ShipmentBillingValidation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
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
	ShipmentID                   string                         `json:"shipmentId"`
	ShipmentStatus               shipment.Status                `json:"shipmentStatus"`
	Policy                       ShipmentBillingReadinessPolicy `json:"policy"`
	Requirements                 []ShipmentBillingRequirement   `json:"requirements"`
	MissingRequirements          []ShipmentBillingRequirement   `json:"missingRequirements"`
	ValidationFailures           []ShipmentBillingValidation    `json:"validationFailures"`
	CanMarkReadyToInvoice        bool                           `json:"canMarkReadyToInvoice"`
	ShouldAutoMarkReadyToInvoice bool                           `json:"shouldAutoMarkReadyToInvoice"`
}

type ShipmentService interface {
	List(
		ctx context.Context,
		req *repositories.ListShipmentsRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	Get(
		ctx context.Context,
		req *repositories.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
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
	AutoMarkReadyToInvoiceIfEligible(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*shipment.Shipment, error)
}
