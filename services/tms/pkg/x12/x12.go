package x12

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/wasmmodule"
)

//go:embed edi.wasm
var moduleBytes []byte

const (
	OpVersion     = "edi.version"
	OpRead        = "x12.read"
	OpAcknowledge = "x12.acknowledge"
	OpWrite       = "x12.write"
	OpGuides      = "x12.guides"
	OpGuide       = "x12.guide"
	OpCheckGuide  = "x12.checkGuide"

	CacheDirEnv = "TRENOVA_EDI_CACHE_DIR"
)

var spec = wasmmodule.Spec{
	Name:         "edi",
	Bytes:        moduleBytes,
	VersionOp:    OpVersion,
	CacheDirEnv:  CacheDirEnv,
	CacheDirName: "trenova-edi",
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

func (r *Runtime) Read(ctx context.Context, req *ReadRequest) (*Reading, error) {
	var reading Reading
	if err := r.Call(ctx, OpRead, req, &reading); err != nil {
		return nil, err
	}
	return &reading, nil
}

func (r *Runtime) Acknowledge(
	ctx context.Context,
	req *AcknowledgeRequest,
) ([]Acknowledgment, error) {
	var result struct {
		Acknowledgments []Acknowledgment `json:"acknowledgments"`
	}
	if err := r.Call(ctx, OpAcknowledge, req, &result); err != nil {
		return nil, err
	}
	if len(result.Acknowledgments) != len(req.Acknowledgments) {
		return nil, fmt.Errorf(
			"edi: %s: %d acknowledgments for %d groups",
			OpAcknowledge,
			len(result.Acknowledgments),
			len(req.Acknowledgments),
		)
	}
	return result.Acknowledgments, nil
}

func (r *Runtime) Write(ctx context.Context, req *WriteRequest) (*Written, error) {
	var written Written
	if err := r.Call(ctx, OpWrite, req, &written); err != nil {
		return nil, err
	}
	return &written, nil
}

func (r *Runtime) Guides(ctx context.Context) ([]GuideSummary, error) {
	var result struct {
		Guides []GuideSummary `json:"guides"`
	}
	if err := r.Call(ctx, OpGuides, nil, &result); err != nil {
		return nil, err
	}
	return result.Guides, nil
}

func (r *Runtime) Guide(ctx context.Context, id string) (sonic.NoCopyRawMessage, error) {
	input, err := sonic.Marshal(guideRequest{ID: id})
	if err != nil {
		return nil, fmt.Errorf("edi: %s: encoding the input: %w", OpGuide, err)
	}
	return r.CallRaw(ctx, OpGuide, input)
}

func (r *Runtime) CheckGuide(
	ctx context.Context,
	guide sonic.NoCopyRawMessage,
) ([]GuideProblem, error) {
	var result struct {
		Problems []GuideProblem `json:"problems"`
	}
	if err := r.Call(ctx, OpCheckGuide, checkGuideRequest{Guide: guide}, &result); err != nil {
		return nil, err
	}
	return result.Problems, nil
}

type guideRequest struct {
	ID string `json:"id"`
}

type checkGuideRequest struct {
	Guide sonic.NoCopyRawMessage `json:"guide"`
}
