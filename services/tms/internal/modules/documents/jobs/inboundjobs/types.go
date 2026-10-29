// Package inboundjobs drives a staged message through to a decision.
//
// The work is a workflow rather than part of the webhook because the webhook
// has to answer the provider quickly or be retried, and reading a message takes
// a model call. Staging and deciding are separated so a slow decision never
// costs the message.
package inboundjobs

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundintake"
)

// The payloads are the service's own, re-exported so the workflow reads
// naturally without the two packages importing each other.

type ProcessInboundMessagePayload = inboundintake.ProcessInboundMessagePayload

type ProcessInboundMessageResult = inboundintake.ProcessInboundMessageResult

type FailInboundMessagePayload = inboundintake.FailInboundMessagePayload

type AttachmentRef = inboundintake.AttachmentRef

type ListInboundAttachmentsResult = inboundintake.ListInboundAttachmentsResult

type RecordInboundAttachmentPayload = inboundintake.RecordInboundAttachmentPayload

type PollInboundAttachmentPayload = inboundintake.PollInboundAttachmentPayload

type AttachmentExtractionState = inboundintake.AttachmentExtractionState

// FetchInboundContentResult says whether the message can go on to be read.
// Readable is false when its content could not be fetched and it has already
// been sent to review saying why.
type FetchInboundContentResult struct {
	Readable bool `json:"readable"`
}

// InboundMessageRetentionResult is what one retention run removed, and which
// tenants it could not finish, so a failure is visible without failing the rest.
type InboundMessageRetentionResult struct {
	Deleted int      `json:"deleted"`
	Failed  []string `json:"failed,omitempty"`
}
