package businesscentral

import (
	"bytes"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	ValidationTokenParam     = "validationToken"
	ChangeTypeCreated        = "created"
	ChangeTypeUpdated        = "updated"
	ChangeTypeDeleted        = "deleted"
	ChangeTypeCollection     = "collection"
	MaxNotificationsPerBatch = 1000
	maxValidationTokenLength = 1024
)

type Notification struct {
	SubscriptionID     string
	ClientState        string
	ExpirationDateTime time.Time
	Resource           string
	ChangeType         string
	LastModified       time.Time
}

type wireNotification struct {
	SubscriptionID       string   `json:"subscriptionId"`
	ClientState          string   `json:"clientState"`
	ExpirationDateTime   wireTime `json:"expirationDateTime"`
	Resource             string   `json:"resource"`
	ChangeType           string   `json:"changeType"`
	LastModifiedDateTime wireTime `json:"lastModifiedDateTime"`
}

type notificationEnvelope struct {
	Value []wireNotification `json:"value"`
}

func ParseNotifications(body []byte) ([]Notification, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, ErrUnexpectedPayload
	}
	var wire notificationEnvelope
	if err := sonic.Unmarshal(trimmed, &wire); err != nil {
		return nil, ErrUnexpectedPayload
	}
	if len(wire.Value) > MaxNotificationsPerBatch {
		return nil, ErrTooManyNotifications
	}

	notifications := make([]Notification, 0, len(wire.Value))
	for idx := range wire.Value {
		item := &wire.Value[idx]
		notifications = append(notifications, Notification{
			SubscriptionID:     strings.TrimSpace(item.SubscriptionID),
			ClientState:        item.ClientState,
			ExpirationDateTime: item.ExpirationDateTime.time(),
			Resource:           item.Resource,
			ChangeType:         strings.ToLower(strings.TrimSpace(item.ChangeType)),
			LastModified:       item.LastModifiedDateTime.time(),
		})
	}
	return notifications, nil
}

func ValidationToken(raw string) (string, bool) {
	if raw == "" || len(raw) > maxValidationTokenLength {
		return "", false
	}
	for idx := range len(raw) {
		if raw[idx] < 0x21 || raw[idx] > 0x7e {
			return "", false
		}
	}
	return raw, true
}
