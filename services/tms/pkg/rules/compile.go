package rules

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

//go:embed rules.wasm
var moduleBytes []byte

const (
	wasmPageBytes = 64 * 1024
	maxWasmPages  = 65536
)

var errABI = errors.New("rules: the module does not implement the rules ABI")

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

var abiExports = map[string]exportSignature{
	exportAlloc: {
		params:  []api.ValueType{api.ValueTypeI32},
		results: []api.ValueType{api.ValueTypeI32},
	},
	exportCall: {
		params:  []api.ValueType{api.ValueTypeI32, api.ValueTypeI32},
		results: []api.ValueType{api.ValueTypeI64},
	},
	exportFree: {params: []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}},
}

func memoryLimitPages(limitBytes uint64) uint32 {
	if limitBytes == 0 {
		limitBytes = DefaultMemoryLimitBytes
	}
	pages := (limitBytes + wasmPageBytes - 1) / wasmPageBytes
	return uint32(min(pages, maxWasmPages))
}

func moduleSum() string {
	sum := sha256.Sum256(moduleBytes)
	return hex.EncodeToString(sum[:])
}

func (r *Runtime) compile() (*compiled, error) {
	ctx := context.Background()
	logger := r.hooks().logger

	cache, dir, cacheErr := compilationCache(moduleSum())
	if cacheErr != nil {
		logger.Warn("rules: compiling without an on-disk cache", zap.Error(cacheErr))
	}

	config := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(r.limitPages).
		WithCompilationCache(cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)

	module, err := runtime.CompileModule(ctx, moduleBytes)
	if err == nil {
		err = checkABI(module)
	}
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("%w: %w", ErrUnavailable, err),
			runtime.Close(ctx),
			cache.Close(ctx),
		)
	}
	logger.Debug("rules: module compiled", zap.String("cacheDir", dir))
	return &compiled{runtime: runtime, module: module, cache: cache}, nil
}

func checkABI(module wazero.CompiledModule) error {
	if imports := module.ImportedFunctions(); len(imports) > 0 {
		return fmt.Errorf("%w: it imports %d functions", errABI, len(imports))
	}
	if _, ok := module.ExportedMemories()[exportMemory]; !ok {
		return fmt.Errorf("%w: no exported memory", errABI)
	}
	exported := module.ExportedFunctions()
	for name, want := range abiExports {
		definition, ok := exported[name]
		if !ok {
			return fmt.Errorf("%w: no export %s", errABI, name)
		}
		if !slices.Equal(definition.ParamTypes(), want.params) ||
			!slices.Equal(definition.ResultTypes(), want.results) {
			return fmt.Errorf("%w: %s has the wrong signature", errABI, name)
		}
	}
	return nil
}
