package agentruntime

import (
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"go.uber.org/zap"
)

// runtimeBuild is what a test runtime was built from, kept so a test can
// change a part and build it again in place.
type runtimeBuild struct {
	params Params
	tunes  []func(*toolaccess.AccessParams, *tooldispatch.DispatcherParams)
}

var runtimeBuilds sync.Map

// buildRuntime builds a runtime as New does, remembering what it was built
// from.
func buildRuntime(p Params) *Service {
	if p.Logger == nil {
		p.Logger = zap.NewNop()
	}
	b := &runtimeBuild{params: p}
	rt := b.build()
	runtimeBuilds.Store(rt, b)

	return rt
}

func (b *runtimeBuild) build() *Service {
	accessParts, dispatchParts := accessParams(b.params), dispatcherParams(b.params)
	for _, tune := range b.tunes {
		tune(&accessParts, &dispatchParts)
	}

	return assemble(b.params, accessParts, dispatchParts)
}

func buildOf(rt *Service) *runtimeBuild {
	b, ok := runtimeBuilds.Load(rt)
	if !ok {
		panic("runtime was not built by buildRuntime")
	}

	return b.(*runtimeBuild)
}

// paramsOf is what a test runtime was built from.
func paramsOf(rt *Service) Params { return buildOf(rt).params }

// rewire builds rt again, in place, with change made to its dependencies.
func rewire(rt *Service, change func(*Params)) {
	b := buildOf(rt)
	change(&b.params)
	*rt = *b.build()
}

// retune builds rt again, in place, with change made to the parts New
// derives from its dependencies.
func retune(rt *Service, change func(*toolaccess.AccessParams, *tooldispatch.DispatcherParams)) {
	b := buildOf(rt)
	b.tunes = append(b.tunes, change)
	*rt = *b.build()
}
