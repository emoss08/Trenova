package wasmmodule

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

const (
	CodeMalformed    = "malformed"
	CodeUnknownOp    = "unknown_op"
	CodeInvalidInput = "invalid_input"
	CodeTooLarge     = "too_large"

	DefaultRecycleMemoryBytes = 64 * 1024 * 1024
	DefaultMemoryLimitBytes   = 256 * 1024 * 1024
	MaxRequestBytes           = 64 * 1024 * 1024
)

var (
	ErrUnavailable = errors.New("module unavailable")
	ErrContextDone = errors.New("module call context ended")
	ErrTrap        = errors.New("module trapped")
	ErrRequest     = errors.New("module refused the request")

	errClosed = errors.New("the runtime is closed")
)

type Spec struct {
	Name         string
	Bytes        []byte
	VersionOp    string
	CacheDirEnv  string
	CacheDirName string
}

type RequestError struct {
	Module string
	Op     string
	Code   string
	Detail string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s: %s: %s", e.Module, e.Op, e.Code, e.Detail)
}

func (e *RequestError) Is(target error) bool {
	return target == ErrRequest
}

type Options struct {
	PoolSize           int
	RecycleMemoryBytes uint64
	MemoryLimitBytes   uint64
}

type VersionInfo struct {
	Version    string   `json:"version"`
	Operations []string `json:"operations"`
}

type Runtime struct {
	spec         Spec
	sum          string
	load         func() (*compiled, error)
	tokens       chan struct{}
	idle         chan *instance
	recycleBytes uint64
	limitPages   uint32
	instrument   atomic.Value
	started      atomic.Bool
	closed       atomic.Bool
}

func New(spec Spec, opts Options) *Runtime {
	size := opts.PoolSize
	if size <= 0 {
		size = runtime.GOMAXPROCS(0)
	}
	recycle := opts.RecycleMemoryBytes
	if recycle == 0 {
		recycle = DefaultRecycleMemoryBytes
	}
	r := &Runtime{
		spec:         spec,
		sum:          moduleSum(spec.Bytes),
		tokens:       make(chan struct{}, size),
		idle:         make(chan *instance, size),
		recycleBytes: recycle,
		limitPages:   memoryLimitPages(opts.MemoryLimitBytes),
	}
	r.instrument.Store(instrumentation{logger: zap.NewNop(), observer: nopObserver{}})
	r.load = sync.OnceValues(r.compile)
	return r
}

func (r *Runtime) Instrument(logger *zap.Logger, observer Observer) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if observer == nil {
		observer = nopObserver{}
	}
	r.instrument.Store(instrumentation{logger: logger, observer: observer})
}

func (r *Runtime) PoolSize() int {
	return cap(r.tokens)
}

func (r *Runtime) Warm(ctx context.Context) error {
	_, err := r.Version(ctx)
	return err
}

func (r *Runtime) Version(ctx context.Context) (VersionInfo, error) {
	var info VersionInfo
	err := r.Call(ctx, r.spec.VersionOp, nil, &info)
	return info, err
}

func (r *Runtime) Name() string {
	return r.spec.Name
}

func (r *Runtime) Call(ctx context.Context, op string, input, output any) error {
	var encoded []byte
	if input != nil {
		var err error
		if encoded, err = sonic.Marshal(input); err != nil {
			return fmt.Errorf("%s: %s: encoding the input: %w", r.spec.Name, op, err)
		}
	}
	result, err := r.CallRaw(ctx, op, encoded)
	if err != nil || output == nil {
		return err
	}
	if err = sonic.Unmarshal(result, output); err != nil {
		return fmt.Errorf("%s: %s: decoding the output: %w", r.spec.Name, op, err)
	}
	return nil
}

func (r *Runtime) CallRaw(ctx context.Context, op string, input []byte) ([]byte, error) {
	started := time.Now()
	result, outcome, err := r.run(ctx, op, input)
	label := opLabel(op)
	var requestErr *RequestError
	if errors.As(err, &requestErr) && requestErr.Code == CodeUnknownOp {
		label = unknownOpLabel
	}
	r.hooks().observer.CallFinished(label, outcome, time.Since(started))
	return result, err
}

func (r *Runtime) Close(ctx context.Context) error {
	if r.closed.Swap(true) {
		return nil
	}
	errs := r.drainIdle(ctx)
	if r.started.Load() {
		if c, err := r.load(); err == nil {
			errs = append(errs, c.close(ctx))
		}
	}
	return errors.Join(errs...)
}

func (r *Runtime) run(
	ctx context.Context,
	op string,
	input []byte,
) (result []byte, outcome string, err error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, OutcomeContextDone, fmt.Errorf(
			"%s: %s: %w: %w",
			r.spec.Name,
			op,
			ErrContextDone,
			ctxErr,
		)
	}
	request, err := r.envelope(op, input)
	if err != nil {
		return nil, OutcomeRequestError, err
	}

	err = r.withInstance(ctx, func(in *instance) error {
		response, callErr := in.exchange(ctx, request)
		if callErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				in.broken = DiscardContextDone
				return fmt.Errorf("%w: %w", ErrContextDone, ctxErr)
			}
			in.broken = DiscardTrap
			r.hooks().logger.Error(
				"the module trapped; discarding the instance",
				zap.String("op", op),
				zap.Error(callErr),
			)
			return fmt.Errorf("%w: %w", ErrTrap, callErr)
		}
		if in.memoryBytes() > r.recycleBytes {
			in.broken = DiscardMemory
		}
		var decodeErr error
		result, decodeErr = r.decodeResponse(op, response)
		var requestErr *RequestError
		if decodeErr != nil && !errors.As(decodeErr, &requestErr) {
			in.broken = DiscardTrap
			r.hooks().logger.Error(
				"the module answered with an unreadable response; discarding the instance",
				zap.String("op", op),
				zap.Error(decodeErr),
			)
		}
		return decodeErr
	})

	var requestErr *RequestError
	switch {
	case err == nil:
		return result, OutcomeOK, nil
	case errors.As(err, &requestErr):
		return nil, OutcomeRequestError, err
	case errors.Is(err, ErrContextDone):
		return nil, OutcomeContextDone, fmt.Errorf("%s: %s: %w", r.spec.Name, op, err)
	case errors.Is(err, ErrUnavailable):
		return nil, OutcomeUnavailable, fmt.Errorf("%s: %s: %w", r.spec.Name, op, err)
	default:
		return nil, OutcomeTrap, fmt.Errorf("%s: %s: %w", r.spec.Name, op, err)
	}
}

func (r *Runtime) withInstance(ctx context.Context, fn func(*instance) error) error {
	if r.closed.Load() {
		return fmt.Errorf("%w: %w", ErrUnavailable, errClosed)
	}
	r.started.Store(true)
	c, err := r.load()
	if err != nil {
		return err
	}
	in, err := r.acquire(ctx, c)
	if err != nil {
		return err
	}
	defer r.release(ctx, in)
	return fn(in)
}

func (r *Runtime) acquire(ctx context.Context, c *compiled) (*instance, error) {
	select {
	case r.tokens <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: waiting for an instance: %w", ErrContextDone, ctx.Err())
	}
	select {
	case in := <-r.idle:
		r.hooks().observer.InstanceAcquired()
		return in, nil
	default:
	}
	in, err := instantiate(ctx, c, r.spec.Name)
	if err != nil {
		<-r.tokens
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%w: instantiating: %w", ErrContextDone, ctxErr)
		}
		r.hooks().logger.Error("could not instantiate the module", zap.Error(err))
		return nil, fmt.Errorf("%w: instantiating: %w", ErrUnavailable, err)
	}
	r.hooks().observer.InstanceAcquired()
	return in, nil
}

func (r *Runtime) release(ctx context.Context, in *instance) {
	hooks := r.hooks()
	hooks.observer.InstanceReleased()
	switch {
	case in.broken != "":
		hooks.observer.InstanceDiscarded(in.broken)
		r.closeInstance(ctx, in)
	case r.closed.Load():
		r.closeInstance(ctx, in)
	default:
		r.idle <- in
	}
	<-r.tokens
}

func (r *Runtime) closeInstance(ctx context.Context, in *instance) {
	if err := in.close(ctx); err != nil {
		r.hooks().logger.Warn("closing a discarded instance", zap.Error(err))
	}
}

func (r *Runtime) drainIdle(ctx context.Context) []error {
	var errs []error
	for {
		select {
		case in := <-r.idle:
			errs = append(errs, in.close(ctx))
		default:
			return errs
		}
	}
}

func (r *Runtime) hooks() instrumentation {
	hooks, _ := r.instrument.Load().(instrumentation)
	return hooks
}
