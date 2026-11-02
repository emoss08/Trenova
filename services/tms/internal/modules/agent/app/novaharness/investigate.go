package novaharness

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	answerTool       = "answer"
	defaultReadCalls = 3
)

const readsGuide = `You may read more before you answer, with the tools listed. Read only what the material you were shown does not settle. Give your answer by calling the ` + answerTool + ` tool once, with its arguments in exactly the shape it asks for.`

// investigate runs a task that may read before it answers. Each call either
// reads or answers through the answer tool; when the calls run out, or an
// answer does not decode, one structured call answers with what was read.
func (h *Harness) investigate(
	ctx context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	run := newInvestigation(task)
	for call := 1; call < run.maxCalls; call++ {
		result, err := h.router.CompleteChat(ctx, run.request())
		if err != nil {
			return nil, fmt.Errorf("nova %s: %w", task.Name, err)
		}
		if text, answered := answerOf(result); answered {
			if llmports.DecodeAnswer(task.Name, text, out) == nil {
				return answerResult(result, text), nil
			}

			break
		}
		if len(result.ToolCalls) == 0 {
			break
		}
		run.read(ctx, result)
	}

	result, err := h.router.CompleteStructured(ctx, run.closing())
	if err != nil {
		return nil, fmt.Errorf("nova %s: %w", task.Name, err)
	}

	return result, llmports.DecodeAnswer(task.Name, result.Text, out)
}

type investigation struct {
	task     *llmports.NovaTask
	box      *toolbox
	specs    []kernelports.ToolSpec
	messages []llmports.Message
	sections []llmports.ContextSection
	maxCalls int
}

func newInvestigation(task *llmports.NovaTask) *investigation {
	box := newToolbox(task.Reads)
	maxCalls := task.MaxCalls
	if maxCalls <= 0 {
		maxCalls = defaultReadCalls
	}

	return &investigation{
		task: task,
		box:  box,
		specs: append(box.specs[:len(box.specs):len(box.specs)], kernelports.ToolSpec{
			Name:        answerTool,
			Description: "Give your answer. Call it once, when you are ready to answer.",
			Parameters:  task.Schema,
		}),
		messages: []llmports.Message{{
			Role:    llmports.RoleUser,
			Content: task.Context.Text(),
			Files:   task.Files,
		}},
		maxCalls: maxCalls,
	}
}

func (r *investigation) request() *llmports.ChatCompletionRequest {
	return &llmports.ChatCompletionRequest{
		TenantInfo:  r.task.TenantInfo,
		Task:        r.task.Name,
		System:      identity + "\n\n" + r.task.Instructions + "\n\n" + readsGuide,
		Messages:    r.messages,
		Tools:       r.specs,
		MaxTokens:   r.task.MaxTokens,
		Reasoning:   r.task.Reasoning,
		Attribution: attribution(r.task),
		Nova:        true,
	}
}

func (r *investigation) read(ctx context.Context, result *llmports.ChatCompletionResult) {
	r.box.distinct(result)
	r.messages = append(r.messages, llmports.Message{
		Role:          llmports.RoleAssistant,
		Content:       result.Text,
		ToolCalls:     result.ToolCalls,
		Reasoning:     result.Reasoning,
		ReplyID:       result.ProviderMessageID,
		ReplyProvider: result.ProviderID,
	})
	for _, call := range result.ToolCalls {
		read := r.box.call(ctx, call)
		r.messages = append(r.messages, llmports.Message{
			Role:       llmports.RoleTool,
			Content:    read.Content,
			ToolCallID: call.ID,
			ToolName:   call.Name,
			IsError:    read.Failed,
		})
		if read.Failed {
			continue
		}
		arguments, _ := sonic.MarshalString(call.Arguments)
		r.sections = append(r.sections, llmports.ContextSection{
			Title:   "What you read with " + call.Name + " " + arguments,
			Content: read.Content,
		})
	}
}

// closing is the task as a single structured call, with what was read after
// the task's own context.
func (r *investigation) closing() *llmports.StructuredCompletionRequest {
	req := request(r.task)
	req.Context.Sections = append(
		append([]llmports.ContextSection(nil), r.task.Context.Sections...),
		r.sections...,
	)

	return req
}

func answerOf(result *llmports.ChatCompletionResult) (string, bool) {
	for _, call := range result.ToolCalls {
		if call.Name != answerTool {
			continue
		}
		if call.ArgumentsError != "" {
			return "", true
		}
		text, err := sonic.MarshalString(call.Arguments)
		if err != nil {
			return "", true
		}

		return text, true
	}

	return "", false
}

func answerResult(
	result *llmports.ChatCompletionResult,
	text string,
) *llmports.StructuredCompletionResult {
	return &llmports.StructuredCompletionResult{
		Text:             text,
		ModelIdentifier:  result.ModelIdentifier,
		InputTokens:      result.InputTokens,
		OutputTokens:     result.OutputTokens,
		ProviderID:       result.ProviderID,
		ProviderKind:     result.ProviderKind,
		LatencyMs:        result.LatencyMs,
		CostUSD:          result.CostUSD,
		CacheReadTokens:  result.CacheReadTokens,
		CacheWriteTokens: result.CacheWriteTokens,
	}
}
