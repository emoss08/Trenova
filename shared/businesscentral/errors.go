package businesscentral

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	CodeRecordNotFound       = "Internal_RecordNotFound"
	CodeEntityWithSameKey    = "Internal_EntityWithSameKeyExists"
	CodeEntityChanged        = "Request_EntityChanged"
	CodeInvalidToken         = "BadRequest_InvalidToken"
	CodeDialogException      = "Application_DialogException"
	codePrefixAuthentication = "Authentication_"
	codePrefixAuthorization  = "Authorization_"
	codePrefixApplication    = "Application_"
	codePrefixBadRequest     = "BadRequest_"
	oauthInvalidGrant        = "invalid_grant"
	oauthInvalidClient       = "invalid_client"
	maxPlainErrorMessageSize = 512
	redactedValue            = "REDACTED"
	minRedactedSecretLength  = 4
	requestIDHeader          = "request-id"
	correlationIDHeader      = "ms-correlation-x"
	retryAfterHeader         = "Retry-After"
	postingDatePhrase        = "posting date"
	allowedPostingPhrase     = "allowed posting"
	rangeOfAllowedPhrase     = "range of allowed"
)

var (
	ErrAccessTokenRequired  = errors.New("businesscentral: access token is required")
	ErrClientIDRequired     = errors.New("businesscentral: client id is required")
	ErrClientSecretRequired = errors.New("businesscentral: client secret is required")
	ErrRedirectURLRequired  = errors.New("businesscentral: redirect url is required")
	ErrStateRequired        = errors.New("businesscentral: state is required")
	ErrCodeRequired         = errors.New("businesscentral: authorization code is required")
	ErrTokenRequired        = errors.New("businesscentral: token is required")
	ErrInvalidGrant         = errors.New(
		"businesscentral: the authorization was revoked or has expired",
	)
	ErrInvalidClient = errors.New(
		"businesscentral: the app's client id or secret was not accepted",
	)
	ErrMalformedToken    = errors.New("businesscentral: the access token is not a readable JWT")
	ErrTenantClaim       = errors.New("businesscentral: the access token carries no tenant id")
	ErrUnexpectedPayload = errors.New("businesscentral: unexpected response payload")
	ErrForeignNextLink   = errors.New(
		"businesscentral: a next page link points outside the Business Central API",
	)
	ErrTooManyResults = errors.New(
		"businesscentral: the read matched more records than can be collected",
	)
	ErrIDRequired            = errors.New("businesscentral: an id is required")
	ErrInvalidID             = errors.New("businesscentral: an id must be a GUID")
	ErrTooManyIDs            = errors.New("businesscentral: at most 50 ids may be read at once")
	ErrInvalidEnvironment    = errors.New("businesscentral: invalid environment name")
	ErrInvalidCompanyRef     = errors.New("businesscentral: invalid company reference")
	ErrInvalidSubscriptionID = errors.New(
		"businesscentral: a subscription id is 1 to 100 letters, digits or hyphens",
	)
	ErrInvalidETag   = errors.New("businesscentral: an etag must be printable ASCII")
	ErrInvalidFilter = errors.New(
		"businesscentral: a filter value is empty, too long or contains control characters",
	)
	ErrInvalidDate        = errors.New("businesscentral: a date must be YYYY-MM-DD")
	ErrDateRequired       = errors.New("businesscentral: a date is required")
	ErrUnknownKind        = errors.New("businesscentral: unknown kind")
	ErrUnknownType        = errors.New("businesscentral: unknown type")
	ErrUnsupportedAction  = errors.New("businesscentral: the document does not support this action")
	ErrPartyRequired      = errors.New("businesscentral: a customer or vendor is required")
	ErrLinesRequired      = errors.New("businesscentral: a line needs exactly one item or account")
	ErrQuantityInvalid    = errors.New("businesscentral: a line quantity must be positive")
	ErrPriceInvalid       = errors.New("businesscentral: a unit price cannot be negative")
	ErrAmountInvalid      = errors.New("businesscentral: the payment amount is not allowed")
	ErrNameRequired       = errors.New("businesscentral: a display name is required")
	ErrJournalCodeInvalid = errors.New("businesscentral: a journal code is 1 to 10 characters")
	ErrFieldTooLong       = errors.New(
		"businesscentral: a field exceeds the length Business Central accepts",
	)
	ErrFieldNotSupported = errors.New(
		"businesscentral: the document kind does not take this field",
	)
	ErrInvalidText         = errors.New("businesscentral: text contains control characters")
	ErrCurrencyCodeInvalid = errors.New("businesscentral: a currency code is up to 10 characters")
	ErrNotificationURL     = errors.New(
		"businesscentral: a notification url must be absolute https",
	)
	ErrResourceRequired     = errors.New("businesscentral: a subscription resource is required")
	ErrClientStateTooLong   = errors.New("businesscentral: client state exceeds 2048 characters")
	ErrTooManyNotifications = errors.New(
		"businesscentral: the notification batch is larger than accepted",
	)
)

type APIError struct {
	Status     int
	Code       string
	Message    string
	RetryAfter time.Duration
	RequestID  string
	cause      *restx.APIError
}

func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("businesscentral api error: status ")
	b.WriteString(strconv.Itoa(e.Status))
	if e.Code != "" {
		b.WriteString(" ")
		b.WriteString(e.Code)
	}
	message := e.Message
	if message == "" {
		message = http.StatusText(e.Status)
	}
	if message != "" {
		b.WriteString(": ")
		b.WriteString(message)
	}
	if e.RequestID != "" {
		b.WriteString(" [request ")
		b.WriteString(e.RequestID)
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
	ErrorCodes  []int
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return "businesscentral oauth error " + e.Code + ": " + e.Description
	}
	return "businesscentral oauth error " + e.Code
}

func (e *OAuthError) Is(target error) bool {
	switch {
	case errors.Is(target, ErrInvalidClient):
		return e.invalidClient()
	case errors.Is(target, ErrInvalidGrant):
		return !e.invalidClient() &&
			(e.Code == oauthInvalidGrant || e.hasAnyCode(invalidGrantCodes))
	default:
		return false
	}
}

func (e *OAuthError) invalidClient() bool {
	return e.Code == oauthInvalidClient || e.hasAnyCode(invalidClientCodes)
}

func (e *OAuthError) hasAnyCode(codes []int) bool {
	for _, code := range e.ErrorCodes {
		if slices.Contains(codes, code) {
			return true
		}
	}
	return false
}

var (
	invalidGrantCodes  = []int{70000, 70008, 700082, 50173}
	invalidClientCodes = []int{7000215, 700016, 7000222}
)

type apiErrorBody struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
}

func decodeAPIError(status int, body []byte, header http.Header) error {
	apiErr := &APIError{
		Status: status,
		cause:  restx.DecodeAPIError(status, body, header),
	}
	if header != nil {
		if retryAfter, ok := restx.ParseRetryAfter(header.Get(retryAfterHeader)); ok {
			apiErr.RetryAfter = retryAfter
		}
		apiErr.RequestID = firstNonBlank(
			header.Get(requestIDHeader),
			header.Get(correlationIDHeader),
		)
	}

	var parsed apiErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil && parsed.Error != nil {
		apiErr.Code = strings.TrimSpace(parsed.Error.Code)
		apiErr.Message = strings.TrimSpace(parsed.Error.Message)
	} else {
		apiErr.Message = plainMessage(body)
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}

func decodeOAuthError(status int, body []byte, header http.Header) error {
	var parsed oauthErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return &OAuthError{
			Status:      status,
			Code:        parsed.Error,
			Description: parsed.ErrorDescription,
			ErrorCodes:  parsed.ErrorCodes,
		}
	}
	return restx.DecodeAPIError(status, body, header)
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

func redactAPIError(err error, redact func(string) string) {
	if apiErr, ok := apiErrorOf(err); ok {
		apiErr.Message = redact(apiErr.Message)
		apiErr.Code = redact(apiErr.Code)
	}
}

func redactSecrets(err error, secrets ...string) {
	replace := func(value string) string {
		for _, secret := range secrets {
			if len(secret) >= minRedactedSecretLength {
				value = strings.ReplaceAll(value, secret, redactedValue)
			}
		}
		return value
	}
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		oauthErr.Description = replace(oauthErr.Description)
	}
	var restErr *restx.APIError
	if errors.As(err, &restErr) {
		restErr.Message = replace(restErr.Message)
		restErr.Body = []byte(replace(string(restErr.Body)))
	}
}

func apiErrorOf(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		return apiErr, true
	}
	return nil, false
}

func statusOf(err error) int {
	if apiErr, ok := apiErrorOf(err); ok {
		return apiErr.Status
	}
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		return oauthErr.Status
	}
	return restx.StatusCode(err)
}

func codeOf(err error) string {
	if apiErr, ok := apiErrorOf(err); ok {
		return apiErr.Code
	}
	return ""
}

func IsAuth(err error) bool {
	return statusOf(err) == http.StatusUnauthorized ||
		strings.HasPrefix(codeOf(err), codePrefixAuthentication)
}

func IsForbidden(err error) bool {
	return statusOf(err) == http.StatusForbidden ||
		strings.HasPrefix(codeOf(err), codePrefixAuthorization)
}

func IsNotFound(err error) bool {
	return statusOf(err) == http.StatusNotFound || codeOf(err) == CodeRecordNotFound
}

func IsDuplicate(err error) bool {
	return codeOf(err) == CodeEntityWithSameKey
}

func IsConflict(err error) bool {
	switch statusOf(err) {
	case http.StatusConflict, http.StatusPreconditionFailed:
		return true
	}
	code := codeOf(err)
	return code == CodeEntityChanged || code == CodeInvalidToken
}

func IsPostingDate(err error) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok || !strings.HasPrefix(apiErr.Code, codePrefixApplication) {
		return false
	}
	message := strings.ToLower(apiErr.Message)
	return strings.Contains(message, postingDatePhrase) ||
		strings.Contains(message, allowedPostingPhrase) ||
		strings.Contains(message, rangeOfAllowedPhrase)
}

func IsValidation(err error) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok {
		return false
	}
	return apiErr.Status == http.StatusBadRequest ||
		strings.HasPrefix(apiErr.Code, codePrefixApplication) ||
		strings.HasPrefix(apiErr.Code, codePrefixBadRequest)
}

func IsRateLimited(err error) bool {
	return statusOf(err) == http.StatusTooManyRequests || restx.IsRateLimited(err)
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
	var transport *restx.TransportError
	if errors.As(err, &transport) {
		return true
	}
	switch statusOf(err) {
	case http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func RetryAfter(err error) time.Duration {
	if apiErr, ok := apiErrorOf(err); ok && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}
	return restx.RetryAfterOf(err)
}
