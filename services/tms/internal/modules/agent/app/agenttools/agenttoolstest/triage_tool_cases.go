package agenttoolstest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TriageMessage() *inboundmessage.InboundMessage {
	mailboxID := pulid.MustNew("imbx_")

	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		MailboxID:   mailboxID,
		Status:      inboundmessage.StatusInReview,
		FromAddress: "Dana@Shipper.example",
		Subject:     "Where is 88213?",
		Version:     3,
		Mailbox: &inboundmessage.Mailbox{
			ID:      mailboxID,
			Address: "loads@carrier.example",
		},
	}
}

func NewFakeTriage() *FakeTriage {
	return &FakeTriage{Message: TriageMessage()}
}

type TriageToolCase struct {
	Build     func(f *FakeTriage) agenttoolports.AgentTool
	Name      string
	Required  []string
	Resource  permission.Resource
	Operation permission.Operation
	Egress    agent.EgressClass
	MaxTier   agent.AutonomyTier
	Condition bool
}

func CheckTriageToolSchemaAndPolicy(t *testing.T, cases []TriageToolCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tool := tc.Build(NewFakeTriage())
			require.Equal(t, tc.Name, tool.Name())

			schema := tool.ParamSchema()
			assert.Equal(t, false, schema[toolschema.KeyAdditionalProperties])
			assert.ElementsMatch(t, tc.Required, schema[toolschema.KeyRequired])
			properties, _ := schema[toolschema.KeyProperties].(map[string]any)
			for _, name := range tc.Required {
				assert.Containsf(t, properties, name, "%s is required but not declared", name)
			}

			policy := tool.Policy()
			assert.Equal(t, tc.Name, policy.Name)
			assert.Equal(t, agenttooltrust.ToolKindAction, policy.Kind)
			assert.Equal(t, tc.Resource, policy.Resource)
			assert.Equal(t, tc.Operation, policy.Operation)
			assert.Equal(t, []agent.EgressClass{tc.Egress}, policy.Egress)
			assert.Equal(t, tc.MaxTier, policy.MaxTier)
			assert.False(t, policy.DefaultTier.Above(policy.MaxTier))
			assert.Equal(t, tc.Condition, policy.Condition != nil)
			assert.NotEmpty(t, policy.Rationale)
			assert.NotEmpty(t, policy.Artifact, "a tool that reports its result names its record")

			_, previews := tool.(agenttoolports.ToolPreviewer)
			assert.True(t, previews, "every write previews what it would do")
			_, reports := tool.(agenttoolports.ToolResultReporter)
			assert.True(t, reports)
		})
	}
}

func CheckTriageToolMailboxCondition(t *testing.T, cases []TriageToolCase) {
	t.Helper()

	for _, tc := range cases {
		if !tc.Condition {
			continue
		}
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			f := NewFakeTriage()
			f.Message.Status = inboundmessage.StatusClassified
			f.Message.Mailbox.ReviewPolicy = inboundmessage.ReviewAutoHandle
			condition := tc.Build(f).Policy().Condition
			params := DeskParams(map[string]any{"messageId": f.Message.ID.String()})

			assert.Equal(t, agent.TierAutoExecute, condition.Limit(t.Context(), params))

			f.Message.Status = inboundmessage.StatusInReview
			assert.Equal(t, agent.TierPropose, condition.Limit(t.Context(), params),
				"a message held for review is a proposal")
		})
	}
}

func CheckTriageToolTargets(t *testing.T, cases []TriageToolCase) {
	t.Helper()

	for _, tc := range cases {
		tool := tc.Build(NewFakeTriage())
		properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		if _, single := properties["messageId"]; !single {
			continue
		}
		id := pulid.MustNew("imsg_")
		target, ok := tool.(agenttoolports.TargetedTool).
			Target(map[string]any{"messageId": id.String()})
		require.Truef(t, ok, "%s names no target", tc.Name)
		assert.Equal(
			t,
			agenttoolports.ToolTarget{Resource: permission.ResourceInboundMessage, ID: id},
			target,
			tc.Name,
		)
	}
}

func TriageToolPolicies(
	f *FakeTriage,
	cases []TriageToolCase,
) map[string]agenttoolports.ToolPolicy {
	policies := make(map[string]agenttoolports.ToolPolicy, len(cases))
	for _, tc := range cases {
		tool := tc.Build(f)
		policies[tool.Name()] = tool.Policy()
	}

	return policies
}
