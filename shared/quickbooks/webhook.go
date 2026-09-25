package quickbooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/bytedance/sonic"
)

const SignatureHeader = "intuit-signature"

type cloudEvent struct {
	AccountID string `json:"intuitaccountid"`
}

type legacyEnvelope struct {
	EventNotifications []struct {
		RealmID string `json:"realmId"`
	} `json:"eventNotifications"`
}

func VerifySignature(verifierToken, signature string, body []byte) error {
	if strings.TrimSpace(verifierToken) == "" {
		return ErrVerifierRequired
	}
	provided, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(provided) == 0 {
		return ErrMissingSignature
	}

	mac := hmac.New(sha256.New, []byte(verifierToken))
	mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}

func ParseRealmIDs(body []byte) ([]string, error) {
	trimmed := strings.TrimSpace(string(body))
	seen := make(map[string]struct{}, 4)
	realms := make([]string, 0, 4)
	add := func(realm string) {
		realm = strings.TrimSpace(realm)
		if realm == "" {
			return
		}
		if _, ok := seen[realm]; ok {
			return
		}
		seen[realm] = struct{}{}
		realms = append(realms, realm)
	}

	switch {
	case strings.HasPrefix(trimmed, "["):
		var events []cloudEvent
		if err := sonic.UnmarshalString(trimmed, &events); err != nil {
			return nil, ErrUnexpectedPayload
		}
		for idx := range events {
			add(events[idx].AccountID)
		}
	case strings.HasPrefix(trimmed, "{"):
		var single cloudEvent
		if err := sonic.UnmarshalString(trimmed, &single); err != nil {
			return nil, ErrUnexpectedPayload
		}
		add(single.AccountID)
		var legacy legacyEnvelope
		if err := sonic.UnmarshalString(trimmed, &legacy); err != nil {
			return nil, ErrUnexpectedPayload
		}
		for idx := range legacy.EventNotifications {
			add(legacy.EventNotifications[idx].RealmID)
		}
	default:
		return nil, ErrUnexpectedPayload
	}

	return realms, nil
}
