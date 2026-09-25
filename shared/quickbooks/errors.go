package quickbooks

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

var (
	ErrRealmIDRequired      = errors.New("quickbooks: realm id is required")
	ErrAccessTokenRequired  = errors.New("quickbooks: access token is required")
	ErrClientIDRequired     = errors.New("quickbooks: client id is required")
	ErrClientSecretRequired = errors.New("quickbooks: client secret is required")
	ErrRedirectURLRequired  = errors.New("quickbooks: redirect url is required")
	ErrStateRequired        = errors.New("quickbooks: state is required")
	ErrCodeRequired         = errors.New("quickbooks: authorization code is required")
	ErrTokenRequired        = errors.New("quickbooks: token is required")
	ErrInvalidGrant         = errors.New("quickbooks: the authorization was revoked or has expired")
	ErrInvalidClient        = errors.New("quickbooks: the app's client id or secret was not accepted")
	ErrUnexpectedPayload    = errors.New("quickbooks: unexpected response payload")
	ErrInvalidSignature     = errors.New("quickbooks: webhook signature does not match")
	ErrMissingSignature     = errors.New("quickbooks: webhook signature is missing")
	ErrVerifierRequired     = errors.New("quickbooks: webhook verifier token is required")
)

type FaultDetail struct {
	Message string `json:"Message"`
	Detail  string `json:"Detail"`
	Code    string `json:"code"`
	Element string `json:"element"`
}

type FaultError struct {
	StatusCode int
	Type       string
	Errors     []FaultDetail
}

func (f *FaultError) Error() string {
	var b strings.Builder
	b.WriteString("quickbooks fault")
	if f.Type != "" {
		b.WriteString(" ")
		b.WriteString(f.Type)
	}
	for idx := range f.Errors {
		b.WriteString(": ")
		b.WriteString(f.Errors[idx].Message)
		if f.Errors[idx].Detail != "" {
			b.WriteString(" (")
			b.WriteString(f.Errors[idx].Detail)
			b.WriteString(")")
		}
	}
	return b.String()
}

func (f *FaultError) FirstCode() string {
	if len(f.Errors) == 0 {
		return ""
	}
	return f.Errors[0].Code
}

type OAuthError struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return "quickbooks oauth error " + e.Code + ": " + e.Description
	}
	return "quickbooks oauth error " + e.Code
}

func (e *OAuthError) Is(target error) bool {
	switch target {
	case ErrInvalidGrant:
		return e.Code == "invalid_grant"
	case ErrInvalidClient:
		return e.Code == "invalid_client"
	default:
		return false
	}
}

type faultEnvelope struct {
	Fault *struct {
		Type  string        `json:"type"`
		Error []FaultDetail `json:"Error"`
	} `json:"Fault"`
}

type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func decodeAPIError(status int, body []byte, header http.Header) error {
	var envelope faultEnvelope
	if err := sonic.Unmarshal(body, &envelope); err == nil && envelope.Fault != nil {
		return &FaultError{
			StatusCode: status,
			Type:       envelope.Fault.Type,
			Errors:     envelope.Fault.Error,
		}
	}
	return restx.DecodeAPIError(status, body, header)
}

func decodeOAuthError(status int, body []byte, header http.Header) error {
	var parsed oauthErrorBody
	if err := sonic.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return &OAuthError{
			StatusCode:  status,
			Code:        parsed.Error,
			Description: parsed.ErrorDescription,
		}
	}
	return restx.DecodeAPIError(status, body, header)
}

func IsUnauthorized(err error) bool {
	var fault *FaultError
	if errors.As(err, &fault) && fault.StatusCode == http.StatusUnauthorized {
		return true
	}
	return restx.IsStatus(err, http.StatusUnauthorized)
}

func IsForbidden(err error) bool {
	var fault *FaultError
	if errors.As(err, &fault) && fault.StatusCode == http.StatusForbidden {
		return true
	}
	return restx.IsStatus(err, http.StatusForbidden)
}

func IsInvalidGrant(err error) bool {
	return errors.Is(err, ErrInvalidGrant)
}

func IsInvalidClient(err error) bool {
	return errors.Is(err, ErrInvalidClient)
}

func IsRateLimited(err error) bool {
	var fault *FaultError
	if errors.As(err, &fault) && fault.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return restx.IsRateLimited(err) || restx.IsStatus(err, http.StatusTooManyRequests)
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
	var fault *FaultError
	if errors.As(err, &fault) {
		return fault.StatusCode >= http.StatusInternalServerError
	}
	var apiErr *restx.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= http.StatusInternalServerError
	}
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		return oauthErr.StatusCode >= http.StatusInternalServerError
	}
	return false
}
