package edimessageview

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type InspectX12Request struct {
	TenantInfo     pagination.TenantInfo            `json:"-"`
	RawX12         string                           `json:"rawX12"`
	TransactionSet edidocument.TransactionSet       `json:"transactionSet"`
	X12Version     string                           `json:"x12Version"`
	Envelope       *edidocument.X12EnvelopeSettings `json:"envelope"`
	Diagnostics    []edidocument.Diagnostic         `json:"diagnostics"`
}

type EDIMessageInspection struct {
	Message    *edimessage.EDIMessage         `json:"message"`
	Inspection edix12inspect.InspectX12Result `json:"inspection"`
	Provenance EDIInspectionProvenance        `json:"provenance"`
}

type EDIInspectionProvenance struct {
	MessageID       pulid.ID `json:"messageId"`
	LadingVersionID pulid.ID `json:"ladingVersionId"`
	FlowInstanceID  pulid.ID `json:"flowInstanceId"`
	GeneratedAt     int64    `json:"generatedAt"`
	GeneratedByID   pulid.ID `json:"generatedById"`
}
