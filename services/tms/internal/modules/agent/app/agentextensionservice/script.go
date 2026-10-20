package agentextensionservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var _ agentextensionports.ScriptRunner = (*Service)(nil)

const (
	scriptMetadataStep = "trenovaScriptStep"
	scriptRunMargin    = 15 * time.Second
	scriptLifeMargin   = 30 * time.Second
	scriptFailure      = "The script could not be run in the sandbox. This was not a fault in the code; " +
		"answer with the discrete reads instead."
)

var errScriptInputs = errors.New("script inputs")

type scriptCall struct {
	*extensionSettings
	settings agentextension.E2BSettings
	provider agentextensionports.SandboxProvider
}

func (s *Service) codeMode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*scriptCall, error) {
	extension, err := s.serving(ctx, tenantInfo, agentextension.CapabilityCodeMode)
	if err != nil {
		return nil, err
	}

	provider, ok := s.sandboxes[extension.typ]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"the {0} extension cannot run code", extension.typ.String(),
		)
	}

	settings, err := agentextension.ParseE2BSettings(extension.values)
	if err != nil {
		s.l.Error("stored code mode settings are invalid",
			zap.String("type", extension.typ.String()), zap.Error(err))
		return nil, &ToolError{Message: "The " + extension.def.title() + invalidSettingsSuffix}
	}
	if !settings.CodeMode {
		return nil, &ToolError{Message: "Read-only scripts are turned off for this organization. " +
			"Answer with the discrete reads."}
	}

	return &scriptCall{extensionSettings: extension, settings: settings, provider: provider}, nil
}

func checkScript(req *agentextensionports.ScriptRequest) error {
	switch {
	case strings.TrimSpace(req.Code) == "":
		return &ToolError{Message: "The script is empty. Send the Python to run."}
	case utf8.RuneCountInString(req.Code) > codemode.MaxCodeChars:
		return &ToolError{Message: fmt.Sprintf(
			"A script is at most %d characters. Make it shorter.", codemode.MaxCodeChars,
		)}
	case req.Broker == nil:
		return errortypes.NewBusinessError("a script needs a broker to read through")
	}
	if len(req.Inputs) > codemode.MaxInputKeys {
		return &ToolError{Message: fmt.Sprintf(
			"A script takes at most %d inputs.", codemode.MaxInputKeys,
		)}
	}

	return nil
}

func (s *Service) RunScript(
	ctx context.Context,
	req *agentextensionports.ScriptRequest,
) (*agentextensionports.ScriptOutcome, error) {
	if err := checkScript(req); err != nil {
		return nil, err
	}
	inputs, err := encodeInputs(req.Inputs)
	if err != nil {
		return nil, &ToolError{Message: fmt.Sprintf(
			"The inputs must encode as JSON in at most %d bytes.", codemode.MaxInputBytes,
		)}
	}

	call, err := s.codeMode(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	limits := codemode.LimitsFor(call.settings.ScriptSeconds, call.settings.ScriptReads)
	req.Broker.Bound(limits)

	metered := meteredCall{
		tenant: req.TenantInfo,
		typ:    call.typ,
		limit:  call.settings.DailyRequestLimit,
	}
	day, err := s.reserve(ctx, metered)
	if err != nil {
		return nil, err
	}

	attempt := &scriptAttempt{call: call, limits: limits}
	outcome, runErr := s.runScript(ctx, attempt, req, inputs)
	settled := settledCall{day: day, failed: runErr != nil}
	if outcome != nil {
		settled.cost = outcome.CostUSD
		settled.failed = outcome.Status == agentextensionports.ScriptFailed
	}
	s.settle(ctx, metered, settled)

	return outcome, runErr
}

func encodeInputs(inputs map[string]any) ([]byte, error) {
	if inputs == nil {
		inputs = map[string]any{}
	}
	encoded, err := sonic.Marshal(inputs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errScriptInputs, err)
	}
	if len(encoded) > codemode.MaxInputBytes {
		return nil, fmt.Errorf("%w: too large", errScriptInputs)
	}

	return encoded, nil
}

type scriptAttempt struct {
	call    *scriptCall
	limits  codemode.Limits
	sandbox *agentextensionports.Sandbox
	started time.Time
}

func (s *Service) runScript(
	ctx context.Context,
	attempt *scriptAttempt,
	req *agentextensionports.ScriptRequest,
	inputs []byte,
) (*agentextensionports.ScriptOutcome, error) {
	call := attempt.call
	nonce, err := newNonce()
	if err != nil {
		return nil, fmt.Errorf("mint the script's nonce: %w", err)
	}
	files, err := scriptFiles(req, nonce, attempt.limits, inputs)
	if err != nil {
		return nil, err
	}

	sandbox, err := call.provider.StartSandbox(ctx, &agentextensionports.SandboxRequest{
		Settings: call.values,
		Metadata: map[string]string{scriptMetadataStep: stepMetadata(req.StepKey)},
		Lifetime: attempt.limits.Wall + scriptLifeMargin,
	})
	if err != nil {
		return nil, s.providerError(call.failing(err, "start a sandbox"))
	}
	attempt.sandbox = sandbox
	attempt.started = time.Now()
	defer s.stopScriptSandbox(ctx, call, sandbox.ID)

	if err = call.provider.WriteFiles(ctx, sandbox, files); err != nil {
		return nil, s.providerError(call.failing(err, "load the script into the sandbox"))
	}

	relay := newScriptRelay(&scriptRelayParams{
		ctx:      ctx,
		nonce:    nonce,
		limits:   attempt.limits,
		broker:   req.Broker,
		provider: call.provider,
		sandbox:  sandbox,
		logger:   s.l,
	})
	execution, err := call.provider.RunCode(ctx, &agentextensionports.RunCodeRequest{
		Sandbox: sandbox,
		Code:    string(codemode.RunnerSource),
		Timeout: attempt.limits.Wall + scriptRunMargin,
		Output:  relay.filter,
	})
	relay.wait()
	if err != nil {
		outcome := scriptUsageOf(attempt)
		outcome.Status = agentextensionports.ScriptFailed
		outcome.Failure = scriptFailure
		s.l.Warn("a script could not be run", zap.Error(err))
		return outcome, nil
	}

	return relay.outcome(scriptUsageOf(attempt), execution), nil
}

func stepMetadata(key string) string {
	if len(key) > 32 {
		return key[:32]
	}

	return key
}

func newNonce() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func scriptFiles(
	req *agentextensionports.ScriptRequest,
	nonce string,
	limits codemode.Limits,
	inputs []byte,
) ([]agentextensionports.SandboxFile, error) {
	config, err := sonic.Marshal(codemode.ConfigFor(nonce, limits))
	if err != nil {
		return nil, fmt.Errorf("encode the script configuration: %w", err)
	}
	index, err := sonic.Marshal(req.Broker.Index())
	if err != nil {
		return nil, fmt.Errorf("encode the script's reads: %w", err)
	}

	return []agentextensionports.SandboxFile{
		{Path: codemode.SandboxConfigPath, Content: config},
		{Path: codemode.SandboxIndexPath, Content: index},
		{Path: codemode.SandboxInputsPath, Content: inputs},
		{Path: codemode.SandboxScriptPath, Content: []byte(req.Code)},
		{Path: codemode.SandboxShimPath, Content: codemode.ShimSource},
		{Path: codemode.SandboxMainPath, Content: codemode.MainSource},
	}, nil
}

func (s *Service) stopScriptSandbox(ctx context.Context, call *scriptCall, sandboxID string) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), analysisStopTimeout)
	defer cancel()

	if err := call.provider.StopSandbox(stopCtx, call.values, sandboxID); err != nil {
		s.l.Warn("could not stop a script sandbox; it ends at its own lifetime",
			zap.String("sandbox", sandboxID), zap.Error(err))
	}
}

func scriptUsageOf(attempt *scriptAttempt) *agentextensionports.ScriptOutcome {
	outcome := &agentextensionports.ScriptOutcome{Limits: attempt.limits}
	if attempt.sandbox == nil || attempt.started.IsZero() {
		return outcome
	}
	ran := time.Since(attempt.started)
	outcome.DurationMs = ran.Milliseconds()
	outcome.CostUSD = attempt.call.provider.CostOf(attempt.sandbox, ran)

	return outcome
}

type scriptRelayParams struct {
	ctx      context.Context
	nonce    string
	limits   codemode.Limits
	broker   codemode.Broker
	provider agentextensionports.SandboxProvider
	sandbox  *agentextensionports.Sandbox
	logger   *zap.Logger
}

type scriptRelay struct {
	p        *scriptRelayParams
	lines    *codemode.LineSplitter
	slots    chan struct{}
	inflight sync.WaitGroup

	mu       sync.Mutex
	output   strings.Builder
	cut      bool
	seen     map[int]struct{}
	result   sonic.NoCopyRawMessage
	resultIn bool
	end      *codemode.End
	limit    codemode.Limit
}

func newScriptRelay(p *scriptRelayParams) *scriptRelay {
	return &scriptRelay{
		p:     p,
		lines: codemode.NewLineSplitter(p.limits.FrameBytes + 256),
		slots: make(chan struct{}, max(p.limits.Concurrent, 1)),
		seen:  make(map[int]struct{}, 8),
	}
}

func (r *scriptRelay) filter(stream, text string) (string, error) {
	if stream != "stdout" {
		return text, nil
	}
	r.lines.Write(text, r.line)

	return "", nil
}

func (r *scriptRelay) line(line []byte, overflowed bool) {
	frame, isFrame := codemode.ParseFrame(line, r.p.nonce)
	if overflowed {
		if isFrame || len(line) == 0 {
			r.note(codemode.LimitFrame)
			return
		}
		r.keep(string(line) + "…\n")
		return
	}
	if !isFrame {
		r.keep(string(line) + "\n")
		return
	}

	switch frame.Kind {
	case codemode.FrameRequest:
		r.request(frame.Body)
	case codemode.FrameResult:
		r.keepResult(frame.Body)
	case codemode.FrameEnd:
		r.keepEnd(frame.Body)
	}
}

func (r *scriptRelay) request(body []byte) {
	var call codemode.Call
	if err := sonic.Unmarshal(body, &call); err != nil || call.Seq < 1 {
		r.note(codemode.LimitFrame)
		return
	}
	r.mu.Lock()
	if _, repeated := r.seen[call.Seq]; repeated {
		r.mu.Unlock()
		return
	}
	r.seen[call.Seq] = struct{}{}
	r.mu.Unlock()

	r.inflight.Add(1)
	go r.answer(call)
}

func (r *scriptRelay) answer(call codemode.Call) {
	defer r.inflight.Done()
	select {
	case r.slots <- struct{}{}:
	case <-r.p.ctx.Done():
		return
	}
	defer func() { <-r.slots }()

	arrived := time.Now()
	reply := r.p.broker.Read(r.p.ctx, call)
	body, err := codemode.ReplyBody(reply, r.p.nonce)
	if err != nil {
		body, _ = codemode.ReplyBody(codemode.Reply{
			Status:  codemode.CallFailed,
			Message: "the read's answer could not be encoded",
		}, r.p.nonce)
	}
	if err = r.p.provider.WriteFiles(r.p.ctx, r.p.sandbox, []agentextensionports.SandboxFile{
		{Path: codemode.ReplyPath(call.Seq), Content: body},
	}); err != nil {
		r.p.logger.Warn("could not answer a script's read",
			zap.Int("seq", call.Seq), zap.String("tool", call.Tool), zap.Error(err))
		return
	}
	r.p.broker.NoteReply(call.Seq, time.Since(arrived).Milliseconds())
}

func (r *scriptRelay) keep(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.output.WriteString(text)
	if r.output.Len() > 2*r.p.limits.OutputChars {
		r.trimLocked()
	}
}

func (r *scriptRelay) trimLocked() {
	kept := tailOf(r.output.String(), r.p.limits.OutputChars)
	r.output.Reset()
	r.output.WriteString(kept)
	r.cut = true
}

func tailOf(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	start := len(text) - limit
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}

	return text[start:]
}

func (r *scriptRelay) keepResult(body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.resultIn {
		return
	}
	r.resultIn = true
	if len(body) > r.p.limits.ResultBytes {
		r.limit = codemode.LimitResult
		return
	}
	var wrapped struct {
		Value sonic.NoCopyRawMessage `json:"value"`
	}
	if err := sonic.Unmarshal(body, &wrapped); err != nil {
		return
	}
	r.result = wrapped.Value
}

func (r *scriptRelay) keepEnd(body []byte) {
	var end codemode.End
	if err := sonic.Unmarshal(body, &end); err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.end == nil || (r.end.Status != codemode.EndKilled && end.Status == codemode.EndKilled) {
		r.end = &end
	}
}

func (r *scriptRelay) note(limit codemode.Limit) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.limit == codemode.LimitNone {
		r.limit = limit
	}
}

func (r *scriptRelay) wait() {
	r.lines.Flush(r.line)
	r.inflight.Wait()
}

func (r *scriptRelay) outcome(
	outcome *agentextensionports.ScriptOutcome,
	execution *agentextensionports.SandboxExecution,
) *agentextensionports.ScriptOutcome {
	r.mu.Lock()
	defer r.mu.Unlock()

	if execution.Stderr != "" {
		r.output.WriteString(execution.Stderr)
	}
	if r.output.Len() > r.p.limits.OutputChars {
		r.trimLocked()
	}
	outcome.Output = r.output.String()
	outcome.OutputTruncated = r.cut
	if r.cut {
		r.noteLocked(codemode.LimitOutput)
	}
	outcome.Result = r.result
	outcome.Limit = r.limit

	end := r.end
	switch {
	case execution.TimedOut:
		outcome.Status = agentextensionports.ScriptTimedOut
		outcome.Limit = codemode.LimitWall
	case end == nil:
		outcome.Status = agentextensionports.ScriptFailed
		outcome.Failure = "The script stopped without finishing, usually because it ran the sandbox " +
			"out of memory. Read less at once."
	default:
		outcome.RoundTripsMs = end.RoundTripsMs
		outcome.Error = end.Error
		r.settleEnd(outcome, end)
	}
	if hit := r.p.broker.Summary().Usage.Hit; hit != codemode.LimitNone &&
		outcome.Limit == codemode.LimitNone {
		outcome.Limit = hit
	}

	return outcome
}

func (r *scriptRelay) noteLocked(limit codemode.Limit) {
	if r.limit == codemode.LimitNone {
		r.limit = limit
	}
}

func (r *scriptRelay) settleEnd(outcome *agentextensionports.ScriptOutcome, end *codemode.End) {
	switch end.Status {
	case codemode.EndOK:
		outcome.Status = agentextensionports.ScriptSucceeded
	case codemode.EndRaised:
		outcome.Status = agentextensionports.ScriptErrored
	case codemode.EndKilled:
		if end.Limit == codemode.LimitWall || end.Limit == codemode.LimitCPU {
			outcome.Status = agentextensionports.ScriptTimedOut
		} else {
			outcome.Status = agentextensionports.ScriptStopped
		}
		outcome.Limit = end.Limit
	default:
		outcome.Status = agentextensionports.ScriptStopped
		if end.Limit != codemode.LimitNone {
			outcome.Limit = end.Limit
		}
	}
}
