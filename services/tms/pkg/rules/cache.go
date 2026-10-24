package rules

import (
	"github.com/emoss08/trenova/pkg/wasmcache"
	"github.com/tetratelabs/wazero"
)

const (
	CacheDirEnv  = "TRENOVA_RULES_CACHE_DIR"
	cacheDirName = "trenova-rules"
)

func compilationCache(moduleSum string) (wazero.CompilationCache, string, error) {
	return wasmcache.Open(wasmcache.BaseDir(CacheDirEnv, cacheDirName), moduleSum)
}
