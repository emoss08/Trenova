package auditctx

import "context"

const (
	ChannelWeb               = "web"
	ChannelAPI               = "api"
	ChannelPortal            = "portal"
	ChannelAgentConversation = "agent_conversation"
	ChannelAgentRun          = "agent_run"
	ChannelSupport           = "support"
	ChannelSCIM              = "scim"
	ChannelJob               = "job"
	ChannelCLI               = "cli"
	ChannelSystem            = "system"
)

type Source struct {
	Channel string
	AIRunID string
	// AgentID is the agent whose tool call made the change, so what people
	// later change of an agent's work can be traced to the agent that did it.
	AgentID string
}

type sourceKey struct{}

func From(ctx context.Context) Source {
	source, _ := ctx.Value(sourceKey{}).(Source)
	return source
}

func WithChannel(ctx context.Context, channel string) context.Context {
	source := From(ctx)
	source.Channel = channel
	return context.WithValue(ctx, sourceKey{}, source)
}

func WithAIRun(ctx context.Context, runID string, conversation bool) context.Context {
	source := From(ctx)
	source.AIRunID = runID
	source.Channel = ChannelAgentRun
	if conversation {
		source.Channel = ChannelAgentConversation
	}
	return context.WithValue(ctx, sourceKey{}, source)
}

func WithAgent(ctx context.Context, agentID string) context.Context {
	if agentID == "" {
		return ctx
	}
	source := From(ctx)
	source.AgentID = agentID
	return context.WithValue(ctx, sourceKey{}, source)
}
