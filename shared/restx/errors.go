package restx

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/jsonflex"
)

type RateLimitedError struct {
	RetryAfter time.Duration
	Key        string
}

func (e *RateLimitedError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded for %q; retry after %s", e.Key, e.RetryAfter)
	}
	return fmt.Sprintf("rate limit exceeded for %q", e.Key)
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Hint       string
	RequestID  string
	RetryAfter time.Duration
	Body       []byte
}

func (e *APIError) Error() string {
	var b strings.Builder
	b.Grow(64 + len(e.Code) + len(e.Message) + len(e.Hint) + len(e.RequestID))
	b.WriteString("api error: status ")
	b.WriteString(strconv.Itoa(e.StatusCode))
	if e.Code != "" {
		b.WriteString(" code ")
		b.WriteString(e.Code)
	}
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if message != "" {
		b.WriteString(": ")
		b.WriteString(message)
	}
	if e.Hint != "" {
		b.WriteString(" (hint: ")
		b.WriteString(e.Hint)
		b.WriteString(")")
	}
	if e.RequestID != "" {
		b.WriteString(" [request ")
		b.WriteString(e.RequestID)
		b.WriteString("]")
	}
	return b.String()
}

type TransportError struct {
	Method string
	Path   string
	Err    error
	redact func(string) string
}

func (e *TransportError) Error() string {
	cause := "unknown transport failure"
	if e.Err != nil {
		cause = e.Err.Error()
	}
	if e.redact != nil {
		cause = e.redact(cause)
	}
	return fmt.Sprintf("transport error: %s %s: %s", e.Method, e.Path, cause)
}

func (e *TransportError) Unwrap() error {
	return e.Err
}

func IsStatus(err error, code int) bool {
	return StatusCode(err) == code && code != 0
}

func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func IsRateLimited(err error) bool {
	var limited *RateLimitedError
	if errors.As(err, &limited) {
		return true
	}
	return IsStatus(err, http.StatusTooManyRequests)
}

func RetryAfterOf(err error) time.Duration {
	var limited *RateLimitedError
	if errors.As(err, &limited) {
		return limited.RetryAfter
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

func DecodeAPIError(status int, body []byte, header http.Header) *APIError {
	apiErr := &APIError{
		StatusCode: status,
		Body:       body,
	}
	if header != nil {
		if retryAfter, ok := ParseRetryAfter(header.Get("Retry-After")); ok {
			apiErr.RetryAfter = retryAfter
		}
		apiErr.RequestID = firstHeader(header, "X-Request-Id", "Request-Id")
	}

	obj, err := jsonflex.DecodeObject(body)
	if err != nil {
		apiErr.Message = http.StatusText(status)
		return apiErr
	}

	if nested, ok := obj.Object("error"); ok {
		apiErr.Message = nested.Text("message", "error", "detail")
		apiErr.Code = nested.Text("code", "type")
		apiErr.Hint = nested.Text("hint")
	} else {
		apiErr.Message = obj.Text("error", "message", "detail", "content")
	}
	if apiErr.Code == "" {
		apiErr.Code = obj.Text("code", "error_code", "errorCode")
	}
	if apiErr.Hint == "" {
		apiErr.Hint = obj.Text("hint")
	}
	if requestID := obj.Text("request_id", "requestId"); requestID != "" {
		apiErr.RequestID = requestID
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}

func firstHeader(header http.Header, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}
