package inboundmessageservice

import (
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

// The workflow's payloads live with the service rather than with the jobs
// package, because the service starts the workflow and the jobs package runs
// it. Putting them in the jobs package would make the two import each other.
// edijobs re-exports ediservice's payloads the same way.

type ProcessInboundMessagePayload struct {
	temporaltype.BasePayload

	MessageID pulid.ID `json:"messageId"`
}

type ProcessInboundMessageResult struct {
	MessageID      pulid.ID                      `json:"messageId"`
	Status         inboundmessage.Status         `json:"status"`
	Classification inboundmessage.Classification `json:"classification"`
	Handled        bool                          `json:"handled"`
	Matched        bool                          `json:"matched"`
}

// FailInboundMessagePayload records why the pipeline gave up, in words the
// inbox can show.
type FailInboundMessagePayload struct {
	temporaltype.BasePayload

	MessageID pulid.ID `json:"messageId"`
	Code      string   `json:"code"`
	Reason    string   `json:"reason"`
}

// ListInboundAttachmentsResult is the work the attachment pass has left.
type ListInboundAttachmentsResult struct {
	Attachments []AttachmentRef `json:"attachments"`
}

// RecordInboundAttachmentPayload ties a finalized upload to its row, or says
// why there is no document to tie.
type RecordInboundAttachmentPayload struct {
	temporaltype.BasePayload

	Attachment  AttachmentRef `json:"attachment"`
	DocumentID  pulid.ID      `json:"documentId"`
	FailureText string        `json:"failureText"`
}

// PollInboundAttachmentPayload asks how far the document pipeline has got.
type PollInboundAttachmentPayload struct {
	temporaltype.BasePayload

	Attachment AttachmentRef `json:"attachment"`
	// GiveUp turns the poll into a verdict: past the deadline, a document still
	// at Pending is a failure rather than something still being worked on.
	GiveUp bool `json:"giveUp"`
}
