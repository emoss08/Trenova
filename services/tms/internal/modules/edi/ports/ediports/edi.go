package ediports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrSendDeferred is GenerateDocument's answer when the partner's flow keeps a message to send
// later with others: nothing was written yet, and nothing failed.
var ErrSendDeferred = errors.New(
	"the partner's flow keeps this message to send later with others",
)

const ServiceFailureEDIActionSkipped servicefailureports.ServiceFailureEDIAction = "skipped"

const ServiceFailureEDIActionGenerated servicefailureports.ServiceFailureEDIAction = "generated"

const ServiceFailureEDIActionBlocked servicefailureports.ServiceFailureEDIAction = "blocked"

const ServiceFailureEDIActionDuplicate servicefailureports.ServiceFailureEDIAction = "duplicate"

const ServiceFailureEDIActionQueued servicefailureports.ServiceFailureEDIAction = "queued"

type PreviewEDIDocumentRequest struct {
	TenantInfo       pagination.TenantInfo         `json:"-"`
	Map              string                        `json:"map"`
	EDIPartnerID     pulid.ID                      `json:"ediPartnerId"`
	ShipmentID       pulid.ID                      `json:"shipmentId"`
	TransferID       pulid.ID                      `json:"transferId"`
	InvoiceID        pulid.ID                      `json:"invoiceId"`
	ShipmentEventID  pulid.ID                      `json:"shipmentEventId"`
	ServiceFailureID pulid.ID                      `json:"serviceFailureId"`
	SourceMessageID  pulid.ID                      `json:"sourceMessageId"`
	TransactionSet   edidocument.TransactionSet    `json:"transactionSet"`
	Direction        edidocument.DocumentDirection `json:"direction"`
	Payload          *edipayload.DocumentPayload   `json:"payload"`
	CarrierSCAC      string                        `json:"carrierScac"`
}

type GenerateEDIDocumentRequest struct {
	TenantInfo                    pagination.TenantInfo         `json:"-"`
	Map                           string                        `json:"map"`
	EDIPartnerID                  pulid.ID                      `json:"ediPartnerId"`
	ShipmentID                    pulid.ID                      `json:"shipmentId"`
	TransferID                    pulid.ID                      `json:"transferId"`
	InvoiceID                     pulid.ID                      `json:"invoiceId"`
	ShipmentEventID               pulid.ID                      `json:"shipmentEventId"`
	ServiceFailureID              pulid.ID                      `json:"serviceFailureId"`
	SourceMessageID               pulid.ID                      `json:"sourceMessageId"`
	TransactionSet                edidocument.TransactionSet    `json:"transactionSet"`
	Direction                     edidocument.DocumentDirection `json:"direction"`
	Payload                       *edipayload.DocumentPayload   `json:"payload"`
	CarrierSCAC                   string                        `json:"-"`
	GeneratedByID                 pulid.ID                      `json:"-"`
	DisableDeliveryQueue          bool                          `json:"-"`
	SuppressTenderRecipientUpsert bool                          `json:"-"`
	IdempotencyKey                string                        `json:"-"`
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

type ServiceFailure214Status = edimessageports.ServiceFailure214StatusRow

type ServiceFailurePayloadBuilder interface {
	BuildShipmentStatusPayloadForServiceFailure(
		ctx context.Context,
		req *servicefailureports.BuildServiceFailureEDIPayloadRequest,
	) (*servicefailureports.ServiceFailureEDIPayloadResult, error)
}

type EDIDocumentService interface {
	PreviewDocument(
		ctx context.Context,
		req *PreviewEDIDocumentRequest,
	) (*EDIDocumentPreview, error)
	GenerateDocument(
		ctx context.Context,
		req *GenerateEDIDocumentRequest,
	) (*edimessage.EDIMessage, error)
}

type ServiceFailure214Service interface {
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
		req edimessageports.GetServiceFailure214StatusRequest,
	) (*ServiceFailure214Status, error)
}

type EDIDocumentPreview struct {
	PartnerName              string                   `json:"partnerName"`
	RawX12                   string                   `json:"rawX12"`
	SegmentCount             int64                    `json:"segmentCount"`
	X12Version               string                   `json:"x12Version"`
	InterchangeControlNumber string                   `json:"interchangeControlNumber"`
	GroupControlNumber       string                   `json:"groupControlNumber"`
	TransactionControlNumber string                   `json:"transactionControlNumber"`
	Diagnostics              []edidocument.Diagnostic `json:"diagnostics"`
	LadingVersionID          pulid.ID                 `json:"ladingVersionId"`
	Map                      string                   `json:"map"`
	Document                 jsonutils.RawJSON        `json:"document"`
	Lineage                  []lading.Lineage         `json:"lineage"`
	Outcomes                 []lading.Outcome         `json:"outcomes"`
}
