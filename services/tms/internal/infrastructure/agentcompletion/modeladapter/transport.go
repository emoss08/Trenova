package modeladapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
)

// TransportError is a non-2xx reply from a provider. Retryability is decided here
// rather than at the call site so every protocol backs off on the same signals.
type TransportError struct {
	StatusCode int
	Retryable  bool
	Message    string
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("provider request failed (status %d): %s", e.StatusCode, e.Message)
}

// IsRetryable reports whether err is worth another attempt. An error that is not
// a TransportError is a transport-level failure (dial, TLS, timeout) and is
// retried, since those are the failures most likely to be transient.
// StreamInterrupted is a stream that failed after the provider had named the
// model serving it. The failure is the wrapped error, so retry decisions read
// through it; the model is kept so a reply cut off partway is recorded under
// the model that actually produced it rather than the configured alias.
type StreamInterrupted struct {
	Model string
	Err   error
}

func (e *StreamInterrupted) Error() string { return e.Err.Error() }

func (e *StreamInterrupted) Unwrap() error { return e.Err }

// interrupted wraps a stream failure with the served model when one is known.
func interrupted(err error, model string) error {
	if err == nil || model == "" {
		return err
	}

	return &StreamInterrupted{Model: model, Err: err}
}

// ServedModel is the model a failed stream reported before it died, or the
// fallback when it never got that far.
func ServedModel(err error, fallback string) string {
	var cut *StreamInterrupted
	if errors.As(err, &cut) && cut.Model != "" {
		return cut.Model
	}

	return fallback
}

func IsRetryable(err error) bool {
	var te *TransportError
	if errors.As(err, &te) {
		return te.Retryable
	}

	return true
}

// postJSON sends body to url and decodes the reply into out. Headers are applied
// by the caller because each protocol authenticates differently.
func postJSON(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	body any,
	out any,
) error {
	encoded, err := sonic.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode provider request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build provider request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute provider request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read provider response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &TransportError{
			StatusCode: resp.StatusCode,
			Retryable: resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode >= http.StatusInternalServerError,
			Message: parseErrorMessage(payload),
		}
	}

	if err = sonic.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode provider response: %w", err)
	}

	return nil
}

func getJSON(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	out any,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build provider request: %w", err)
	}

	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute provider request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read provider response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &TransportError{
			StatusCode: resp.StatusCode,
			Retryable: resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode >= http.StatusInternalServerError,
			Message: parseErrorMessage(payload),
		}
	}

	if err = sonic.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode provider response: %w", err)
	}

	return nil
}

// errorEnvelope covers the shapes providers actually return. OpenAI-compatible
// servers nest under "error", Ollama returns a bare "error" string, and some
// self-hosted runtimes return neither, in which case the raw body is the message.
type errorEnvelope struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	Detail string `json:"detail"`
}

func parseErrorMessage(payload []byte) string {
	var envelope errorEnvelope
	if err := sonic.Unmarshal(payload, &envelope); err == nil {
		if msg := strings.TrimSpace(envelope.Error.Message); msg != "" {
			return msg
		}
		if msg := strings.TrimSpace(envelope.Detail); msg != "" {
			return msg
		}
	}

	// Ollama reports {"error": "model 'x' not found"}, where error is a string
	// rather than an object, so the struct decode above leaves it empty.
	var bare struct {
		Error string `json:"error"`
	}
	if err := sonic.Unmarshal(payload, &bare); err == nil {
		if msg := strings.TrimSpace(bare.Error); msg != "" {
			return msg
		}
	}

	return truncate(strings.TrimSpace(string(payload)), maxErrorBodyChars)
}

const maxErrorBodyChars = 500

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "..."
}
