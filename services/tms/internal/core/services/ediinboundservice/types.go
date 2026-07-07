package ediinboundservice

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12inspect"
	"github.com/emoss08/trenova/pkg/pagination"
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
	inspection          edix12inspect.InspectX12Result
	controlNumber       string
	senderQualifier     string
	senderID            string
	receiverQualifier   string
	receiverID          string
	functionalGroupID   string
	groupControlNumber  string
	applicationSender   string
	applicationReceiver string
	transactions        []parsedTransaction
}

type parsedTransaction struct {
	set                edi.TransactionSet
	controlNumber      string
	groupControlNumber string
	functionalGroupID  string
	segments           []edix12inspect.X12Segment
	raw                string
}

type transactionOutcome struct {
	message  *edi.EDIMessage
	warnings []string
	err      error
}

type acknowledgmentEntry struct {
	originalTransactionSet    string
	originalControlNumber     string
	acknowledgmentCode        string
	diagnostics               []edi.AcknowledgmentDiagnostic
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
}

type shipmentStatusDetails struct {
	referenceID string
	shipmentRef string
	statusCode  string
	reasonCode  string
	eventAt     int64
}

const inboundDefaultMappingKey = "DEFAULT"
