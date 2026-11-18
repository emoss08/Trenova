package emailservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

var ErrMalformedProviderEvent = errors.New("the email provider's event could not be read")

type ProviderEvent struct {
	Event             *email.Event
	ProviderMessageID string
	SuppressionReason email.SuppressionReason
}

func (p *ProviderEvent) For(tenantInfo pagination.TenantInfo) HandleProviderEventParams {
	p.Event.OrganizationID = tenantInfo.OrgID
	p.Event.BusinessUnitID = tenantInfo.BuID
	return HandleProviderEventParams{
		TenantInfo:        tenantInfo,
		Event:             p.Event,
		ProviderMessageID: p.ProviderMessageID,
		SuppressionReason: p.SuppressionReason,
	}
}

type RelayedSystemEvent struct {
	ID       string
	Provider email.Provider
	Payload  []byte
}

type resendWebhookPayload struct {
	Type string `json:"type"`
	Data struct {
		ID      string              `json:"id"`
		EmailID string              `json:"email_id"`
		To      jsonflex.StringList `json:"to"`
		Bounce  struct {
			Type string `json:"type"`
		} `json:"bounce"`
	} `json:"data"`
	CreatedAt string `json:"created_at"`
}

type postmarkWebhookPayload struct {
	RecordType  string `json:"RecordType"`
	ID          any    `json:"ID"`
	MessageID   string `json:"MessageID"`
	Recipient   string `json:"Recipient"`
	Email       string `json:"Email"`
	Type        string `json:"Type"`
	ReceivedAt  string `json:"ReceivedAt"`
	DeliveredAt string `json:"DeliveredAt"`
	BouncedAt   string `json:"BouncedAt"`
}

func ParseProviderEvent(
	provider email.Provider,
	body []byte,
	deliveryID string,
) (*ProviderEvent, bool, error) {
	switch provider {
	case email.ProviderResend:
		return ParseResendEvent(body, deliveryID)
	case email.ProviderPostmark:
		return ParsePostmarkEvent(body)
	default:
		return nil, false, fmt.Errorf(
			"%w: unknown provider %q",
			ErrMalformedProviderEvent,
			provider,
		)
	}
}

func ParseResendEvent(body []byte, deliveryID string) (*ProviderEvent, bool, error) {
	var payload resendWebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("%w: %w", ErrMalformedProviderEvent, err)
	}
	eventType, known := resendEventType(&payload)
	if !known {
		return nil, false, nil
	}
	providerEventID := deliveryID
	if providerEventID == "" {
		providerEventID = payload.Data.ID
	}
	event, err := newProviderEvent(body, &email.Event{
		Provider:        email.ProviderResend,
		ProviderEventID: providerEventID,
		Type:            eventType,
		Recipient:       resendRecipient(&payload),
	})
	if err != nil {
		return nil, false, err
	}
	return &ProviderEvent{
		Event:             event,
		ProviderMessageID: payload.Data.EmailID,
		SuppressionReason: resendSuppressionReason(eventType),
	}, true, nil
}

func ParsePostmarkEvent(body []byte) (*ProviderEvent, bool, error) {
	var payload postmarkWebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("%w: %w", ErrMalformedProviderEvent, err)
	}
	eventType, known := postmarkEventType(&payload)
	if !known {
		return nil, false, nil
	}
	event, err := newProviderEvent(body, &email.Event{
		Provider:        email.ProviderPostmark,
		ProviderEventID: postmarkProviderEventID(&payload),
		Type:            eventType,
		Recipient:       postmarkRecipient(&payload),
	})
	if err != nil {
		return nil, false, err
	}
	return &ProviderEvent{
		Event:             event,
		ProviderMessageID: payload.MessageID,
		SuppressionReason: postmarkSuppressionReason(&payload),
	}, true, nil
}

func newProviderEvent(body []byte, event *email.Event) (*email.Event, error) {
	event.OccurredAt = timeutils.NowUnix()
	event.Raw = map[string]any{}
	if err := sonic.Unmarshal(body, &event.Raw); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedProviderEvent, err)
	}
	return event, nil
}

func (s *Service) HandleRelayedSystemEvent(
	ctx context.Context,
	relayed *RelayedSystemEvent,
) error {
	parsed, reason := readRelayedSystemEvent(relayed)
	if reason == "" {
		tenantInfo, found, err := s.TenantOfSystemMessage(
			ctx,
			relayed.Provider,
			parsed.ProviderMessageID,
		)
		if err != nil {
			return err
		}
		if found {
			return s.HandleProviderEvent(ctx, parsed.For(tenantInfo))
		}
		reason = dropNotSentHere
	}
	s.l.Info(
		"a relayed system email event was dropped",
		zap.String("eventId", relayed.ID),
		zap.String("provider", string(relayed.Provider)),
		zap.String("reason", reason),
	)
	return nil
}

const (
	dropUnreadable  = "unreadable"
	dropNotABounce  = "not_a_bounce_or_complaint"
	dropNotSentHere = "not_sent_by_this_instance"
)

func readRelayedSystemEvent(relayed *RelayedSystemEvent) (event *ProviderEvent, drop string) {
	parsed, known, err := ParseProviderEvent(relayed.Provider, relayed.Payload, relayed.ID)
	switch {
	case err != nil:
		return nil, dropUnreadable
	case !known || (parsed.Event.Type != email.EventTypeBounced &&
		parsed.Event.Type != email.EventTypeComplained):
		return nil, dropNotABounce
	default:
		return parsed, ""
	}
}

func resendEventType(payload *resendWebhookPayload) (email.EventType, bool) {
	switch payload.Type {
	case "email.sent":
		return email.EventTypeSent, true
	case "email.delivered":
		return email.EventTypeDelivered, true
	case "email.opened":
		return email.EventTypeOpened, true
	case "email.clicked":
		return email.EventTypeClicked, true
	case "email.bounced":
		if strings.EqualFold(payload.Data.Bounce.Type, "Transient") {
			return "", false
		}
		return email.EventTypeBounced, true
	case "email.complained":
		return email.EventTypeComplained, true
	case "email.failed":
		return email.EventTypeFailed, true
	default:
		return "", false
	}
}

func resendRecipient(payload *resendWebhookPayload) string {
	if len(payload.Data.To) != 1 {
		return ""
	}
	return payload.Data.To.First()
}

func resendSuppressionReason(eventType email.EventType) email.SuppressionReason {
	switch eventType {
	case email.EventTypeBounced:
		return email.SuppressionReasonHardBounce
	case email.EventTypeComplained:
		return email.SuppressionReasonComplaint
	default:
		return ""
	}
}

var postmarkHardBounceTypes = map[string]struct{}{
	"HardBounce":          {},
	"BadEmailAddress":     {},
	"ManuallyDeactivated": {},
	"SpamNotification":    {},
}

func postmarkIsHardBounce(payload *postmarkWebhookPayload) bool {
	_, ok := postmarkHardBounceTypes[payload.Type]
	return ok
}

func postmarkEventType(payload *postmarkWebhookPayload) (email.EventType, bool) {
	switch payload.RecordType {
	case "Delivery":
		return email.EventTypeDelivered, true
	case "Open":
		return email.EventTypeOpened, true
	case "Click":
		return email.EventTypeClicked, true
	case "Bounce":
		if !postmarkIsHardBounce(payload) {
			return "", false
		}
		return email.EventTypeBounced, true
	case "SpamComplaint":
		return email.EventTypeComplained, true
	default:
		return "", false
	}
}

func postmarkSuppressionReason(payload *postmarkWebhookPayload) email.SuppressionReason {
	switch {
	case payload.RecordType == "Bounce" && postmarkIsHardBounce(payload):
		return email.SuppressionReasonHardBounce
	case payload.RecordType == "SpamComplaint":
		return email.SuppressionReasonComplaint
	default:
		return ""
	}
}

func postmarkProviderEventID(payload *postmarkWebhookPayload) string {
	if id := postmarkStringValue(payload.ID); id != "" {
		return id
	}
	timestamp := payload.DeliveredAt
	if timestamp == "" {
		timestamp = payload.BouncedAt
	}
	if timestamp == "" {
		timestamp = payload.ReceivedAt
	}
	return strings.Join(
		[]string{payload.RecordType, payload.MessageID, postmarkRecipient(payload), timestamp},
		":",
	)
}

func postmarkRecipient(payload *postmarkWebhookPayload) string {
	if payload.Recipient != "" {
		return payload.Recipient
	}
	return payload.Email
}

func postmarkStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}
