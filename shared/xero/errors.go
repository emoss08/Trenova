package xero

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	TypeValidation           = "ValidationException"
	TypeAuthUnsuccessful     = "AuthenticationUnsuccessful"
	TypeInsufficientScope    = "InsufficientScope"
	RateLimitMinute          = "minute"
	RateLimitDay             = "day"
	RateLimitAppMinute       = "appminute"
	RateLimitConcurrent      = "concurrent"
	correlationHeader        = "Xero-Correlation-Id"
	rateLimitProblemHeader   = "X-Rate-Limit-Problem"
	wwwAuthenticateHeader    = "WWW-Authenticate"
	maxPlainErrorMessageSize = 512
)

var (
	ErrTenantIDRequired     = errors.New("xero: tenant id is required")
	ErrAccessTokenRequired  = errors.New("xero: access token is required")
	ErrClientIDRequired     = errors.New("xero: client id is required")
	ErrClientSecretRequired = errors.New("xero: client secret is required")
	ErrRedirectURLRequired  = errors.New("xero: redirect url is required")
	ErrStateRequired        = errors.New("xero: state is required")
	ErrCodeRequired         = errors.New("xero: authorization code is required")
	ErrTokenRequired        = errors.New("xero: token is required")
	ErrInvalidGrant         = errors.New("xero: the authorization was revoked or has expired")
	ErrInvalidClient        = errors.New("xero: the app's client id or secret was not accepted")
	ErrMalformedToken       = errors.New("xero: the access token is not a readable JWT")
	ErrAuthEventMissing     = errors.New(
		"xero: the access token carries no authentication event id",
	)
	ErrUnexpectedPayload = errors.New("xero: unexpected response payload")
	ErrIDRequired        = errors.New("xero: an id is required")
	ErrInvalidID         = errors.New("xero: an id must be a GUID")
	ErrTooManyIDs        = errors.New("xero: at most 100 ids may be read at once")
	ErrInvalidPage       = errors.New("xero: a page number starts at 1")
	ErrIdempotencyKey    = errors.New(
		"xero: an idempotency key of 1 to 128 characters is required",
	)
	ErrReportDateRequired = errors.New("xero: a report needs a date")
	ErrEmptyFilter        = errors.New("xero: a search needs at least one criterion")
	ErrInvalidFilter      = errors.New(
		"xero: a search criterion contains characters Xero cannot match",
	)
	ErrWebhookKeyRequired = errors.New("xero: webhook key is required")
	ErrMissingSignature   = errors.New("xero: webhook signature is missing")
	ErrInvalidSignature   = errors.New("xero: webhook signature does not match")
	ErrInvalidDate        = errors.New("xero: unrecognised date")
	ErrDateRequired       = errors.New("xero: a date is required")
	ErrContactRequired    = errors.New("xero: a contact is required")
	ErrLinesRequired      = errors.New("xero: a document needs at least one line")
	ErrLineDescription    = errors.New("xero: every line needs a description or an item code")
	ErrUnknownType        = errors.New("xero: unknown document type")
	ErrUnknownStatus      = errors.New("xero: unknown document status")
	ErrUnknownAmountTypes = errors.New("xero: unknown line amount types")
	ErrAmountNotPositive  = errors.New("xero: an amount must be greater than zero")
	ErrNegativeRate       = errors.New("xero: a currency rate cannot be negative")
	ErrTargetRequired     = errors.New(
		"xero: a payment applies to exactly one invoice or credit note",
	)
	ErrAccountRequired = errors.New(
		"xero: a payment needs exactly one account id or account code",
	)
	ErrPaymentsRequired    = errors.New("xero: a batch payment needs at least one payment")
	ErrItemCodeInvalid     = errors.New("xero: an item code is 1 to 30 characters")
	ErrItemNameInvalid     = errors.New("xero: an item name is 1 to 50 characters")
	ErrContactNameInvalid  = errors.New("xero: a contact name is 1 to 255 characters")
	ErrFieldTooLong        = errors.New("xero: a field exceeds the length Xero accepts")
	ErrCurrencyCodeInvalid = errors.New("xero: a currency code is three letters")
)

type APIError struct {
	Status             int
	Type               string
	Message            string
	ValidationMessages []string
	RateLimitProblem   string
	RetryAfter         time.Duration
	CorrelationID      string
	cause              *restx.APIError
}

func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("xero api error: status ")
	b.WriteString(strconv.Itoa(e.Status))
	if e.Type != "" {
		b.WriteString(" ")
		b.WriteString(e.Type)
	}
	message := e.Message
	if message == "" {
		message = http.StatusText(e.Status)
	}
	if message != "" {
		b.WriteString(": ")
		b.WriteString(message)
	}
	for _, detail := range e.ValidationMessages {
		b.WriteString("; ")
		b.WriteString(detail)
	}
	if e.RateLimitProblem != "" {
		b.WriteString(" (rate limit: ")
		b.WriteString(e.RateLimitProblem)
		b.WriteString(")")
	}
	if e.CorrelationID != "" {
		b.WriteString(" [correlation ")
		b.WriteString(e.CorrelationID)
		b.WriteString("]")
	}
	return b.String()
}

func (e *APIError) Unwrap() error {
	if e.cause == nil {
		return nil
	}
	return e.cause
}

type OAuthError struct {
	Status      int
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return "xero oauth error " + e.Code + ": " + e.Description
	}
	return "xero oauth error " + e.Code
}

func (e *OAuthError) Is(target error) bool {
	switch {
	case errors.Is(target, ErrInvalidGrant):
		return e.Code == "invalid_grant"
	case errors.Is(target, ErrInvalidClient):
		return e.Code == "invalid_client" || e.Code == "unauthorized_client"
	default:
		return false
	}
}

type validationError struct {
	Message string `json:"Message"`
}

type errorElement struct {
	ValidationErrors []validationError `json:"ValidationErrors"`
	LineItems        []struct {
		ValidationErrors []validationError `json:"ValidationErrors"`
	} `json:"LineItems"`
}

type apiErrorBody struct {
	Type     *string        `json:"Type"`
	Message  string         `json:"Message"`
	Title    string         `json:"Title"`
	Detail   string         `json:"Detail"`
	Elements []errorElement `json:"Elements"`
}

type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func decodeAPIError(status int, body []byte, header http.Header) error {
	apiErr := &APIError{
		Status: status,
		cause:  restx.DecodeAPIError(status, body, header),
	}
	if header != nil {
		apiErr.CorrelationID = strings.TrimSpace(header.Get(correlationHeader))
		apiErr.RateLimitProblem = strings.ToLower(
			strings.TrimSpace(header.Get(rateLimitProblemHeader)),
		)
		if retryAfter, ok := restx.ParseRetryAfter(header.Get("Retry-After")); ok {
			apiErr.RetryAfter = retryAfter
		}
	}

	var parsed apiErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil {
		if parsed.Type != nil {
			apiErr.Type = strings.TrimSpace(*parsed.Type)
		}
		apiErr.Message = firstNonBlank(parsed.Message, parsed.Detail, parsed.Title)
		apiErr.ValidationMessages = validationMessages(parsed.Elements)
	} else {
		apiErr.Message = plainMessage(body)
	}

	if status == http.StatusUnauthorized && apiErr.Type == "" && header != nil {
		challenge := strings.ToLower(header.Get(wwwAuthenticateHeader))
		if strings.Contains(challenge, "insufficent_scope") ||
			strings.Contains(challenge, "insufficient_scope") {
			apiErr.Type = TypeInsufficientScope
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}

func validationMessages(elements []errorElement) []string {
	messages := make([]string, 0, len(elements))
	seen := make(map[string]struct{}, len(elements))
	add := func(items []validationError) {
		for idx := range items {
			message := strings.TrimSpace(items[idx].Message)
			if message == "" {
				continue
			}
			if _, dup := seen[message]; dup {
				continue
			}
			seen[message] = struct{}{}
			messages = append(messages, message)
		}
	}
	for idx := range elements {
		add(elements[idx].ValidationErrors)
		for line := range elements[idx].LineItems {
			add(elements[idx].LineItems[line].ValidationErrors)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return messages
}

func plainMessage(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" || strings.HasPrefix(text, "<") {
		return ""
	}
	if len(text) <= maxPlainErrorMessageSize {
		return text
	}
	cut := maxPlainErrorMessageSize
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func decodeOAuthError(status int, body []byte, header http.Header) error {
	var parsed oauthErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return &OAuthError{
			Status:      status,
			Code:        parsed.Error,
			Description: parsed.ErrorDescription,
		}
	}
	return restx.DecodeAPIError(status, body, header)
}

func apiErrorOf(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		return apiErr, true
	}
	return nil, false
}

func hasStatus(err error, status int) bool {
	if apiErr, ok := apiErrorOf(err); ok {
		return apiErr.Status == status
	}
	return restx.IsStatus(err, status)
}

func validationContains(err error, needles ...string) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok || apiErr.Status != http.StatusBadRequest {
		return false
	}
	for _, message := range apiErr.ValidationMessages {
		lower := strings.ToLower(message)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				return true
			}
		}
	}
	return false
}

func IsAuth(err error) bool {
	return hasStatus(err, http.StatusUnauthorized)
}

func IsInsufficientScope(err error) bool {
	apiErr, ok := apiErrorOf(err)
	return ok && apiErr.Status == http.StatusUnauthorized && apiErr.Type == TypeInsufficientScope
}

func IsForbidden(err error) bool {
	return hasStatus(err, http.StatusForbidden)
}

func IsNotFound(err error) bool {
	return hasStatus(err, http.StatusNotFound)
}

func IsRateLimited(err error) bool {
	return hasStatus(err, http.StatusTooManyRequests) || restx.IsRateLimited(err)
}

func IsValidation(err error) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok || apiErr.Status != http.StatusBadRequest {
		return false
	}
	return apiErr.Type == TypeValidation || len(apiErr.ValidationMessages) > 0
}

func IsDuplicateNumber(err error) bool {
	return validationContains(err,
		"must be unique",
		"already exists",
		"has already been used",
		"is already assigned to another",
	)
}

func IsLockDate(err error) bool {
	return validationContains(err, "lock date", "locked")
}

func IsOrganisationOffline(err error) bool {
	return hasStatus(err, http.StatusServiceUnavailable)
}

func IsInvalidGrant(err error) bool {
	return errors.Is(err, ErrInvalidGrant)
}

func IsInvalidClient(err error) bool {
	return errors.Is(err, ErrInvalidClient)
}

func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if IsRateLimited(err) {
		return true
	}
	var transport *restx.TransportError
	if errors.As(err, &transport) {
		return true
	}
	if apiErr, ok := apiErrorOf(err); ok {
		return apiErr.Status >= http.StatusInternalServerError
	}
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		return oauthErr.Status >= http.StatusInternalServerError
	}
	return restx.StatusCode(err) >= http.StatusInternalServerError
}
