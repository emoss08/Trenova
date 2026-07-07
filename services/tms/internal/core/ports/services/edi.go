package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ServiceFailureEDITrigger string

const (
	ServiceFailureEDITriggerReviewed ServiceFailureEDITrigger = "Reviewed"
	ServiceFailureEDITriggerResolved ServiceFailureEDITrigger = "Resolved"
)

func (t ServiceFailureEDITrigger) IsValid() bool {
	switch t {
	case ServiceFailureEDITriggerReviewed, ServiceFailureEDITriggerResolved:
		return true
	default:
		return false
	}
}

type ServiceFailureEDIAction string

const (
	ServiceFailureEDIActionSkipped   ServiceFailureEDIAction = "skipped"
	ServiceFailureEDIActionGenerated ServiceFailureEDIAction = "generated"
	ServiceFailureEDIActionBlocked   ServiceFailureEDIAction = "blocked"
	ServiceFailureEDIActionDuplicate ServiceFailureEDIAction = "duplicate"
)

type PreviewEDIDocumentRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID pulid.ID              `json:"partnerDocumentProfileId"`
	EDIPartnerID             pulid.ID              `json:"ediPartnerId"`
	ShipmentID               pulid.ID              `json:"shipmentId"`
	TransferID               pulid.ID              `json:"transferId"`
	InvoiceID                pulid.ID              `json:"invoiceId"`
	ShipmentEventID          pulid.ID              `json:"shipmentEventId"`
	ServiceFailureID         pulid.ID              `json:"serviceFailureId"`
	SourceMessageID          pulid.ID              `json:"sourceMessageId"`
	TransactionSet           edi.TransactionSet    `json:"transactionSet"`
	Direction                edi.DocumentDirection `json:"direction"`
	Payload                  *edi.DocumentPayload  `json:"payload"`
}

type GenerateEDIDocumentRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID pulid.ID              `json:"partnerDocumentProfileId"`
	EDIPartnerID             pulid.ID              `json:"ediPartnerId"`
	ShipmentID               pulid.ID              `json:"shipmentId"`
	TransferID               pulid.ID              `json:"transferId"`
	InvoiceID                pulid.ID              `json:"invoiceId"`
	ShipmentEventID          pulid.ID              `json:"shipmentEventId"`
	ServiceFailureID         pulid.ID              `json:"serviceFailureId"`
	SourceMessageID          pulid.ID              `json:"sourceMessageId"`
	TransactionSet           edi.TransactionSet    `json:"transactionSet"`
	Direction                edi.DocumentDirection `json:"direction"`
	Payload                  *edi.DocumentPayload  `json:"payload"`
	GeneratedByID            pulid.ID              `json:"-"`
	DisableDeliveryQueue     bool                  `json:"-"`
}

type ServiceFailure214LifecycleRequest struct {
	TenantInfo       pagination.TenantInfo          `json:"-"`
	ServiceFailureID pulid.ID                       `json:"serviceFailureId"`
	ShipmentID       pulid.ID                       `json:"shipmentId"`
	Trigger          ServiceFailureEDITrigger       `json:"trigger"`
	PreviousStatus   servicefailure.Status          `json:"previousStatus"`
	NewStatus        servicefailure.Status          `json:"newStatus"`
	GeneratedByID    pulid.ID                       `json:"-"`
	ServiceFailure   *servicefailure.ServiceFailure `json:"-"`
}

type ServiceFailure214LifecycleResult struct {
	Trigger                  ServiceFailureEDITrigger `json:"trigger"`
	Action                   ServiceFailureEDIAction  `json:"action"`
	MessageID                pulid.ID                 `json:"messageId,omitempty"`
	SkippedReason            string                   `json:"skippedReason,omitempty"`
	EDIPartnerID             pulid.ID                 `json:"ediPartnerId,omitempty"`
	PartnerDocumentProfileID pulid.ID                 `json:"partnerDocumentProfileId,omitempty"`
	Mandatory                bool                     `json:"mandatory"`
	Diagnostics              []edix12.Diagnostic      `json:"diagnostics"`
}

type ServiceFailure214Status = repositories.ServiceFailure214Status

type EDIService interface {
	BuildShipmentStatusPayloadForServiceFailure(
		ctx context.Context,
		req *BuildServiceFailureEDIPayloadRequest,
	) (*ServiceFailureEDIPayloadResult, error)
	PreviewDocument(ctx context.Context, req *PreviewEDIDocumentRequest) (*EDIDocumentPreview, error)
	GenerateDocument(ctx context.Context, req *GenerateEDIDocumentRequest) (*edi.EDIMessage, error)
	PreviewServiceFailure214ForLifecycle(
		ctx context.Context,
		req *ServiceFailure214LifecycleRequest,
	) (*ServiceFailure214LifecycleResult, error)
	GenerateServiceFailure214ForLifecycle(
		ctx context.Context,
		req *ServiceFailure214LifecycleRequest,
	) (*ServiceFailure214LifecycleResult, error)
	GetServiceFailure214Status(
		ctx context.Context,
		req repositories.GetServiceFailure214StatusRequest,
	) (*ServiceFailure214Status, error)
}

type EDIDocumentPreview struct {
	RawX12                   string                         `json:"rawX12"`
	SegmentCount             int64                          `json:"segmentCount"`
	X12Version               string                         `json:"x12Version"`
	InterchangeControlNumber string                         `json:"interchangeControlNumber"`
	GroupControlNumber       string                         `json:"groupControlNumber"`
	TransactionControlNumber string                         `json:"transactionControlNumber"`
	Diagnostics              []edix12.Diagnostic            `json:"diagnostics"`
	Profile                  *edi.EDIPartnerDocumentProfile `json:"profile,omitempty"`
	TemplateVersion          *edi.EDITemplateVersion        `json:"templateVersion,omitempty"`
}
