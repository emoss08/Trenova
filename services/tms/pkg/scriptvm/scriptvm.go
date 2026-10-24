package scriptvm

import (
	"context"
	"errors"
	"sync"
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
)

var (
	ErrUnavailable = errors.New("scriptvm: the script module is unavailable")
	ErrHost        = errors.New("scriptvm: the host could not answer the script")
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

type Runtime struct {
	load   func() (*compiled, error)
	logger *zap.Logger
}

func New(logger *zap.Logger) *Runtime {
	if logger == nil {
		logger = zap.NewNop()
	}
	r := &Runtime{logger: logger.Named("scriptvm")}
	r.load = sync.OnceValues(r.compile)
	return r
}

func (r *Runtime) Warm() error {
	_, err := r.load()
	return err
}

func (r *Runtime) Close(ctx context.Context) error {
	c, err := r.load()
	if err != nil {
		return nil
	}
	return c.close(ctx)
}

func (r *Runtime) Run(ctx context.Context, program *Program, host Host) (*Outcome, error) {
	c, err := r.load()
	if err != nil {
		return nil, err
	}
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
			if ctx.Err() != nil {
				return wallOutcome(), nil
			}
			return nil, errors.Join(ErrHost, hostErr)
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
