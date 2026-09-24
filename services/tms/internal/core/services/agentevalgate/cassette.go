package agentevalgate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	CassetteAuthored = "authored"
	CassetteRecorded = "recorded"
)

var (
	ErrCassetteExhausted = errors.New("the cassette has no recorded response for this step")
	ErrNotRecordable     = errors.New("a cassette replays chat turns only")
	canonicalJSON        = sonic.Config{SortMapKeys: true}.Froze()
)

type Cassette struct {
	Case   string         `json:"case"`
	Source string         `json:"source"`
	Model  string         `json:"model,omitempty"`
	Steps  []CassetteStep `json:"steps"`
}

type CassetteStep struct {
	Step        int              `json:"step"`
	RequestHash string           `json:"requestHash"`
	Response    CassetteResponse `json:"response"`
}

type CassetteResponse struct {
	Text      string         `json:"text,omitempty"`
	ToolCalls []CassetteCall `json:"toolCalls,omitempty"`
	Model     string         `json:"model,omitempty"`
}

type CassetteCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type StaleStep struct {
	Step     int
	Recorded string
	Now      string
}

func LoadCassette(path string) (*Cassette, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cassette Cassette
	if err = sonic.Unmarshal(raw, &cassette); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &cassette, nil
}

func (c *Cassette) Save(path string) error {
	encoded, err := MarshalJSON(c)
	if err != nil {
		return err
	}

	return WriteFile(path, encoded)
}

type hashedMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []CassetteCall `json:"toolCalls,omitempty"`
	ToolCallID string         `json:"toolCallId,omitempty"`
	ToolName   string         `json:"toolName,omitempty"`
	IsError    bool           `json:"isError,omitempty"`
}

type hashedRequest struct {
	System   string                  `json:"system"`
	Messages []hashedMessage         `json:"messages"`
	Tools    []serviceports.ToolSpec `json:"tools"`
}

func HashRequest(req *serviceports.ChatCompletionRequest) (string, error) {
	hashed := hashedRequest{
		System:   req.System,
		Messages: make([]hashedMessage, 0, len(req.Messages)),
		Tools:    req.Tools,
	}
	for idx := range req.Messages {
		message := req.Messages[idx]
		hashed.Messages = append(hashed.Messages, hashedMessage{
			Role:       string(message.Role),
			Content:    message.Content,
			ToolCalls:  cassetteCalls(message.ToolCalls),
			ToolCallID: message.ToolCallID,
			ToolName:   message.ToolName,
			IsError:    message.IsError,
		})
	}

	encoded, err := canonicalJSON.Marshal(hashed)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)

	return hex.EncodeToString(sum[:]), nil
}

func cassetteCalls(calls []serviceports.ToolCall) []CassetteCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]CassetteCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, CassetteCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments})
	}

	return out
}

func (r CassetteResponse) completion() *serviceports.ChatCompletionResult {
	calls := make([]serviceports.ToolCall, 0, len(r.ToolCalls))
	for _, call := range r.ToolCalls {
		calls = append(calls, serviceports.ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	model := r.Model
	if model == "" {
		model = "cassette"
	}

	return &serviceports.ChatCompletionResult{
		Text:            r.Text,
		ToolCalls:       calls,
		ModelIdentifier: model,
	}
}

func responseOf(result *serviceports.ChatCompletionResult) CassetteResponse {
	return CassetteResponse{
		Text:      result.Text,
		ToolCalls: cassetteCalls(result.ToolCalls),
		Model:     result.ModelIdentifier,
	}
}

type CassettePlayer struct {
	mu       sync.Mutex
	cassette *Cassette
	next     int
	stale    []StaleStep
	hashes   []string
}

func NewCassettePlayer(cassette *Cassette) *CassettePlayer {
	return &CassettePlayer{cassette: cassette}
}

func (p *CassettePlayer) Stale() []StaleStep {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]StaleStep(nil), p.stale...)
}

func (p *CassettePlayer) Hashes() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]string(nil), p.hashes...)
}

func (p *CassettePlayer) Played() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.next
}

func (p *CassettePlayer) CompleteChat(
	_ context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	hash, err := HashRequest(req)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.next >= len(p.cassette.Steps) {
		return nil, fmt.Errorf("%w: step %d of %s", ErrCassetteExhausted, p.next, p.cassette.Case)
	}
	step := p.cassette.Steps[p.next]
	p.next++
	p.hashes = append(p.hashes, hash)
	if step.RequestHash != hash {
		p.stale = append(p.stale, StaleStep{Step: step.Step, Recorded: step.RequestHash, Now: hash})
	}

	return step.Response.completion(), nil
}

func (p *CassettePlayer) StreamChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	result, err := p.CompleteChat(ctx, req)
	if err != nil {
		return nil, err
	}
	if sink != nil && result.Text != "" {
		sink(result.Text)
	}

	return result, nil
}

func (*CassettePlayer) CompleteStructured(
	context.Context,
	*serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	return nil, ErrNotRecordable
}

func (*CassettePlayer) SubmitBackground(
	context.Context,
	*serviceports.StructuredCompletionRequest,
) (*serviceports.BackgroundSubmission, error) {
	return nil, ErrNotRecordable
}

func (*CassettePlayer) PollBackground(
	context.Context,
	*serviceports.BackgroundPollRequest,
) (*serviceports.BackgroundOutcome, error) {
	return nil, ErrNotRecordable
}

type CassetteRecorder struct {
	serviceports.CompletionService

	mu       sync.Mutex
	cassette Cassette
}

func NewCassetteRecorder(
	completion serviceports.CompletionService,
	caseName string,
) *CassetteRecorder {
	return &CassetteRecorder{
		CompletionService: completion,
		cassette:          Cassette{Case: caseName, Source: CassetteRecorded},
	}
}

func (r *CassetteRecorder) Cassette() *Cassette {
	r.mu.Lock()
	defer r.mu.Unlock()

	recorded := r.cassette
	recorded.Steps = append([]CassetteStep(nil), r.cassette.Steps...)

	return &recorded
}

func (r *CassetteRecorder) CompleteChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	return r.record(req, func() (*serviceports.ChatCompletionResult, error) {
		return r.CompletionService.CompleteChat(ctx, req)
	})
}

func (r *CassetteRecorder) StreamChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	return r.record(req, func() (*serviceports.ChatCompletionResult, error) {
		return r.CompletionService.StreamChat(ctx, req, sink)
	})
}

func (r *CassetteRecorder) record(
	req *serviceports.ChatCompletionRequest,
	call func() (*serviceports.ChatCompletionResult, error),
) (*serviceports.ChatCompletionResult, error) {
	hash, err := HashRequest(req)
	if err != nil {
		return nil, err
	}

	result, err := call()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cassette.Steps = append(r.cassette.Steps, CassetteStep{
		Step:        len(r.cassette.Steps),
		RequestHash: hash,
		Response:    responseOf(result),
	})
	if r.cassette.Model == "" {
		r.cassette.Model = result.ModelIdentifier
	}

	return result, nil
}
