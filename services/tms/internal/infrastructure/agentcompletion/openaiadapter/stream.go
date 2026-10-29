package openaiadapter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

// maxStreamLineBytes bounds one line of a streamed reply. A single tool-call
// argument payload can be large, but anything past this is not a line a model
// produced; it is a broken framing that would otherwise consume memory forever.
const maxStreamLineBytes = 4 << 20

// postStream sends body and hands back the open reply body. A non-2xx status
// is read to completion and reported the same way postJSON reports it, so a
// stream that fails before its first byte is indistinguishable from a failed
// blocking call to the retry logic.
//
// It takes the whole call rather than a client because a stream needs the one
// without a whole-request deadline, and because the returned body is wrapped in
// an idle guard built from the call's own timeout.
func postStream(
	ctx context.Context,
	call *modeladapter.Call,
	url string,
	headers map[string]string,
	body any,
) (io.ReadCloser, error) {
	encoded, err := modeladapter.RequestJSON.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}

	// Cancelled by the idle guard when the provider stops sending, and by Close
	// when the caller is done, so neither leaves the connection open.
	streamCtx, cancel := context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodPost,
		url,
		bytes.NewReader(encoded),
	)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("build provider request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/x-ndjson, application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := call.StreamHTTPClient().Do(req)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("execute provider request: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()

		return nil, modeladapter.NewTransportError(resp, payload)
	}

	return modeladapter.NewIdleGuard(resp.Body, cancel, call.StreamIdleTimeout()), nil
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

// textReasoning wraps readable thinking with nothing to replay. Nil for
// nothing, so a reply without thinking carries no trace at all.
func textReasoning(text string) *modeladapter.ReasoningTrace {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	return &modeladapter.ReasoningTrace{Text: text}
}
