package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TransportFileKind string

const (
	TransportFileSchedulePDF = TransportFileKind("SchedulePdf")
	TransportFileScheduleCSV = TransportFileKind("ScheduleCsv")
	TransportFilePacket      = TransportFileKind("Packet")
)

type TransportFile struct {
	Kind        TransportFileKind `json:"kind"`
	Name        string            `json:"name"`
	ContentType string            `json:"contentType"`
	Content     []byte            `json:"-"`
	InvoiceID   pulid.ID          `json:"invoiceId,omitempty"`
}

type TransportMessage struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

type MessageRenderer func(part, parts int, attachments []string) (*TransportMessage, error)

type TransportSendRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	Submission     *factoring.Submission `json:"-"`
	Transport      *factoring.Transport  `json:"-"`
	Secret         string                `json:"-"`
	Files          []TransportFile       `json:"-"`
	Render         MessageRenderer       `json:"-"`
	IdempotencyKey string                `json:"-"`
}

type TransportSendResult struct {
	Reference string `json:"reference"`
	Parts     int    `json:"parts"`
}

type TransportTestRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Transport  *factoring.Transport  `json:"-"`
	Secret     string                `json:"-"`
}

type FactoringTransport interface {
	Method() factoring.TransportMethod
	Send(ctx context.Context, req *TransportSendRequest) (*TransportSendResult, error)
	Test(ctx context.Context, req *TransportTestRequest) error
}
