package audit

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

type ActorKind string

const (
	ActorPeople       = ActorKind("People")
	ActorAgents       = ActorKind("Agents")
	ActorIntegrations = ActorKind("Integrations")
	ActorJobs         = ActorKind("Jobs")
)

var ActorKinds = []ActorKind{ActorPeople, ActorAgents, ActorIntegrations, ActorJobs}

const (
	OffHoursFrom = 20
	OffHoursTo   = 6
)

var (
	PeopleChannels      = []Channel{ChannelWeb, ChannelPortal, ChannelSupport}
	AgentChannels       = []Channel{ChannelAgentConversation, ChannelAgentRun}
	IntegrationChannels = []Channel{ChannelAPI, ChannelSCIM}
	JobChannels         = []Channel{ChannelJob, ChannelCLI, ChannelSystem}

	SensitiveOperations = []permission.Operation{permission.OpDelete, permission.OpExport}
)

func (k ActorKind) Channels() []Channel {
	switch k {
	case ActorAgents:
		return AgentChannels
	case ActorIntegrations:
		return IntegrationChannels
	case ActorJobs:
		return JobChannels
	default:
		return PeopleChannels
	}
}

func ActorKindOfPrincipal(principalType string) ActorKind {
	switch principalType {
	case principalAgent:
		return ActorAgents
	case principalAPIKey:
		return ActorIntegrations
	case principalSystem:
		return ActorJobs
	default:
		return ActorPeople
	}
}

func IsOffHours(hour int) bool {
	return hour >= OffHoursFrom || hour < OffHoursTo
}
