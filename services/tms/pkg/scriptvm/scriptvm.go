package scriptvm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultOutputChars = 8_000
	DefaultMemoryBytes = 256 << 20
	DefaultCPU         = 30 * time.Second
	DefaultRecursion   = 200
	DefaultCalls       = 1_000
	MaxRequestBytes    = 16 << 20

	DefaultMaxConcurrent = 8
	DefaultQueueWait     = 10 * time.Second
	ClockResolution      = time.Millisecond
)

var (
	ErrUnavailable = errors.New("scriptvm: the script module is unavailable")
	ErrHost        = errors.New("scriptvm: the host could not answer the script")
	ErrBusy        = errors.New("scriptvm: every script slot on this worker is in use")
)

type Function string

const (
	FunctionRead   = Function("read")
	FunctionGather = Function("gather")
	FunctionResult = Function("result")
)

type Status string

const (
	StatusOK     = Status("ok")
	StatusRaised = Status("raised")
	StatusLimit  = Status("limit")
	StatusFailed = Status("failed")
)

type Limit string

const (
	LimitNone   = Limit("")
	LimitMemory = Limit("memory")
	LimitCPU    = Limit("cpu")
	LimitCalls  = Limit("calls")
	LimitWall   = Limit("wall")
)

type Clock struct {
	Now      time.Time
	Location *time.Location
}

type Limits struct {
	OutputChars int
	MemoryBytes int
	CPU         time.Duration
	Recursion   int
	Calls       int
}

type Program struct {
	Code   string
	Index  []byte
	Inputs []byte
	Clock  Clock
	Limits Limits
}

type Host interface {
	Call(ctx context.Context, function Function, payload []byte) ([]byte, error)
}

type ScriptError struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Traceback string `json:"traceback,omitempty"`
}

type Outcome struct {
	Status          Status
	Limit           Limit
	Error           *ScriptError
	Output          string
	OutputTruncated bool
	Duration        time.Duration
}

type Options struct {
	Logger        *zap.Logger
	MaxConcurrent int
	QueueWait     time.Duration
}

type Runtime struct {
	load      func() (*compiled, error)
	loaded    atomic.Pointer[compiled]
	logger    *zap.Logger
	slots     chan struct{}
	queueWait time.Duration
}

func New(opts Options) *Runtime {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	concurrent := opts.MaxConcurrent
	if concurrent <= 0 {
		concurrent = DefaultMaxConcurrent
	}
	queueWait := opts.QueueWait
	if queueWait <= 0 {
		queueWait = DefaultQueueWait
	}
	r := &Runtime{
		logger:    logger.Named("scriptvm"),
		slots:     make(chan struct{}, concurrent),
		queueWait: queueWait,
	}
	r.load = sync.OnceValues(r.compile)
	return r
}

func (r *Runtime) acquire(ctx context.Context) (func(), error) {
	release := func() { <-r.slots }
	select {
	case r.slots <- struct{}{}:
		return release, nil
	default:
	}

	wait, cancel := context.WithTimeout(ctx, r.queueWait)
	defer cancel()
	select {
	case r.slots <- struct{}{}:
		return release, nil
	case <-wait.Done():
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrBusy
	}
}

func (r *Runtime) Warm() error {
	_, err := r.load()
	return err
}

func (r *Runtime) Close(ctx context.Context) error {
	c := r.loaded.Load()
	if c == nil {
		return nil
	}
	return c.close(ctx)
}

func (r *Runtime) Run(ctx context.Context, program *Program, host Host) (*Outcome, error) {
	c, err := r.load()
	if err != nil {
		return nil, err
	}
	release, err := r.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	started := time.Now()
	in, err := instantiate(ctx, c, program.Limits)
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	defer in.close(ctx)

	outcome, err := r.drive(ctx, in, program, host)
	if outcome != nil {
		outcome.Duration = time.Since(started)
	}
	return outcome, err
}

func (r *Runtime) drive(
	ctx context.Context,
	in *instance,
	program *Program,
	host Host,
) (*Outcome, error) {
	request, err := startRequest(program)
	if err != nil {
		return nil, err
	}
	for {
		step, stepErr := in.step(ctx, request)
		if stepErr != nil {
			return r.stopped(ctx, in, stepErr), nil
		}
		if step.Step == stepDone {
			return step.outcome(), nil
		}
		reply, hostErr := host.Call(ctx, step.Function, []byte(step.Payload))
		if hostErr != nil {
			return hostFailed(ctx, hostErr)
		}
		request, err = resumeRequest(reply)
		if err != nil {
			return nil, err
		}
	}
}

func (r *Runtime) stopped(ctx context.Context, in *instance, err error) *Outcome {
	switch {
	case ctx.Err() != nil:
		return wallOutcome()
	case in.outOfMemory(err):
		return &Outcome{Status: StatusLimit, Limit: LimitMemory}
	default:
		r.logger.Warn("a script stopped the interpreter", zap.Error(err))
		return &Outcome{
			Status: StatusFailed,
			Error: &ScriptError{
				Name:  "Failed",
				Value: "the interpreter stopped while running the script",
			},
		}
	}
}

func hostFailed(ctx context.Context, err error) (*Outcome, error) {
	select {
	case <-ctx.Done():
		return wallOutcome(), nil
	default:
		return nil, errors.Join(ErrHost, err)
	}
}

func wallOutcome() *Outcome {
	return &Outcome{Status: StatusLimit, Limit: LimitWall}
}

func (l Limits) withDefaults() Limits {
	if l.OutputChars <= 0 {
		l.OutputChars = DefaultOutputChars
	}
	if l.MemoryBytes <= 0 {
		l.MemoryBytes = DefaultMemoryBytes
	}
	if l.CPU <= 0 {
		l.CPU = DefaultCPU
	}
	if l.Recursion <= 0 {
		l.Recursion = DefaultRecursion
	}
	if l.Calls <= 0 {
		l.Calls = DefaultCalls
	}
	return l
}
