package novaharness

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const defaultMaxCalls = 4

var _ llmports.TaskConverser = (*Harness)(nil)

// Converse answers a question from a few reads, streaming the answer as it is
// written. It runs on a worker and calls the router itself; each model call
// may ask for reads, and the last is asked to answer with what it has.
func (h *Harness) Converse(
	ctx context.Context,
	talk *llmports.NovaConversation,
	events llmports.NovaEvents,
) (*llmports.NovaAnswer, error) {
	if h == nil || h.router == nil {
		return nil, llmports.ErrNoProviderConfigured
	}
	available, err := h.Available(ctx, talk.Name, talk.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, llmports.ErrNoProviderConfigured
	}

	run := &conversation{
		harness:  h,
		talk:     talk,
		events:   events,
		box:      newToolbox(talk.Tools),
		cited:    make(map[string]struct{}),
		answer:   &llmports.NovaAnswer{},
		messages: []llmports.Message{{Role: llmports.RoleUser, Content: question(talk)}},
	}

	return run.answerQuestion(ctx)
}

type conversation struct {
	harness  *Harness
	talk     *llmports.NovaConversation
	events   llmports.NovaEvents
	box      *toolbox
	cited    map[string]struct{}
	answer   *llmports.NovaAnswer
	messages []llmports.Message
}

func (c *conversation) answerQuestion(ctx context.Context) (*llmports.NovaAnswer, error) {
	maxCalls := c.talk.MaxCalls
	if maxCalls <= 0 {
		maxCalls = defaultMaxCalls
	}

	for call := 1; ; call++ {
		last := call >= maxCalls
		result, err := c.harness.router.StreamChat(ctx, c.request(last), c.events.Delta)
		if err != nil {
			if result != nil {
				c.answer.Text = result.Text
				c.answer.Truncated = true
			}

			return c.answer, fmt.Errorf("nova %s: %w", c.talk.Name, err)
		}
		c.answer.Model = result.ModelIdentifier
		c.answer.ProviderID = result.ProviderID

		if len(result.ToolCalls) == 0 || last {
			c.answer.Text = strings.TrimSpace(result.Text)
			c.answer.Truncated = result.Truncated

			return c.answer, nil
		}

		c.box.distinct(result)
		c.events.Message(result.Text, result.ToolCalls, result.ModelIdentifier)
		c.messages = append(c.messages, llmports.Message{
			Role:          llmports.RoleAssistant,
			Content:       result.Text,
			ToolCalls:     result.ToolCalls,
			Reasoning:     result.Reasoning,
			ReplyID:       result.ProviderMessageID,
			ReplyProvider: result.ProviderID,
		})
		for _, toolCall := range result.ToolCalls {
			c.read(ctx, toolCall)
		}
	}
}

func (c *conversation) request(last bool) *llmports.ChatCompletionRequest {
	return &llmports.ChatCompletionRequest{
		TenantInfo: c.talk.TenantInfo,
		Task:       c.talk.Name,
		// The whole prompt is the same on every call of a task, so it is
		// cached as one block with the tools.
		System:         identity + "\n\n" + c.talk.Instructions,
		Messages:       c.messages,
		Tools:          c.box.specs,
		ToolChoiceNone: last,
		MaxTokens:      c.talk.MaxTokens,
		Attribution:    aiusageports.AIUsageAttribution{UserID: c.talk.UserID, Feature: c.talk.Feature},
	}
}

func (c *conversation) read(ctx context.Context, call llmports.ToolCall) {
	c.events.ToolStarted(call)
	result := c.box.call(ctx, call)
	c.events.ToolFinished(call, result)

	c.cite(result.Records)
	c.messages = append(c.messages, llmports.Message{
		Role:       llmports.RoleTool,
		Content:    result.Content,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		IsError:    result.Failed,
	})
}

func (c *conversation) cite(records []agentpagecontext.EntityRef) {
	for _, record := range records {
		key := record.Type + ":" + record.ID
		if _, seen := c.cited[key]; seen {
			continue
		}
		c.cited[key] = struct{}{}
		c.answer.Records = append(c.answer.Records, record)
	}
}

func question(talk *llmports.NovaConversation) string {
	text := talk.Context.Text()
	if text == "" {
		return talk.Question
	}

	return text + "\n\n## Question\n" + talk.Question
}
