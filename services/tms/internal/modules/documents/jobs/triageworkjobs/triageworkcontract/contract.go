package triageworkcontract

import (
	"strconv"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DraftManyWorkflowName   = "TriageDraftManyWorkflow"
	SplitFilingWorkflowName = "TriageSplitFilingWorkflow"
	MaxDraftBatch           = 200
)

func DraftManyWorkflowID(batchID pulid.ID) string {
	return "triage-draft-many/" + batchID.String()
}

func SplitFilingWorkflowID(attachmentID pulid.ID, filedAt int64) string {
	return "triage-split-filing/" + attachmentID.String() + "/" + strconv.FormatInt(filedAt, 10)
}

type DraftManyPayload struct {
	temporaltype.BasePayload

	UserID     pulid.ID   `json:"userId"`
	MessageIDs []pulid.ID `json:"messageIds"`
}

type DraftOnePayload struct {
	temporaltype.BasePayload

	UserID    pulid.ID `json:"userId"`
	MessageID pulid.ID `json:"messageId"`
}

type DraftManyResult struct {
	Drafted int `json:"drafted"`
	Failed  int `json:"failed"`
}

type SplitRange struct {
	FromPage       int      `json:"fromPage"`
	ToPage         int      `json:"toPage"`
	DocumentTypeID pulid.ID `json:"documentTypeId"`
	DocumentType   string   `json:"documentType"`
	SessionID      pulid.ID `json:"sessionId"`
}

type SplitFilingPayload struct {
	temporaltype.BasePayload

	UserID       pulid.ID     `json:"userId"`
	MessageID    pulid.ID     `json:"messageId"`
	AttachmentID pulid.ID     `json:"attachmentId"`
	ShipmentID   pulid.ID     `json:"shipmentId"`
	FiledAt      int64        `json:"filedAt"`
	Ranges       []SplitRange `json:"ranges"`
}

type SplitDocument struct {
	SplitRange

	DocumentID pulid.ID `json:"documentId"`
}

type RecordSplitPayload struct {
	SplitFilingPayload

	Documents []SplitDocument `json:"documents"`
	Failed    int             `json:"failed"`
}
