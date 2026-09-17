package modeladapter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/stringutils"
)

// StreamSink receives model text as it is produced, in order, possibly one
// token at a time. It is never called after Stream returns.
type StreamSink func(delta string)

// Streamer is an adapter whose protocol can deliver text incrementally. The
// final Response is the same one Complete would have returned; the sink is a
// preview of its Text, not a substitute for it.
type Streamer interface {
	Stream(ctx context.Context, call *Call, sink StreamSink) (*Response, error)
}

// maxStreamLineBytes bounds one line of a streamed reply. A single tool-call
// argument payload can be large, but anything past this is not a line a model
// produced; it is a broken framing that would otherwise consume memory forever.
const maxStreamLineBytes = 4 << 20

// postStream sends body and hands back the open reply body. A non-2xx status
// is read to completion and reported the same way postJSON reports it, so a
// stream that fails before its first byte is indistinguishable from a failed
// blocking call to the retry logic.
func postStream(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	body any,
) (io.ReadCloser, error) {
	encoded, err := sonic.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/x-ndjson, application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute provider request: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		return nil, &TransportError{
			StatusCode: resp.StatusCode,
			Retryable: resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode >= http.StatusInternalServerError,
			Message: parseErrorMessage(payload),
		}
	}

	return resp.Body, nil
}

// readSSE parses a text/event-stream body and hands each event to handle. It
// follows the specification's framing: an event ends at a blank line, several
// data lines join with newlines, and a line starting with a colon is a comment
// (providers send those as keep-alives). Returning an error from handle stops
// the read and surfaces that error.
func readSSE(r io.Reader, handle func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), maxStreamLineBytes)

	var (
		event string
		data  []string
	)

	flush := func() error {
		if len(data) == 0 {
			event = ""
			return nil
		}
		err := handle(event, strings.Join(data, "\n"))
		event = ""
		data = data[:0]

		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
			continue
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read provider stream: %w", err)
	}

	return flush()
}

// readNDJSON hands each non-empty line to handle. Ollama streams this way: one
// JSON object per line, no framing beyond the newline.
func readNDJSON(r io.Reader, handle func(line []byte) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), maxStreamLineBytes)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := handle(line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read provider stream: %w", err)
	}

	return nil
}

// streamError is a failure the provider reported inside an otherwise healthy
// stream. It is retryable when the provider says the fault is on its side.
func streamError(errType, message string) error {
	retryable := strings.Contains(errType, "overloaded") ||
		strings.Contains(errType, "rate_limit") ||
		strings.Contains(errType, "server_error") ||
		strings.Contains(errType, "api_error")

	return &TransportError{
		StatusCode: http.StatusBadGateway,
		Retryable:  retryable,
		Message:    stringutils.FirstNonEmpty(message, "provider stream failed"),
	}
}

// decodeArguments turns a JSON-encoded argument string into a map. Unparseable
// arguments become an empty map for the same reason the blocking path tolerates
// them: the tool's own validation reports a clearer error than a failed turn.
func decodeArguments(raw string) map[string]any {
	args := map[string]any{}
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		_ = sonic.Unmarshal([]byte(trimmed), &args)
	}

	return args
}
