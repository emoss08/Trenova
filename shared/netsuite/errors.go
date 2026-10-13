package netsuite

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	CodeNonexistentID            = "NONEXISTENT_ID"
	CodeConcurrencyLimitExceeded = "CONCURRENCY_LIMIT_EXCEEDED"
	CodeInvalidLogin             = "INVALID_LOGIN"
	CodeIdempotencyError         = "IDEMPOTENCY_ERROR"
	CodeUserError                = "USER_ERROR"
	CodeInvalidContent           = "INVALID_CONTENT"
	CodeInvalidRequest           = "INVALID_REQUEST"
	CodeInsufficientPermission   = "INSUFFICIENT_PERMISSION"
	CodeUnexpectedError          = "UNEXPECTED_ERROR"
	oauthInvalidGrant            = "invalid_grant"
	oauthInvalidClient           = "invalid_client"
	maxPlainErrorMessageSize     = 512
	redactedValue                = "REDACTED"
	minRedactedSecretLength      = 4
	retryAfterHeader             = "Retry-After"
	permissionPhrase             = "permission"
)

var (
	ErrAccessTokenRequired  = errors.New("netsuite: access token is required")
	ErrInvalidAccountID     = errors.New("netsuite: invalid account id")
	ErrInvalidSubsidiaryID  = errors.New("netsuite: a subsidiary id must be a positive integer")
	ErrInvalidCompanyRef    = errors.New("netsuite: invalid company reference")
	ErrClientIDRequired     = errors.New("netsuite: client id is required")
	ErrClientSecretRequired = errors.New("netsuite: client secret is required")
	ErrRedirectURLRequired  = errors.New("netsuite: redirect url is required")
	ErrInvalidState         = errors.New(
		"netsuite: state must be 22 to 1024 printable characters",
	)
	ErrCodeRequired       = errors.New("netsuite: authorization code is required")
	ErrInvalidVerifier    = errors.New("netsuite: a PKCE code verifier is 43 to 128 characters")
	ErrInvalidChallenge   = errors.New("netsuite: a PKCE S256 code challenge is required")
	ErrTokenRequired      = errors.New("netsuite: token is required")
	ErrInvalidCertificate = errors.New(
		"netsuite: a certificate id is 1 to 100 printable characters",
	)
	ErrInvalidPrivateKey = errors.New("netsuite: the private key is not an ECDSA P-256 key")
	ErrSubjectRequired   = errors.New(
		"netsuite: a certificate subject of up to 64 characters is required",
	)
	ErrInvalidValidity = errors.New("netsuite: a certificate is valid for at most two years")
	ErrInvalidGrant    = errors.New("netsuite: the authorization was revoked or has expired")
	ErrInvalidClient   = errors.New(
		"netsuite: the integration's client id, secret or certificate was not accepted",
	)
	ErrUnexpectedPayload = errors.New("netsuite: unexpected response payload")
	ErrForeignLocation   = errors.New(
		"netsuite: a record location points outside the account's REST API",
	)
	ErrInvalidRecordType = errors.New("netsuite: invalid record type")
	ErrInvalidRecordID   = errors.New(
		"netsuite: a record id is an internal id or eid: and an external id",
	)
	ErrInvalidExternalID = errors.New(
		"netsuite: an external id is 1 to 255 letters, digits, underscores or hyphens",
	)
	ErrInvalidFieldName = errors.New("netsuite: invalid field or sublist name")
	ErrBodyRequired     = errors.New("netsuite: a record body is required")
	ErrOutRequired      = errors.New("netsuite: a destination for the response is required")
	ErrQueryRequired    = errors.New("netsuite: a SuiteQL query is required")
	ErrInvalidLimit     = errors.New("netsuite: a SuiteQL limit is at most 1000")
	ErrInvalidRows      = errors.New("netsuite: query rows must be a non-nil pointer to a slice")
	ErrInvalidQueryTime = errors.New(
		"netsuite: a query time must be YYYY-MM-DD HH24:MI:SS or a date YYYY-MM-DD",
	)
)

var (
	closedPeriodPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)not\s+(within|in)\s+an?\s+open\s+(accounting\s+)?period`),
		regexp.MustCompile(`(?i)date\s+range\s+of\s+your\s+accounting\s+period`),
		regexp.MustCompile(
			`(?i)(accounting\s+)?period\s+(is|has\s+been|was)\s+(closed|locked)`,
		),
		regexp.MustCompile(`(?i)\b(closed|locked)\s+(accounting\s+|posting\s+)?period`),
		regexp.MustCompile(`(?i)\ba/[rp]\s+(period|is\s+locked|locked)`),
		regexp.MustCompile(`(?i)\blocked\s+for\s+a/[rp]\b`),
		regexp.MustCompile(`(?i)\bperiod\b[^.]*\b(closed|locked)\b`),
	}
	duplicateExternalIDPattern = regexp.MustCompile(
		`(?i)(external\s*_?id|already\s+exists|\bunique\b|duplicate\s+key)`,
	)
	invalidReferencePattern  = regexp.MustCompile(`(?i)invalid\b.*\breference\s+key\b`)
	unsupportedTablePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(error\s+occurred:|^)\s*record\s+'[^']+'\s+was\s+not\s+found`),
		regexp.MustCompile(`(?i)invalid\s+or\s+unsupported\s+search`),
	}
)

type ErrorDetail struct {
	Detail string
	Code   string
	Path   string
}

type APIError struct {
	Status     int
	Type       string
	Title      string
	Details    []ErrorDetail
	RetryAfter time.Duration
	cause      *restx.APIError
}

func (e *APIError) Code() string {
	for idx := range e.Details {
		if e.Details[idx].Code != "" {
			return e.Details[idx].Code
		}
	}
	return ""
}

func (e *APIError) Detail() string {
	for idx := range e.Details {
		if e.Details[idx].Detail != "" {
			return e.Details[idx].Detail
		}
	}
	return ""
}

func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("netsuite api error: status ")
	b.WriteString(strconv.Itoa(e.Status))
	if code := e.Code(); code != "" {
		b.WriteString(" ")
		b.WriteString(code)
	}
	message := e.Detail()
	if message == "" {
		message = e.Title
	}
	if message == "" {
		message = http.StatusText(e.Status)
	}
	if message != "" {
		b.WriteString(": ")
		b.WriteString(message)
	}
	return b.String()
}

func (e *APIError) Unwrap() error {
	if e.cause == nil {
		return nil
	}
	return e.cause
}

func (e *APIError) hasCode(codes ...string) bool {
	for idx := range e.Details {
		for _, code := range codes {
			if strings.EqualFold(e.Details[idx].Code, code) {
				return true
			}
		}
	}
	return false
}

func (e *APIError) detailMatches(patterns ...*regexp.Regexp) bool {
	for idx := range e.Details {
		for _, pattern := range patterns {
			if pattern.MatchString(e.Details[idx].Detail) {
				return true
			}
		}
	}
	for _, pattern := range patterns {
		if pattern.MatchString(e.Title) {
			return true
		}
	}
	return false
}

func (e *APIError) detailContains(phrase string) bool {
	for idx := range e.Details {
		if strings.Contains(strings.ToLower(e.Details[idx].Detail), phrase) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(e.Title), phrase)
}

type OAuthError struct {
	Status      int
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return "netsuite oauth error " + e.Code + ": " + e.Description
	}
	return "netsuite oauth error " + e.Code
}

func (e *OAuthError) Is(target error) bool {
	switch {
	case errors.Is(target, ErrInvalidClient):
		return e.Code == oauthInvalidClient
	case errors.Is(target, ErrInvalidGrant):
		return e.Code == oauthInvalidGrant
	default:
		return false
	}
}

type errorBody struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Details []struct {
		Detail string `json:"detail"`
		Code   string `json:"o:errorCode"`
		Path   string `json:"o:errorPath"`
	} `json:"o:errorDetails"`
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
		if retryAfter, ok := restx.ParseRetryAfter(header.Get(retryAfterHeader)); ok {
			apiErr.RetryAfter = retryAfter
		}
	}

	var parsed errorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil &&
		(parsed.Title != "" || len(parsed.Details) > 0) {
		apiErr.Type = strings.TrimSpace(parsed.Type)
		apiErr.Title = strings.TrimSpace(parsed.Title)
		apiErr.Details = make([]ErrorDetail, 0, len(parsed.Details))
		for idx := range parsed.Details {
			apiErr.Details = append(apiErr.Details, ErrorDetail{
				Detail: strings.TrimSpace(parsed.Details[idx].Detail),
				Code:   strings.TrimSpace(parsed.Details[idx].Code),
				Path:   strings.TrimSpace(parsed.Details[idx].Path),
			})
		}
	} else {
		apiErr.Title = plainMessage(body)
	}
	if apiErr.Title == "" {
		apiErr.Title = http.StatusText(status)
	}
	return apiErr
}

func decodeOAuthError(status int, body []byte, header http.Header) error {
	var parsed oauthErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return &OAuthError{
			Status:      status,
			Code:        strings.TrimSpace(parsed.Error),
			Description: strings.TrimSpace(parsed.ErrorDescription),
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

func redactAPIError(err error, redact func(string) string) {
	apiErr, ok := apiErrorOf(err)
	if !ok {
		return
	}
	apiErr.Title = redact(apiErr.Title)
	for idx := range apiErr.Details {
		apiErr.Details[idx].Detail = redact(apiErr.Details[idx].Detail)
		apiErr.Details[idx].Path = redact(apiErr.Details[idx].Path)
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

func hasCode(err error, codes ...string) bool {
	apiErr, ok := apiErrorOf(err)
	return ok && apiErr.hasCode(codes...)
}

func IsNotFound(err error) bool {
	return statusOf(err) == http.StatusNotFound || hasCode(err, CodeNonexistentID)
}

func IsConcurrencyLimit(err error) bool {
	return statusOf(err) == http.StatusTooManyRequests ||
		hasCode(err, CodeConcurrencyLimitExceeded)
}

func IsRateLimited(err error) bool {
	return IsConcurrencyLimit(err) || restx.IsRateLimited(err)
}

func IsUnauthorized(err error) bool {
	return statusOf(err) == http.StatusUnauthorized || hasCode(err, CodeInvalidLogin)
}

func IsForbidden(err error) bool {
	return statusOf(err) == http.StatusForbidden
}

func IsIdempotencyConflict(err error) bool {
	return hasCode(err, CodeIdempotencyError)
}

func IsDuplicateExternalID(err error) bool {
	apiErr, ok := apiErrorOf(err)
	return ok && apiErr.hasCode(CodeUserError, CodeInvalidContent) &&
		apiErr.detailMatches(duplicateExternalIDPattern)
}

func IsClosedPeriod(err error) bool {
	apiErr, ok := apiErrorOf(err)
	return ok && apiErr.hasCode(CodeUserError) && apiErr.detailMatches(closedPeriodPatterns...)
}

func IsInvalidReference(err error) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok || !apiErr.hasCode(CodeInvalidContent) || IsDuplicateExternalID(err) {
		return false
	}
	if apiErr.detailMatches(invalidReferencePattern) {
		return true
	}
	for idx := range apiErr.Details {
		if apiErr.Details[idx].Code == CodeInvalidContent && apiErr.Details[idx].Path != "" {
			return true
		}
	}
	return false
}

func IsPermission(err error) bool {
	apiErr, ok := apiErrorOf(err)
	if !ok {
		return false
	}
	if apiErr.hasCode(CodeInsufficientPermission) {
		return true
	}
	return (apiErr.Status == http.StatusForbidden || apiErr.hasCode(CodeUserError)) &&
		apiErr.detailContains(permissionPhrase)
}

func IsUnsupportedTable(err error) bool {
	apiErr, ok := apiErrorOf(err)
	return ok && apiErr.detailMatches(unsupportedTablePatterns...)
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
	if hasCode(err, CodeUnexpectedError) {
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
