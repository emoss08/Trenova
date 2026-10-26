package wasmmodule

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/pkg/wasmcache"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

const (
	wasmPageBytes = 64 * 1024
	maxWasmPages  = 65536
)

var errABI = errors.New("the module does not implement the module ABI")

type compiled struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
	cache   wazero.CompilationCache
}

func (c *compiled) close(ctx context.Context) error {
	return errors.Join(c.runtime.Close(ctx), c.cache.Close(ctx))
}

type exportSignature struct {
	params  []api.ValueType
	results []api.ValueType
}

func abiExports(name string) map[string]exportSignature {
	return map[string]exportSignature{
		exportName(name, exportAlloc): {
			params:  []api.ValueType{api.ValueTypeI32},
			results: []api.ValueType{api.ValueTypeI32},
		},
		exportName(name, exportCall): {
			params:  []api.ValueType{api.ValueTypeI32, api.ValueTypeI32},
			results: []api.ValueType{api.ValueTypeI64},
		},
		exportName(name, exportFree): {
			params: []api.ValueType{api.ValueTypeI32, api.ValueTypeI32},
		},
	}
}

func memoryLimitPages(limitBytes uint64) uint32 {
	if limitBytes == 0 {
		limitBytes = DefaultMemoryLimitBytes
	}
	pages := (limitBytes + wasmPageBytes - 1) / wasmPageBytes
	return uint32(min(pages, maxWasmPages))
}

func moduleSum(module []byte) string {
	sum := sha256.Sum256(module)
	return hex.EncodeToString(sum[:])
}

func (r *Runtime) compile() (*compiled, error) {
	ctx := context.Background()
	logger := r.hooks().logger

	cache, dir, cacheErr := wasmcache.Open(
		wasmcache.BaseDir(r.spec.CacheDirEnv, r.spec.CacheDirName),
		r.sum,
	)
	if cacheErr != nil {
		logger.Warn("compiling without an on-disk cache", zap.Error(cacheErr))
	}

	config := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(r.limitPages).
		WithCompilationCache(cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)

	module, err := runtime.CompileModule(ctx, r.spec.Bytes)
	if err == nil {
		err = checkABI(module, r.spec.Name)
	}
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("%w: %w", ErrUnavailable, err),
			runtime.Close(ctx),
			cache.Close(ctx),
		)
	}
	logger.Debug("module compiled", zap.String("cacheDir", dir))
	return &compiled{runtime: runtime, module: module, cache: cache}, nil
}

func checkABI(module wazero.CompiledModule, name string) error {
	if imports := module.ImportedFunctions(); len(imports) > 0 {
		return fmt.Errorf("%w: it imports %d functions", errABI, len(imports))
	}
	if _, ok := module.ExportedMemories()[exportMemory]; !ok {
		return fmt.Errorf("%w: no exported memory", errABI)
	}
	exported := module.ExportedFunctions()
	for export, want := range abiExports(name) {
		definition, ok := exported[export]
		if !ok {
			return fmt.Errorf("%w: no export %s", errABI, export)
		}
		if !slices.Equal(definition.ParamTypes(), want.params) ||
			!slices.Equal(definition.ResultTypes(), want.results) {
			return fmt.Errorf("%w: %s has the wrong signature", errABI, export)
		}
	}
	return nil
}
