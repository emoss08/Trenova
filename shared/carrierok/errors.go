package carrierok

import (
	"errors"
	"net/http"

	"github.com/emoss08/trenova/shared/restx"
)

var (
	ErrNotFound          = errors.New("carrierok: no matching carrier profile")
	ErrAPIKeyRequired    = errors.New("carrierok: API key is required")
	ErrInvalidQuery      = errors.New("carrierok: invalid query")
	ErrUnexpectedPayload = errors.New("carrierok: unexpected response payload")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || restx.IsStatus(err, http.StatusNotFound)
}

func IsPaymentRequired(err error) bool {
	return restx.IsStatus(err, http.StatusPaymentRequired)
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
