package agentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func Resources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:    permission.ResourceAgentRun.String(),
			DisplayName: "Agent Run",
			Description: "Billing exception agent runs",
			Category:    "AI & agents",
			ReferenceView: []string{
				"agentDefinitionId",
				"subjectType",
				"subjectId",
				"trigger",
				"definition",
			},
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View agent runs"},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Start agent runs",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:      permission.ResourceAgentProposal.String(),
			DisplayName:   "Decisions",
			Description:   "What the agents propose and ask a person to approve or decline",
			Category:      "Desk",
			ReferenceView: []string{"toolName", "status"},
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View agent proposals",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Accept, modify or reject agent proposals, and resolve proposals and plans",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceAgentException.String(),
			DisplayName: "Agent Exception",
			Description: "Agent exceptions raised for human review",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View agent exceptions",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Raise agent exceptions",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Resolve agent exceptions",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceAgentControl.String(),
			DisplayName: "Agent Control",
			Description: "Per-organization agent autonomy configuration",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View agent control settings, and dismiss or restore provider failure notices",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Update agent control settings, and apply or restore tune-ups",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		// What the organization has told its agents. Reading it shows what every
		// agent is told. A person's own memories need only the assistant, and
		// anyone may offer one for their team or the organization; keeping,
		// changing, retiring and approving what reaches other people's
		// conversations is manage, which the role templates never hand out, so it
		// stays an administrator's unless a role is given it on purpose.
		{
			Resource:    permission.ResourceAgentMemory.String(),
			DisplayName: "Agent Memory",
			Description: "Standing instructions, facts, corrections and profile guidance kept for agents between runs",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View what agents remember",
				},
				{
					Operation:   permission.OpManage,
					DisplayName: permissionspec.DisplayName(permission.OpManage),
					Description: "Keep, change, retire, review and approve memories and profile guidance shared with a team, the organization or every agent",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceAgentEvalSuite.String(),
			DisplayName: "Agent Evaluation Suite",
			Description: "Evaluation cases agents are replayed against and scored on",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View evaluation cases and scores",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Capture or write an evaluation case and replay it",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Edit what a case expects, activate, quarantine or retire it, and cancel runs",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceAgentFeedback.String(),
			DisplayName: "Agent Feedback",
			Description: "Everyone's ratings of AI output, with what they saw and why they rated it",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View everyone's ratings of AI output and each agent's satisfaction",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		// The sealed record of what every agent did and on whose behalf: each
		// model call, tool call, proposal, decision and write. It names people,
		// records and arguments across the whole organization, so it is kept for
		// those who answer for the organization's agents; reading an agent's runs
		// does not grant it, and no agent can hold it.
		{
			Resource:    permission.ResourceAIAuditTrail.String(),
			DisplayName: "AI Audit Trail",
			Description: "The tamper-evident record of what agents did, for whom, and who decided",
			Category:    "Users & security",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View the AI audit trail and verify its chain",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export the AI audit trail and download exports",
				},
			},
			DefaultSensitivity: permission.SensitivityConfidential,
		},

		// unattended, so it sits at the same sensitivity as provider configuration.
		{
			Resource:      permission.ResourceAgentDefinition.String(),
			DisplayName:   "Agent Definition",
			Description:   "Per-organization agent configurations and their tool access",
			Category:      "AI & agents",
			ReferenceView: []string{"name", "icon", "accent", "template"},
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View configured agents",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Add an agent",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Update an agent's tools and autonomy",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Remove an agent",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		// Using the assistant is an ordinary operational act rather than an
		// administrative one, so it is held separately from configuring agents: a
		// dispatcher should be able to ask questions without being able to change
		// which tools an agent holds.
		{
			Resource:    permission.ResourceAssistant.String(),
			DisplayName: "Desk",
			Description: "Conversations with the agents and what they produce",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View own conversations and pin artifacts in them",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Start conversations and send messages",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Delete own conversations",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		// An insight names a customer, a location or a driver alongside a figure, so
		// reading the panel is a real disclosure. It is held separately from the
		// records it draws on because a reader still needs permission over those:
		// the panel filters to what its reader may already see, and this permission
		// only decides whether they get a panel at all.
		{
			Resource:    permission.ResourceInsight.String(),
			DisplayName: "Insight",
			Description: "Operational findings surfaced on the home screen",
			Category:    "Reports",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View operational insights",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Dismiss an insight that is not worth acting on, or restore one",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		// The watchtower is a feed over records a reader may already be allowed to
		// see: every item is filtered again by the permission on the source it came
		// from, so this one decides only whether a person gets the feed at all.
		{
			Resource:    permission.ResourceWatchtower.String(),
			DisplayName: "Watchtower",
			Description: "The live feed of what needs attention across the operation",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View the watchtower feed",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Mark the feed seen, dismiss an item, or hand one to an agent",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		// A briefing gathers the morning's figures across dispatch, billing and
		// compliance into one page, which is a wider disclosure than any single
		// screen it draws on.
		{
			Resource:    permission.ResourceBriefing.String(),
			DisplayName: "Briefing",
			Description: "The daily briefing written for a role from the day's figures",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "Read the daily briefing",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Write today's briefing again from current figures",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		// Configuring a provider decides which endpoint an organization's freight and
		// billing data is sent to, and whether that endpoint may be on the local
		// network, so it is held at the same sensitivity as credential management.
		{
			Resource:    permission.ResourceAgentExtension.String(),
			DisplayName: "Agent Extension",
			Description: "Capabilities turned on for agents only, such as web research, and their credentials",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View the extension marketplace",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Turn extensions on or off, change their settings and test them",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceWebResearch.String(),
			DisplayName: "Web Research",
			Description: "Agents searching and reading public web pages on a person's behalf, through an enabled extension",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "Let agents search and read the web for this person",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourcePortalAutomation.String(),
			DisplayName: "Portal Automation",
			Description: "Agents reading and acting on the third-party web portals the organization set up, through an enabled extension",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "Let agents look things up on portals for this person",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Act",
					Description: "Let agents propose booking, uploading or accepting on portals for this person",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceCodeMode.String(),
			DisplayName: "Scripts and analysis",
			Description: "Agents answering questions across many records, and analyzing report results and attached files, with short read-only scripts",
			Category:    "Desk",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "Let agents run read-only scripts over what this person may read, the reports they ran and the files they attached",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceAIProvider.String(),
			DisplayName: "AI Provider",
			Description: "Model endpoints, including self-hosted servers, and their task routing",
			Category:    "AI & agents",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View AI providers",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Add an AI provider",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Update an AI provider and its task routing",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Remove an AI provider",
				},
				{
					Operation:   permission.OpManage,
					DisplayName: "Manage",
					Description: "Test an AI provider connection",
				},
			},
			DefaultSensitivity: permission.SensitivityRestricted,
		},
	}
}
