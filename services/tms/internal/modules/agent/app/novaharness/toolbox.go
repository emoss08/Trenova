package novaharness

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

// toolbox is the reads a task declared, looked up by the name a model calls.
type toolbox struct {
	tools map[string]llmports.NovaTool
	specs []kernelports.ToolSpec
	used  map[string]struct{}
}

func newToolbox(tools []llmports.NovaTool) *toolbox {
	box := &toolbox{
		tools: make(map[string]llmports.NovaTool, len(tools)),
		specs: make([]kernelports.ToolSpec, 0, len(tools)),
		used:  make(map[string]struct{}),
	}
	for _, tool := range tools {
		spec := tool.Spec()
		box.tools[spec.Name] = tool
		box.specs = append(box.specs, spec)
	}

	return box
}

// distinct gives every call in the reply an id no earlier call used, so each
// result pairs with its own call.
func (b *toolbox) distinct(result *llmports.ChatCompletionResult) {
	result.DistinctCallIDs(b.used, func() string { return pulid.MustNew("call_").String() })
}

func (b *toolbox) call(ctx context.Context, call llmports.ToolCall) llmports.NovaToolResult {
	if call.ArgumentsError != "" {
		return failed(fmt.Sprintf("The arguments for %s did not parse: %s", call.Name, call.ArgumentsError))
	}
	tool, ok := b.tools[call.Name]
	if !ok {
		return failed(fmt.Sprintf("There is no tool named %s. Use one of the tools listed.", call.Name))
	}

	return tool.Call(ctx, call.Arguments)
}

func failed(content string) llmports.NovaToolResult {
	return llmports.NovaToolResult{Content: content, Failed: true}
}
