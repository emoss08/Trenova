package ediports

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ServiceFailureEDIActionSkipped servicefailureports.ServiceFailureEDIAction = "skipped"

const ServiceFailureEDIActionGenerated servicefailureports.ServiceFailureEDIAction = "generated"

const ServiceFailureEDIActionBlocked servicefailureports.ServiceFailureEDIAction = "blocked"

const ServiceFailureEDIActionDuplicate servicefailureports.ServiceFailureEDIAction = "duplicate"

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
	CarrierSCAC              string                `json:"carrierScac"`
}

type GenerateEDIDocumentRequest struct {
	TenantInfo                    pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID      pulid.ID              `json:"partnerDocumentProfileId"`
	EDIPartnerID                  pulid.ID              `json:"ediPartnerId"`
	ShipmentID                    pulid.ID              `json:"shipmentId"`
	TransferID                    pulid.ID              `json:"transferId"`
	InvoiceID                     pulid.ID              `json:"invoiceId"`
	ShipmentEventID               pulid.ID              `json:"shipmentEventId"`
	ServiceFailureID              pulid.ID              `json:"serviceFailureId"`
	SourceMessageID               pulid.ID              `json:"sourceMessageId"`
	TransactionSet                edi.TransactionSet    `json:"transactionSet"`
	Direction                     edi.DocumentDirection `json:"direction"`
	Payload                       *edi.DocumentPayload  `json:"payload"`
	CarrierSCAC                   string                `json:"-"`
	GeneratedByID                 pulid.ID              `json:"-"`
	DisableDeliveryQueue          bool                  `json:"-"`
	SuppressTenderRecipientUpsert bool                  `json:"-"`
	IdempotencyKey                string                `json:"-"`
}

type ServiceFailure214LifecycleRequest struct {
	TenantInfo       pagination.TenantInfo                        `json:"-"`
	ServiceFailureID pulid.ID                                     `json:"serviceFailureId"`
	ShipmentID       pulid.ID                                     `json:"shipmentId"`
	Trigger          servicefailureports.ServiceFailureEDITrigger `json:"trigger"`
	PreviousStatus   servicefailure.Status                        `json:"previousStatus"`
	NewStatus        servicefailure.Status                        `json:"newStatus"`
	GeneratedByID    pulid.ID                                     `json:"-"`
	ServiceFailure   *servicefailure.ServiceFailure               `json:"-"`
}

type ServiceFailure214Status = ServiceFailure214StatusRow

type EDIService interface {
	BuildShipmentStatusPayloadForServiceFailure(
		ctx context.Context,
		req *servicefailureports.BuildServiceFailureEDIPayloadRequest,
	) (*servicefailureports.ServiceFailureEDIPayloadResult, error)
	PreviewDocument(
		ctx context.Context,
		req *PreviewEDIDocumentRequest,
	) (*EDIDocumentPreview, error)
	GenerateDocument(ctx context.Context, req *GenerateEDIDocumentRequest) (*edi.EDIMessage, error)
	PreviewServiceFailure214ForLifecycle(
		ctx context.Context,
		req *ServiceFailure214LifecycleRequest,
	) (*servicefailureports.ServiceFailure214LifecycleResult, error)
	GenerateServiceFailure214ForLifecycle(
		ctx context.Context,
		req *ServiceFailure214LifecycleRequest,
	) (*servicefailureports.ServiceFailure214LifecycleResult, error)
	GetServiceFailure214Status(
		ctx context.Context,
		req GetServiceFailure214StatusRequest,
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
