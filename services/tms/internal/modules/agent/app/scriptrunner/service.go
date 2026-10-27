package scriptrunner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/scriptvm"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ scriptports.Runner = (*Service)(nil)

const (
	scriptFailure = "The script could not be run. This was not a fault in the code; answer with " +
		"the discrete reads instead."
	scriptVMUnavailable = "The script interpreter is not available right now. Answer with the " +
		"discrete reads."
	scriptVMBusy = "Too many scripts are running right now. Answer with the discrete reads, or " +
		"run the script again in a moment."
	warmupTimeout = 2 * time.Minute
)

var errScriptInputs = errors.New("script inputs")

type Params struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

type Service struct {
	vm *scriptvm.Runtime
	l  *zap.Logger
}

func New(p Params) *Service {
	svc := NewRunner(p.Logger)
	p.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return svc.vm.Close(ctx)
		},
	})

	return svc
}

func NewRunner(logger *zap.Logger) *Service {
	logger = logger.Named("service.script-runner")
	return &Service{vm: scriptvm.New(scriptvm.Options{Logger: logger}), l: logger}
}

func (s *Service) Warm() {
	started := time.Now()
	if err := s.vm.Warm(); err != nil {
		s.l.Error("the script interpreter failed to load; every agent script will fail",
			zap.Error(err))
		return
	}
	s.l.Info("script interpreter ready", zap.Duration("took", time.Since(started)))
}

func (s *Service) Run(
	ctx context.Context,
	req *scriptports.Request,
) (*scriptports.Outcome, error) {
	if err := checkScript(req); err != nil {
		return nil, err
	}
	inputs, err := encodeInputs(req.Inputs)
	if err != nil {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"The inputs must encode as JSON in at most %d bytes.", codemode.MaxInputBytes,
		))
	}
	index, err := sonic.Marshal(req.Broker.Index())
	if err != nil {
		return nil, fmt.Errorf("encode the script's reads: %w", err)
	}
	datasets, err := sonic.Marshal(datasetIndex(req.Datasets))
	if err != nil {
		return nil, fmt.Errorf("encode the script's datasets: %w", err)
	}

	limits := codemode.DefaultLimits()
	req.Broker.Bound(&limits)

	runCtx, cancel := context.WithTimeout(ctx, limits.Wall)
	defer cancel()

	host := newHost(req.Broker, &limits)
	host.datasets = req.Datasets
	result, err := s.vm.Run(runCtx, &scriptvm.Program{
		Code:     req.Code,
		Index:    index,
		Inputs:   inputs,
		Datasets: datasets,
		Earlier:  req.Earlier,
		Clock:    scriptvm.Clock{Now: time.Now(), Location: location(req.Timezone)},
		Limits: scriptvm.Limits{
			OutputChars: limits.OutputChars,
			MemoryBytes: int(limits.MemoryBytes),
			CPU:         limits.Wall,
			Recursion:   scriptvm.DefaultRecursion,
			Calls:       limits.Calls + limits.Concurrent,
		},
	}, host)
	if err != nil {
		return s.failed(&limits, err), nil
	}

	return host.outcome(result, &limits, req.Broker.Summary().Usage.Hit), nil
}

func (s *Service) failed(limits *codemode.Limits, err error) *scriptports.Outcome {
	failure := scriptFailure
	switch {
	case errors.Is(err, scriptvm.ErrUnavailable):
		failure = scriptVMUnavailable
	case errors.Is(err, scriptvm.ErrBusy):
		failure = scriptVMBusy
	}
	s.l.Warn("a script could not be run", zap.Error(err))

	return &scriptports.Outcome{
		Status:  scriptports.Failed,
		Failure: failure,
		Limits:  *limits,
	}
}

func checkScript(req *scriptports.Request) error {
	switch {
	case strings.TrimSpace(req.Code) == "":
		return errortypes.NewBusinessError("The script is empty. Send the Python to run.")
	case utf8.RuneCountInString(req.Code) > codemode.MaxCodeChars:
		return errortypes.NewBusinessError(fmt.Sprintf(
			"A script is at most %d characters. Make it shorter.", codemode.MaxCodeChars,
		))
	case req.Broker == nil:
		return errortypes.NewBusinessError("a script needs a broker to read through")
	}
	if len(req.Inputs) > codemode.MaxInputKeys {
		return errortypes.NewBusinessError(fmt.Sprintf(
			"A script takes at most %d inputs.", codemode.MaxInputKeys,
		))
	}

	return nil
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

func location(timezone string) *time.Location {
	if timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}

	return loc
}

type host struct {
	broker   codemode.Broker
	limits   *codemode.Limits
	datasets []scriptports.Dataset
	seq      atomic.Int64

	mu       sync.Mutex
	result   sonic.NoCopyRawMessage
	resultIn bool
	limit    codemode.Limit
	shown    analysis.Display
	notShown int
}

func newHost(broker codemode.Broker, limits *codemode.Limits) *host {
	return &host{broker: broker, limits: limits}
}

type scriptCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

func (h *host) Call(
	ctx context.Context,
	function scriptvm.Function,
	payload []byte,
) ([]byte, error) {
	switch function {
	case scriptvm.FunctionRead:
		var call scriptCall
		if err := sonic.Unmarshal(payload, &call); err != nil {
			return sonic.Marshal(invalid("the call could not be read"))
		}
		return sonic.Marshal(h.read(ctx, h.next(), call, len(payload)))
	case scriptvm.FunctionGather:
		var calls []sonic.NoCopyRawMessage
		if err := sonic.Unmarshal(payload, &calls); err != nil {
			return sonic.Marshal(invalid("the calls could not be read"))
		}
		return sonic.Marshal(h.gather(ctx, calls))
	case scriptvm.FunctionResult:
		return sonic.Marshal(h.keepResult(payload))
	case scriptvm.FunctionRows:
		return h.rows(payload)
	case scriptvm.FunctionShow:
		return sonic.Marshal(h.show(payload))
	default:
		return nil, fmt.Errorf("a script asked for an unknown function %q", function)
	}
}

func (h *host) next() int {
	return int(h.seq.Add(1))
}

func (h *host) read(ctx context.Context, seq int, call scriptCall, size int) codemode.Reply {
	if size > h.limits.FrameBytes {
		h.note(codemode.LimitFrame)
		return invalid("the call's arguments are too large")
	}
	started := time.Now()
	reply := h.broker.Read(ctx, codemode.Call{Seq: seq, Tool: call.Tool, Arguments: call.Args})
	h.broker.NoteReply(seq, time.Since(started).Milliseconds())

	return reply
}

func (h *host) gather(ctx context.Context, raw []sonic.NoCopyRawMessage) []codemode.Reply {
	replies := make([]codemode.Reply, len(raw))
	seqs := make([]int, len(raw))
	for idx := range raw {
		seqs[idx] = h.next()
	}
	slots := make(chan struct{}, max(h.limits.Concurrent, 1))
	var wg sync.WaitGroup
	for idx := range raw {
		var call scriptCall
		if err := sonic.Unmarshal(raw[idx], &call); err != nil {
			replies[idx] = invalid("the call could not be read")
			continue
		}
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			replies[idx] = h.read(ctx, seqs[idx], call, len(raw[idx]))
		})
	}
	wg.Wait()

	return replies
}

func (h *host) keepResult(payload []byte) codemode.Reply {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.resultIn {
		return invalid("result() may be called once")
	}
	h.resultIn = true
	if len(payload) > h.limits.ResultBytes {
		h.noteLocked(codemode.LimitResult)
		return codemode.Reply{
			Status:  codemode.CallLimit,
			Message: "the result is too large",
			Limit:   codemode.LimitResult,
		}
	}
	h.result = sonic.NoCopyRawMessage(payload)

	return codemode.Reply{Status: codemode.CallOK}
}

func (h *host) note(limit codemode.Limit) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.noteLocked(limit)
}

func (h *host) noteLocked(limit codemode.Limit) {
	if h.limit == codemode.LimitNone {
		h.limit = limit
	}
}

func invalid(message string) codemode.Reply {
	return codemode.Reply{Status: codemode.CallInvalid, Message: message}
}

func (h *host) outcome(
	result *scriptvm.Outcome,
	limits *codemode.Limits,
	brokerHit codemode.Limit,
) *scriptports.Outcome {
	h.mu.Lock()
	defer h.mu.Unlock()

	outcome := &scriptports.Outcome{
		Output:          result.Output,
		OutputTruncated: result.OutputTruncated,
		Result:          h.result,
		Limits:          *limits,
		DurationMs:      result.Duration.Milliseconds(),
		Limit:           h.limit,
		Shown:           h.shown,
		NotShown:        h.notShown,
	}
	if result.Error != nil {
		outcome.Error = &codemode.ScriptError{
			Name:      result.Error.Name,
			Value:     result.Error.Value,
			Traceback: result.Error.Traceback,
		}
	}

	switch result.Status {
	case scriptvm.StatusOK:
		outcome.Status = scriptports.Succeeded
	case scriptvm.StatusRaised:
		outcome.Status = scriptports.Errored
	case scriptvm.StatusLimit:
		outcome.Limit = limitOf(result.Limit)
		if result.Limit == scriptvm.LimitWall || result.Limit == scriptvm.LimitCPU {
			outcome.Status = scriptports.TimedOut
		} else {
			outcome.Status = scriptports.Stopped
		}
	case scriptvm.StatusFailed:
		outcome.Status = scriptports.Failed
		outcome.Failure = scriptFailure
		outcome.Error = nil
	default:
		outcome.Status = scriptports.Failed
		outcome.Failure = scriptFailure
		outcome.Error = nil
	}
	if outcome.Limit == codemode.LimitNone && result.OutputTruncated {
		outcome.Limit = codemode.LimitOutput
	}
	if outcome.Limit == codemode.LimitNone {
		outcome.Limit = brokerHit
	}

	return outcome
}

func limitOf(limit scriptvm.Limit) codemode.Limit {
	switch limit {
	case scriptvm.LimitWall:
		return codemode.LimitWall
	case scriptvm.LimitCPU:
		return codemode.LimitCPU
	case scriptvm.LimitMemory:
		return codemode.LimitMemory
	case scriptvm.LimitCalls:
		return codemode.LimitCalls
	case scriptvm.LimitNone:
		return codemode.LimitNone
	default:
		return codemode.LimitNone
	}
}
