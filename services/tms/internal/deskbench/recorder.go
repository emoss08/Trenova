package deskbench

import (
	"context"
	"slices"
	"sync"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type ModelCallKind string

const (
	ModelCallChat       = ModelCallKind("chat")
	ModelCallStream     = ModelCallKind("stream")
	ModelCallStructured = ModelCallKind("structured")
)

type ModelCall struct {
	Seq              int                                       `json:"seq"`
	Kind             ModelCallKind                             `json:"kind"`
	StartedAt        time.Time                                 `json:"startedAt"`
	FinishedAt       time.Time                                 `json:"finishedAt"`
	Attribution      serviceports.AIUsageAttribution           `json:"attribution"`
	Chat             *serviceports.ChatCompletionRequest       `json:"chat,omitempty"`
	Structured       *serviceports.StructuredCompletionRequest `json:"structured,omitempty"`
	ChatResult       *serviceports.ChatCompletionResult        `json:"chatResult,omitempty"`
	StructuredResult *serviceports.StructuredCompletionResult  `json:"structuredResult,omitempty"`
	Retries          []serviceports.ChatRetryNotice            `json:"retries,omitempty"`
	Error            string                                    `json:"error,omitempty"`
}

func (c *ModelCall) Duration() time.Duration {
	return c.FinishedAt.Sub(c.StartedAt)
}

func (c *ModelCall) belongsTo(threadID, turnID pulid.ID) bool {
	if c.Attribution.ThreadID.IsNotNil() {
		return c.Attribution.ThreadID == threadID
	}

	return turnID.IsNotNil() && c.Attribution.OwnerID == turnID
}

type Recorder struct {
	serviceports.CompletionService

	mu    sync.Mutex
	seq   int
	calls []*ModelCall
}

func NewRecorder(inner serviceports.CompletionService) *Recorder {
	return &Recorder{CompletionService: inner}
}

func (r *Recorder) CompleteChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	call := r.openChat(ModelCallChat, req)
	restore := call.watchRetries(&r.mu, req)
	defer restore()

	result, err := r.CompletionService.CompleteChat(ctx, req)
	r.closeChat(call, result, err)

	return result, err
}

func (r *Recorder) StreamChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	call := r.openChat(ModelCallStream, req)
	restore := call.watchRetries(&r.mu, req)
	defer restore()

	result, err := r.CompletionService.StreamChat(ctx, req, sink)
	r.closeChat(call, result, err)

	return result, err
}

func (r *Recorder) CompleteStructured(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	call := &ModelCall{
		Kind:        ModelCallStructured,
		StartedAt:   time.Now(),
		Attribution: req.Attribution,
		Structured:  snapshot(req),
	}
	r.open(call)

	result, err := r.CompletionService.CompleteStructured(ctx, req)

	r.mu.Lock()
	defer r.mu.Unlock()
	call.FinishedAt = time.Now()
	call.StructuredResult = snapshot(result)
	if err != nil {
		call.Error = err.Error()
	}

	return result, err
}

func (r *Recorder) For(threadID, turnID pulid.ID, start, end time.Time) []*ModelCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*ModelCall, 0, 8)
	for _, call := range r.calls {
		if call.StartedAt.Before(start) || call.StartedAt.After(end) {
			continue
		}
		if call.belongsTo(threadID, turnID) {
			out = append(out, call)
		}
	}

	return out
}

func (r *Recorder) Unattributed(start, end time.Time) []*ModelCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*ModelCall, 0, 4)
	for _, call := range r.calls {
		if call.StartedAt.Before(start) || call.StartedAt.After(end) {
			continue
		}
		if call.Attribution.ThreadID.IsNil() && call.Attribution.OwnerID.IsNil() {
			out = append(out, call)
		}
	}

	return out
}

func (r *Recorder) open(call *ModelCall) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	call.Seq = r.seq
	r.calls = append(r.calls, call)
}

func (r *Recorder) openChat(kind ModelCallKind, req *serviceports.ChatCompletionRequest) *ModelCall {
	call := &ModelCall{
		Kind:        kind,
		StartedAt:   time.Now(),
		Attribution: req.Attribution,
		Chat:        snapshot(req),
	}
	r.open(call)

	return call
}

func (r *Recorder) closeChat(
	call *ModelCall,
	result *serviceports.ChatCompletionResult,
	err error,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	call.FinishedAt = time.Now()
	call.ChatResult = snapshot(result)
	if err != nil {
		call.Error = err.Error()
	}
}

func (c *ModelCall) watchRetries(mu *sync.Mutex, req *serviceports.ChatCompletionRequest) func() {
	original := req.RetrySink
	req.RetrySink = func(notice serviceports.ChatRetryNotice) {
		mu.Lock()
		c.Retries = append(c.Retries, notice)
		mu.Unlock()
		if original != nil {
			original(notice)
		}
	}

	return func() { req.RetrySink = original }
}

func snapshot[T any](value *T) *T {
	if value == nil {
		return nil
	}

	copied := new(T)
	if err := jsonutils.Convert(value, copied); err != nil {
		return value
	}

	return copied
}

func sortCalls(calls []*ModelCall) {
	slices.SortFunc(calls, func(a, b *ModelCall) int { return a.Seq - b.Seq })
}
