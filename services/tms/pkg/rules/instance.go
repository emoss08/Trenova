package rules

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	exportAlloc  = "rules_alloc"
	exportCall   = "rules_call"
	exportFree   = "rules_free"
	exportMemory = "memory"
)

var (
	errOutOfBounds   = errors.New("rules: the module named memory outside its linear memory")
	errMissingExport = errors.New("rules: the instance is missing an export")
)

type instance struct {
	module api.Module
	memory api.Memory
	alloc  api.Function
	call   api.Function
	free   api.Function
	stack  [2]uint64
	broken string
}

func instantiate(ctx context.Context, c *compiled) (*instance, error) {
	module, err := c.runtime.InstantiateModule(
		ctx,
		c.module,
		wazero.NewModuleConfig().WithName("").WithStartFunctions(),
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
	if in.memory == nil || in.alloc == nil || in.call == nil || in.free == nil {
		return nil, errors.Join(errMissingExport, module.Close(context.WithoutCancel(ctx)))
	}
	return in, nil
}

func (in *instance) exchange(ctx context.Context, request []byte) ([]byte, error) {
	requestSize := len(request)
	if requestSize < 0 || requestSize > math.MaxUint32 {
		return nil, fmt.Errorf("%w: a request of %d bytes", errOutOfBounds, requestSize)
	}
	length := uint32(requestSize)
	var pointer uint32
	if length > 0 {
		in.stack[0] = api.EncodeU32(length)
		if err := in.alloc.CallWithStack(ctx, in.stack[:]); err != nil {
			return nil, err
		}
		pointer = api.DecodeU32(in.stack[0])
		if pointer == 0 || !in.memory.Write(pointer, request) {
			return nil, fmt.Errorf("%w: request buffer at %d", errOutOfBounds, pointer)
		}
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

func (in *instance) memoryBytes() uint64 {
	return uint64(in.memory.Size())
}

func (in *instance) close(ctx context.Context) error {
	return in.module.Close(context.WithoutCancel(ctx))
}
