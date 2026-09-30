package xero

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const SignatureHeader = "x-xero-signature"

type WebhookEvent struct {
	ResourceURL   string
	ResourceID    string
	EventDate     time.Time
	EventType     string
	EventCategory string
	TenantID      string
	TenantType    string
}

type WebhookPayload struct {
	Events             []WebhookEvent
	FirstEventSequence int64
	LastEventSequence  int64
	Entropy            string
}

type wireWebhookEvent struct {
	ResourceURL   string   `json:"resourceUrl"`
	ResourceID    string   `json:"resourceId"`
	EventDateUTC  wireTime `json:"eventDateUtc"`
	EventType     string   `json:"eventType"`
	EventCategory string   `json:"eventCategory"`
	TenantID      string   `json:"tenantId"`
	TenantType    string   `json:"tenantType"`
}

type wireWebhookPayload struct {
	Events             []wireWebhookEvent `json:"events"`
	FirstEventSequence int64              `json:"firstEventSequence"`
	LastEventSequence  int64              `json:"lastEventSequence"`
	Entropy            string             `json:"entropy"`
}

func VerifySignature(key string, body []byte, signature string) error {
	if strings.TrimSpace(key) == "" {
		return ErrWebhookKeyRequired
	}
	provided, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(provided) == 0 {
		return ErrMissingSignature
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}

func ParseWebhook(body []byte) (*WebhookPayload, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, ErrUnexpectedPayload
	}
	var wire wireWebhookPayload
	if err := sonic.Unmarshal(trimmed, &wire); err != nil {
		return nil, ErrUnexpectedPayload
	}

	payload := &WebhookPayload{
		Events:             make([]WebhookEvent, 0, len(wire.Events)),
		FirstEventSequence: wire.FirstEventSequence,
		LastEventSequence:  wire.LastEventSequence,
		Entropy:            wire.Entropy,
	}
	for idx := range wire.Events {
		event := &wire.Events[idx]
		payload.Events = append(payload.Events, WebhookEvent{
			ResourceURL:   event.ResourceURL,
			ResourceID:    event.ResourceID,
			EventDate:     event.EventDateUTC.time(),
			EventType:     event.EventType,
			EventCategory: event.EventCategory,
			TenantID:      strings.TrimSpace(event.TenantID),
			TenantType:    event.TenantType,
		})
	}
	return payload, nil
}

func (p *WebhookPayload) TenantIDs() []string {
	if p == nil {
		return []string{}
	}
	seen := make(map[string]struct{}, len(p.Events))
	tenants := make([]string, 0, len(p.Events))
	for idx := range p.Events {
		tenant := p.Events[idx].TenantID
		if tenant == "" {
			continue
		}
		if _, dup := seen[tenant]; dup {
			continue
		}
		seen[tenant] = struct{}{}
		tenants = append(tenants, tenant)
	}
	return tenants
}
