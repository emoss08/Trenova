package inboundmessageservice

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/shared/timeutils"
)

// parsePayload turns a provider's body into the one shape the staging path
// understands. Which provider a mailbox listens to is stored on the mailbox, so
// a delivery cannot be parsed as the wrong one by sending it to the wrong URL.
func parsePayload(
	provider inboundmessage.Provider,
	body []byte,
) (*providerMessage, error) {
	switch provider {
	case inboundmessage.ProviderResend:
		return parseResend(body)
	case inboundmessage.ProviderPostmark:
		return parsePostmark(body)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, provider)
	}
}

type resendPayload struct {
	Type string `json:"type"`
	Data struct {
		ID        string   `json:"email_id"`
		MessageID string   `json:"message_id"`
		From      string   `json:"from"`
		To        []string `json:"to"`
		Cc        []string `json:"cc"`
		Subject   string   `json:"subject"`
		Text      string   `json:"text"`
		HTML      string   `json:"html"`
		CreatedAt string   `json:"created_at"`
		Headers   []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		Attachments []struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			Content     string `json:"content"`
		} `json:"attachments"`
	} `json:"data"`
}

func parseResend(body []byte) (*providerMessage, error) {
	var payload resendPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedPayload, err)
	}

	// Resend posts every webhook kind to the same endpoint. Anything that is
	// not a received email is not an error — it is simply not ours, and saying
	// so lets the caller answer 200 rather than make the provider retry.
	if payload.Type != resendInboundEvent {
		return nil, ErrNotAnInboundMessage
	}
	if payload.Data.ID == "" {
		return nil, fmt.Errorf("%w: the delivery carries no email id", ErrMalformedPayload)
	}

	message := &providerMessage{
		ProviderMessageID: payload.Data.ID,
		MessageID:         payload.Data.MessageID,
		Subject:           payload.Data.Subject,
		TextBody:          payload.Data.Text,
		HTMLBody:          payload.Data.HTML,
		ReceivedAt:        timeutils.NowUnix(),
	}

	message.FromName, message.FromAddress = splitMailbox(payload.Data.From)
	for _, to := range payload.Data.To {
		if address := normalizeAddress(to); address != "" {
			message.ToAddresses = append(message.ToAddresses, address)
		}
	}
	for _, cc := range payload.Data.Cc {
		if address := normalizeAddress(cc); address != "" {
			message.CcAddresses = append(message.CcAddresses, address)
		}
	}

	for _, header := range payload.Data.Headers {
		switch strings.ToLower(header.Name) {
		case "in-reply-to":
			message.InReplyTo = strings.TrimSpace(header.Value)
		case "references":
			message.References = strings.Fields(header.Value)
		case "message-id":
			if message.MessageID == "" {
				message.MessageID = strings.TrimSpace(header.Value)
			}
		}
	}

	for _, attachment := range payload.Data.Attachments {
		content, err := base64.StdEncoding.DecodeString(attachment.Content)
		if err != nil {
			// One unreadable attachment does not discard the message. The
			// message is still what the sender believes was received, and the
			// file is recorded as failed rather than silently dropped.
			content = nil
		}
		message.Attachments = append(message.Attachments, providerAttachment{
			FileName:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     content,
		})
	}

	return message, nil
}

const resendInboundEvent = "email.received"

type postmarkPayload struct {
	MessageID string `json:"MessageID"`
	From      string `json:"From"`
	FromName  string `json:"FromName"`
	To        string `json:"To"`
	Cc        string `json:"Cc"`
	Subject   string `json:"Subject"`
	TextBody  string `json:"TextBody"`
	HTMLBody  string `json:"HtmlBody"`
	SpamScore string `json:"SpamScore"`
	Headers   []struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	} `json:"Headers"`
	Attachments []struct {
		Name          string `json:"Name"`
		ContentType   string `json:"ContentType"`
		Content       string `json:"Content"`
		ContentLength int64  `json:"ContentLength"`
	} `json:"Attachments"`
}

func parsePostmark(body []byte) (*providerMessage, error) {
	var payload postmarkPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedPayload, err)
	}
	if payload.MessageID == "" {
		return nil, fmt.Errorf("%w: the delivery carries no message id", ErrMalformedPayload)
	}

	message := &providerMessage{
		ProviderMessageID: payload.MessageID,
		FromAddress:       normalizeAddress(payload.From),
		FromName:          strings.TrimSpace(payload.FromName),
		ToAddresses:       splitAddressList(payload.To),
		CcAddresses:       splitAddressList(payload.Cc),
		Subject:           payload.Subject,
		TextBody:          payload.TextBody,
		HTMLBody:          payload.HTMLBody,
		ReceivedAt:        timeutils.NowUnix(),
		SpamScore:         parseSpamScore(payload.SpamScore),
	}

	for _, header := range payload.Headers {
		switch strings.ToLower(header.Name) {
		case "in-reply-to":
			message.InReplyTo = strings.TrimSpace(header.Value)
		case "references":
			message.References = strings.Fields(header.Value)
		case "message-id":
			message.MessageID = strings.TrimSpace(header.Value)
		}
	}

	for _, attachment := range payload.Attachments {
		content, err := base64.StdEncoding.DecodeString(attachment.Content)
		if err != nil {
			content = nil
		}
		message.Attachments = append(message.Attachments, providerAttachment{
			FileName:    attachment.Name,
			ContentType: attachment.ContentType,
			Content:     content,
		})
	}

	return message, nil
}

// parseSpamScore reads Postmark's score, which arrives as a string and is
// absent on a mailbox with filtering off. An unreadable score is zero rather
// than an error: it is a hint for a person, not a gate.
func parseSpamScore(raw string) float64 {
	var score float64
	if _, err := fmt.Sscanf(strings.TrimSpace(raw), "%f", &score); err != nil {
		return 0
	}

	return score
}

// splitMailbox separates `Jane Doe <jane@acme.com>` into its two parts. A bare
// address has no display name, which is the common case for machine senders.
func splitMailbox(raw string) (name, address string) {
	raw = strings.TrimSpace(raw)

	open := strings.LastIndex(raw, "<")
	closed := strings.LastIndex(raw, ">")
	if open < 0 || closed < open {
		return "", normalizeAddress(raw)
	}

	name = strings.TrimSpace(strings.Trim(raw[:open], `" `))

	return name, normalizeAddress(raw[open+1 : closed])
}
