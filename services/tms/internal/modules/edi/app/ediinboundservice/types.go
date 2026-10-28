package ediinboundservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/pulid"
)

type PollMailboxRequest struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type PollMailboxResult struct {
	ProfileID     pulid.ID   `json:"profileId"`
	StagedFileIDs []pulid.ID `json:"stagedFileIds"`
	SkippedFiles  int        `json:"skippedFiles"`
}

type ProcessInboundFileRequest struct {
	FileID     pulid.ID              `json:"fileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Reprocess  bool                  `json:"reprocess"`
}

type PollableProfile struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type parsedInterchange struct {
	inspection          *edix12inspect.InspectX12Result
	reading             *x12.Reading
	controlNumber       string
	senderQualifier     string
	senderID            string
	receiverQualifier   string
	receiverID          string
	functionalGroupID   string
	groupControlNumber  string
	applicationSender   string
	applicationReceiver string
	version             string
	usage               string
	groups              []parsedGroup
	transactions        []parsedTransaction
}

type parsedGroup struct {
	group               *x12.Group
	functionalGroupID   string
	controlNumber       string
	applicationSender   string
	applicationReceiver string
	version             string
}

type parsedTransaction struct {
	set                edidocument.TransactionSet
	controlNumber      string
	groupControlNumber string
	functionalGroupID  string
	segments           []edix12inspect.X12Segment
	raw                string
	version            string
	group              int
	status             x12.TransactionStatus
	findings           []x12.Finding
}

func (t *parsedTransaction) rejected() bool {
	return t.status == x12.TransactionRejected
}

type transactionOutcome struct {
	message  *edimessage.EDIMessage
	warnings []string
	err      error
	tendered bool
}

type acknowledgmentEntry struct {
	originalTransactionSet    string
	originalControlNumber     string
	acknowledgmentCode        string
	diagnostics               []edipayload.AcknowledgmentDiagnostic
	groupAcknowledgmentCode   string
	originalFunctionalGroupID string
	originalGroupControl      string
	acceptedCount             int64
	receivedCount             int64
	includedCount             int64
}

type tenderResponseDetails struct {
	scac            string
	shipmentRef     string
	reservationCode string
	remarks         string
	references      []string
}

type shipmentStatusDetails struct {
	controlNumber string
	referenceID   string
	shipmentRef   string
	statusCode    string
	reasonCode    string
	eventAt       int64
	references    []string
	events        []edipayload.ShipmentStatusEvent
}

const stopSequenceQualifier = "QN"

const inboundDefaultMappingKey = "DEFAULT"
