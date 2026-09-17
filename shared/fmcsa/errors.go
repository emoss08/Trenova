package fmcsa

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/restx"
)

var (
	ErrNotFound          = errors.New("fmcsa: no matching carrier record")
	ErrWebKeyRequired    = errors.New("fmcsa: QCMobile web key is required")
	ErrInvalidArgument   = errors.New("fmcsa: invalid argument")
	ErrUnexpectedPayload = errors.New("fmcsa: unexpected response payload")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || restx.IsStatus(err, http.StatusNotFound)
}

func IsUnauthorized(err error) bool {
	return restx.IsStatus(err, http.StatusUnauthorized) ||
		restx.IsStatus(err, http.StatusForbidden)
}

func IsRateLimited(err error) bool {
	return restx.IsRateLimited(err)
}

func decodeError(status int, body []byte, header http.Header) error {
	return restx.DecodeAPIError(status, body, header)
}

func isCredentialMessage(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "webkey") ||
		strings.Contains(lower, "web key") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "not authorized")
}
