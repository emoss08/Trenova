package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
)

var (
	// ErrInboundContentRejected is a provider refusing to return a message for
	// a reason another attempt will not change: the key is wrong or lacks
	// permission, or the message does not exist under it.
	ErrInboundContentRejected = errors.New("the provider refused to return the message")
	// ErrInboundContentUnavailable is a provider that could not answer just
	// now: a network failure, a rate limit or a server error.
	ErrInboundContentUnavailable = errors.New("the provider could not return the message just now")
)

// InboundContentRequest names one message at a provider and the key to read it
// with.
type InboundContentRequest struct {
	Provider          inboundmessage.Provider
	APIKey            string
	ProviderMessageID string
}

// InboundContent is the part of a message a metadata-only webhook leaves out.
type InboundContent struct {
	Text        string
	HTML        string
	MessageID   string
	InReplyTo   string
	References  []string
	Attachments []InboundContentAttachment
}

// InboundContentAttachment is one file, with its bytes or the reason there are
// none. A file too large to accept is a refusal on its own row, not a failure
// of the whole message.
type InboundContentAttachment struct {
	ProviderID  string
	FileName    string
	ContentType string
	Content     []byte
	FailureText string
}

// InboundContentFetcher reads a received message's body, headers and
// attachments from the provider that received it.
type InboundContentFetcher interface {
	Fetch(ctx context.Context, req *InboundContentRequest) (*InboundContent, error)
}
