package scriptvm

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

const (
	exportAlloc  = "scripts_alloc"
	exportCall   = "scripts_call"
	exportFree   = "scripts_free"
	exportOOM    = "scripts_oom_exit_code"
	exportMemory = "memory"

	maxWasmPages   = 16_384
	maxMemoryBytes = 768 << 20
)

var (
	errOutOfBounds   = errors.New("scriptvm: the module named memory outside its linear memory")
	errMissingExport = errors.New("scriptvm: the instance is missing an export")
)

type instance struct {
	module  api.Module
	memory  api.Memory
	alloc   api.Function
	call    api.Function
	free    api.Function
	oomCode uint32
	stack   [2]uint64
}

func instantiate(ctx context.Context, c *compiled, limits Limits) (*instance, error) {
	if limits.MemoryBytes > maxMemoryBytes {
		return nil, fmt.Errorf("scriptvm: a script may use at most %d bytes", maxMemoryBytes)
	}
	module, err := c.runtime.InstantiateModule(
		ctx,
		c.module,
		wazero.NewModuleConfig().
			WithName("").
			WithStartFunctions().
			WithSysNanotime().
			WithSysWalltime().
			WithRandSource(rand.Reader),
	)
	if err != nil {
		return nil, err
	}
	in := &instance{
		module: module,
		memory: module.ExportedMemory(exportMemory),
		alloc:  module.ExportedFunction(exportAlloc),
		call:   module.ExportedFunction(exportCall),
		free:   module.ExportedFunction(exportFree),
	}
	oom := module.ExportedFunction(exportOOM)
	if in.memory == nil || in.alloc == nil || in.call == nil || in.free == nil || oom == nil {
		return nil, errors.Join(errMissingExport, module.Close(context.WithoutCancel(ctx)))
	}
	code, err := oom.Call(ctx)
	if err != nil || len(code) != 1 {
		return nil, errors.Join(errMissingExport, err, module.Close(context.WithoutCancel(ctx)))
	}
	in.oomCode = api.DecodeU32(code[0])
	return in, nil
}

func (in *instance) step(ctx context.Context, request []byte) (*wireStep, error) {
	raw, err := in.exchange(ctx, request)
	if err != nil {
		return nil, err
	}
	return decodeStep(raw)
}

func (in *instance) exchange(ctx context.Context, request []byte) ([]byte, error) {
	requestSize := len(request)
	if requestSize == 0 || requestSize > math.MaxUint32 {
		return nil, fmt.Errorf("%w: a request of %d bytes", errOutOfBounds, requestSize)
	}
	length := uint32(requestSize)
	in.stack[0] = api.EncodeU32(length)
	if err := in.alloc.CallWithStack(ctx, in.stack[:]); err != nil {
		return nil, err
	}
	pointer := api.DecodeU32(in.stack[0])
	if pointer == 0 || !in.memory.Write(pointer, request) {
		return nil, fmt.Errorf("%w: request buffer at %d", errOutOfBounds, pointer)
	}

	in.stack[0], in.stack[1] = api.EncodeU32(pointer), api.EncodeU32(length)
	if err := in.call.CallWithStack(ctx, in.stack[:]); err != nil {
		return nil, err
	}
	packed := in.stack[0]
	address, size := api.DecodeU32(packed>>32), api.DecodeU32(packed)

	view, ok := in.memory.Read(address, size)
	if !ok || size == 0 {
		return nil, fmt.Errorf("%w: response of %d bytes at %d", errOutOfBounds, size, address)
	}
	response := make([]byte, size)
	copy(response, view)

	if err := in.release(ctx, address, size); err != nil {
		return nil, err
	}
	if err := in.release(ctx, pointer, length); err != nil {
		return nil, err
	}
	return response, nil
}

func (in *instance) release(ctx context.Context, pointer, length uint32) error {
	if pointer == 0 || length == 0 {
		return nil
	}
	in.stack[0], in.stack[1] = api.EncodeU32(pointer), api.EncodeU32(length)
	return in.free.CallWithStack(ctx, in.stack[:])
}

func (in *instance) outOfMemory(err error) bool {
	var exit *sys.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == in.oomCode
}

func (in *instance) close(ctx context.Context) {
	_ = in.module.Close(context.WithoutCancel(ctx))
}
