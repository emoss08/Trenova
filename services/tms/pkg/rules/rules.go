package rules

import (
	"context"
	_ "embed"
	"sync"

	"github.com/emoss08/trenova/pkg/wasmmodule"
)

//go:embed rules.wasm
var moduleBytes []byte

const (
	OpVersion = "rules.version"

	CacheDirEnv = "TRENOVA_RULES_CACHE_DIR"
)

var spec = wasmmodule.Spec{
	Name:         "rules",
	Bytes:        moduleBytes,
	VersionOp:    OpVersion,
	CacheDirEnv:  CacheDirEnv,
	CacheDirName: "trenova-rules",
}

type Runtime struct {
	*wasmmodule.Runtime
}

var defaultRuntime = sync.OnceValue(func() *Runtime {
	return New(wasmmodule.Options{})
})

func New(opts wasmmodule.Options) *Runtime {
	return &Runtime{Runtime: wasmmodule.New(spec, opts)}
}

func Default() *Runtime {
	return defaultRuntime()
}

func Call(ctx context.Context, op string, input, output any) error {
	return Default().Call(ctx, op, input, output)
}

func CallRaw(ctx context.Context, op string, input []byte) ([]byte, error) {
	return Default().CallRaw(ctx, op, input)
}

func Version(ctx context.Context) (wasmmodule.VersionInfo, error) {
	return Default().Version(ctx)
}
