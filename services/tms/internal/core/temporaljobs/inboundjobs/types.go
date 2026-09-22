// Package inboundjobs drives a staged message through to a decision.
//
// The work is a workflow rather than part of the webhook because the webhook
// has to answer the provider quickly or be retried, and reading a message takes
// a model call. Staging and deciding are separated so a slow decision never
// costs the message.
package inboundjobs

import "github.com/emoss08/trenova/internal/core/services/inboundmessageservice"

// The payloads are the service's own, re-exported so the workflow reads
// naturally without the two packages importing each other.

type ProcessInboundMessagePayload = inboundmessageservice.ProcessInboundMessagePayload

type ProcessInboundMessageResult = inboundmessageservice.ProcessInboundMessageResult

type FailInboundMessagePayload = inboundmessageservice.FailInboundMessagePayload

type AttachmentRef = inboundmessageservice.AttachmentRef

type ListInboundAttachmentsResult = inboundmessageservice.ListInboundAttachmentsResult

type RecordInboundAttachmentPayload = inboundmessageservice.RecordInboundAttachmentPayload

type PollInboundAttachmentPayload = inboundmessageservice.PollInboundAttachmentPayload

type AttachmentExtractionState = inboundmessageservice.AttachmentExtractionState

// InboundMessageRetentionResult is what one retention run removed, and which
// tenants it could not finish, so a failure is visible without failing the rest.
type InboundMessageRetentionResult struct {
	Deleted int      `json:"deleted"`
	Failed  []string `json:"failed,omitempty"`
}
