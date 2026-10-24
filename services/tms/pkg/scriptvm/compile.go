package scriptvm

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/pkg/wasmcache"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/zap"
)

//go:embed scripts.wasm
var moduleBytes []byte

const (
	CacheDirEnv  = "TRENOVA_SCRIPTS_CACHE_DIR"
	cacheDirName = "trenova-scripts"
	wasiModule   = wasi_snapshot_preview1.ModuleName
)

var (
	errABI  = errors.New("scriptvm: the module does not implement the scripts ABI")
	timeNow = time.Now
)

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
	exportOOM:  {results: []api.ValueType{api.ValueTypeI32}},
}

func moduleSum() string {
	sum := sha256.Sum256(moduleBytes)
	return hex.EncodeToString(sum[:])
}

func (r *Runtime) compile() (*compiled, error) {
	ctx := context.Background()
	cache, dir, cacheErr := wasmcache.Open(
		wasmcache.BaseDir(CacheDirEnv, cacheDirName),
		moduleSum(),
	)
	if cacheErr != nil {
		r.logger.Warn("compiling the script module without an on-disk cache", zap.Error(cacheErr))
	}

	config := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(maxWasmPages).
		WithCompilationCache(cache)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)

	started := time.Now()
	module, err := runtime.CompileModule(ctx, moduleBytes)
	if err == nil {
		err = checkABI(module)
	}
	if err == nil {
		_, err = wasi_snapshot_preview1.Instantiate(ctx, runtime)
	}
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("%w: %w", ErrUnavailable, err),
			runtime.Close(ctx),
			cache.Close(ctx),
		)
	}
	r.logger.Debug("script module compiled",
		zap.String("cacheDir", dir), zap.Duration("took", time.Since(started)))
	return &compiled{runtime: runtime, module: module, cache: cache}, nil
}

func checkABI(module wazero.CompiledModule) error {
	for _, imported := range module.ImportedFunctions() {
		from, name, _ := imported.Import()
		if from != wasiModule {
			return fmt.Errorf("%w: it imports %s.%s", errABI, from, name)
		}
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
