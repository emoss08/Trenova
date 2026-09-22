// Package inboundmessageservice receives mail forwarded to a monitored address
// and turns it into a row somebody can act on.
//
// The endpoint it serves is unauthenticated by necessity — a mail provider
// cannot hold a session — so everything about who a delivery belongs to is
// derived from the token in its URL, and everything about whether to believe it
// is derived from its signature.
package inboundmessageservice

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
)

// providerMessage is what every provider's payload is flattened into, so the
// staging path is written once rather than per provider.
type providerMessage struct {
	ProviderMessageID string
	MessageID         string
	InReplyTo         string
	References        []string
	FromAddress       string
	FromName          string
	ToAddresses       []string
	CcAddresses       []string
	Subject           string
	TextBody          string
	HTMLBody          string
	ReceivedAt        int64
	SpamScore         float64
	Attachments       []providerAttachment
}

type providerAttachment struct {
	FileName    string
	ContentType string
	// Content is the decoded bytes. Providers send these base64-encoded inside
	// the webhook body, which is why the body cap is measured in tens of
	// megabytes rather than the one the outbound event hooks use.
	Content []byte
}

// boundedText trims a body to what the row will hold.
//
// The whole message goes to object storage; this is the part that has to be
// searchable and has to still be there when storage is not. Cutting on a rune
// boundary matters because the column is text, not bytes, and a half-written
// rune would make the row unreadable rather than merely truncated.
func boundedText(body string) string {
	if len(body) <= inboundmessage.MaxBodyBytes {
		return body
	}

	cut := body[:inboundmessage.MaxBodyBytes]

	// Trimming back to a whole rune means dropping a trailing continuation byte
	// *and* a lead byte whose continuations were cut off — the second is the
	// one that is easy to miss, and it leaves the column holding text no reader
	// can decode.
	for len(cut) > 0 {
		if r, size := utf8.DecodeLastRuneInString(cut); r != utf8.RuneError || size > 1 {
			break
		}
		cut = cut[:len(cut)-1]
	}

	return cut
}

// normalizeAddress lowercases and trims, because a provider will hand back
// whatever the sender's client wrote and matching on it is case-insensitive
// everywhere else.
func normalizeAddress(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

// splitAddressList reads the comma-separated form providers use for To and Cc.
func splitAddressList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	addresses := make([]string, 0, len(parts))
	for _, part := range parts {
		if address := normalizeAddress(part); address != "" {
			addresses = append(addresses, address)
		}
	}
	if len(addresses) == 0 {
		return nil
	}

	return addresses
}
