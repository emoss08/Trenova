package intelkit

import (
	"context"
	"errors"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	messageUnauthorized    = "the provider rejected the configured credentials"
	messagePaymentRequired = "the provider reports the account's payment failed"
	messageNotFound        = "no carrier matched the identifier"
	messageRateLimited     = "the provider is rate limiting requests"
	messageInvalidRequest  = "the provider rejected the request"
	messageUnavailable     = "the provider is unavailable"
)

type UserError struct {
	Message string
	Err     error
}

func (e *UserError) Error() string { return e.Message }

func (e *UserError) Unwrap() error { return e.Err }

func NewUserError(message string, cause error) error {
	return &UserError{Message: message, Err: cause}
}

type ErrorMapper struct {
	Provider        integration.Type
	Unauthorized    func(error) bool
	PaymentRequired func(error) bool
	NotFound        func(error) bool
	RateLimited     func(error) bool
	Invalid         func(error) bool
}

func (m ErrorMapper) Map(err error) error {
	if err == nil {
		return nil
	}

	var limited *restx.RateLimitedError
	if errors.As(err, &limited) {
		return err
	}

	var providerErr *carrierintelports.CarrierIntelProviderError
	if errors.As(err, &providerErr) {
		return err
	}

	var transportErr *restx.TransportError
	isTransport := errors.As(err, &transportErr)
	contextErr := errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
	if !isTransport && contextErr {
		return err
	}

	status := restx.StatusCode(err)
	mapped := &carrierintelports.CarrierIntelProviderError{
		Provider:   m.Provider,
		StatusCode: status,
		Err:        err,
	}

	switch {
	case m.matches(m.Unauthorized, err) ||
		status == http.StatusUnauthorized || status == http.StatusForbidden:
		mapped.Kind = carrierintelports.CarrierIntelErrorUnauthorized
		mapped.Message = messageUnauthorized
	case m.matches(m.PaymentRequired, err) || status == http.StatusPaymentRequired:
		mapped.Kind = carrierintelports.CarrierIntelErrorPaymentRequired
		mapped.Message = messagePaymentRequired
	case m.matches(m.NotFound, err) || status == http.StatusNotFound:
		mapped.Kind = carrierintelports.CarrierIntelErrorNotFound
		mapped.Message = messageNotFound
	case m.matches(m.RateLimited, err) || status == http.StatusTooManyRequests:
		mapped.Kind = carrierintelports.CarrierIntelErrorRateLimited
		mapped.Message = messageRateLimited
		mapped.RetryAfter = restx.RetryAfterOf(err)
	case m.matches(m.Invalid, err) ||
		status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		mapped.Kind = carrierintelports.CarrierIntelErrorInvalidRequest
		mapped.Message = messageInvalidRequest
	default:
		mapped.Kind = carrierintelports.CarrierIntelErrorUnavailable
		mapped.Message = messageUnavailable
	}
	return mapped
}

func (m ErrorMapper) InvalidRequest(message string) error {
	return &carrierintelports.CarrierIntelProviderError{
		Provider: m.Provider,
		Kind:     carrierintelports.CarrierIntelErrorInvalidRequest,
		Message:  message,
	}
}

func (m ErrorMapper) Unsupported(message string) error {
	return &carrierintelports.CarrierIntelProviderError{
		Provider: m.Provider,
		Kind:     carrierintelports.CarrierIntelErrorUnsupported,
		Message:  message,
	}
}

func (m ErrorMapper) NotFoundError() error {
	return &carrierintelports.CarrierIntelProviderError{
		Provider:   m.Provider,
		Kind:       carrierintelports.CarrierIntelErrorNotFound,
		StatusCode: http.StatusOK,
		Message:    messageNotFound,
	}
}

func (ErrorMapper) matches(check func(error) bool, err error) bool {
	return check != nil && check(err)
}
