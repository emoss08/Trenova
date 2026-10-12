package webhooks

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
)

const (
	maxNameLength          = 255
	maxURLLength           = 2047
	maxCustomHeaders       = 5
	maxCustomHeaderKeySize = 100
)

func ValidateCreateRequest(req CreateRequest) error {
	if err := validateName(req.Name); err != nil {
		return err
	}
	if err := validateURL(req.Url); err != nil {
		return err
	}
	if req.Version != nil && !req.Version.Valid() {
		return fmt.Errorf("%w: %q", ErrWebhookVersionInvalid, *req.Version)
	}
	if req.EventTypes != nil {
		for _, eventType := range *req.EventTypes {
			if !eventType.Valid() {
				return fmt.Errorf("%w: %q", ErrWebhookEventInvalid, eventType)
			}
		}
	}
	return validateCustomHeaders(req.CustomHeaders)
}

func ValidateUpdateRequest(req UpdateRequest) error {
	if req.Name != nil {
		if err := validateName(*req.Name); err != nil {
			return err
		}
	}
	if req.Url != nil {
		if err := validateURL(*req.Url); err != nil {
			return err
		}
	}
	if req.Version != nil && !req.Version.Valid() {
		return fmt.Errorf("%w: %q", ErrWebhookVersionInvalid, *req.Version)
	}
	return validateCustomHeaders(req.CustomHeaders)
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrWebhookNameRequired
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return ErrWebhookNameTooLong
	}
	return nil
}

func validateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrWebhookURLRequired
	}
	if len(raw) > maxURLLength {
		return ErrWebhookURLTooLong
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") {
		return ErrWebhookURLInvalid
	}
	return nil
}

func validateCustomHeaders(headers *[]samsaraspec.CustomHeadersObjectRequestBody) error {
	if headers == nil {
		return nil
	}
	if len(*headers) > maxCustomHeaders {
		return ErrCustomHeadersTooMany
	}
	for _, header := range *headers {
		if strings.TrimSpace(header.Key) == "" || header.Value == "" {
			return ErrCustomHeaderInvalid
		}
		if utf8.RuneCountInString(header.Key) > maxCustomHeaderKeySize {
			return ErrCustomHeaderKeyTooLong
		}
	}
	return nil
}
