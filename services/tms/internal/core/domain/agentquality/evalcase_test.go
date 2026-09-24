package agentquality

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCase() *EvalCase {
	evalCase := &EvalCase{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("agd_"),
		Source:            CaseSourceCurated,
		Status:            CaseStatusActive,
		Trigger:           agent.RunTriggerChat,
		Input:             "Put shipment S-100 on hold until the customer pays",
		HeldTools:         []string{"get_shipment", "place_shipment_hold"},
		Expected: Expected{
			ToolMode: ToolMatchAnyOrder,
			Tools: []ExpectedTool{{
				Name:  "place_shipment_hold",
				Args:  map[string]any{"shipmentId": "shp_1"},
				Rules: map[string]Tolerance{"reason": {Kind: TolerancePresent}},
			}},
			MustMention: []string{"hold"},
		},
	}
	hash, err := evalCase.ComputeContentHash()
	if err != nil {
		panic(err)
	}
	evalCase.ContentHash = hash

	return evalCase
}

func fieldsOf(t *testing.T, evalCase *EvalCase) []string {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	evalCase.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestEvalCaseValidate_AcceptsAWellFormedCase(t *testing.T) {
	t.Parallel()

	assert.Empty(t, fieldsOf(t, validCase()))
}

func TestEvalCaseValidate_Refusals(t *testing.T) {
	t.Parallel()

	abs := -1.0
	rel := 1.5
	tests := []struct {
		name   string
		mutate func(*EvalCase)
		field  string
	}{
		{name: "no input", mutate: func(c *EvalCase) { c.Input = "" }, field: "input"},
		{name: "blank input", mutate: func(c *EvalCase) { c.Input = "   " }, field: "input"},
		{name: "bad source", mutate: func(c *EvalCase) { c.Source = "Scraped" }, field: "source"},
		{name: "bad status", mutate: func(c *EvalCase) { c.Status = "Live" }, field: "status"},
		{
			name:   "half a subject",
			mutate: func(c *EvalCase) { c.SubjectType = agent.SubjectShipment },
			field:  "subjectId",
		},
		{
			name:   "decided without proposal",
			mutate: func(c *EvalCase) { c.Source = CaseSourceDecidedProposal },
			field:  "sourceProposalId",
		},
		{
			name:   "thumbs up without message",
			mutate: func(c *EvalCase) { c.Source = CaseSourceThumbsUp },
			field:  "sourceMessageId",
		},
		{
			name:   "duplicate held tool",
			mutate: func(c *EvalCase) { c.HeldTools = []string{"get_shipment", "get_shipment"} },
			field:  "heldTools[1].name",
		},
		{
			name: "unknown tolerance",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Rules = map[string]Tolerance{"shipmentId": {Kind: "fuzzy"}}
			},
			field: "expected.tools[0].rules.shipmentId.kind",
		},
		{
			name: "negative absolute tolerance",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Args["rate"] = 100
				c.Expected.Tools[0].Rules = map[string]Tolerance{
					"rate": {Kind: ToleranceNumeric, Abs: &abs},
				}
			},
			field: "expected.tools[0].rules.rate.abs",
		},
		{
			name: "relative tolerance above one",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Args["rate"] = 100
				c.Expected.Tools[0].Rules = map[string]Tolerance{
					"rate": {Kind: ToleranceNumeric, Rel: &rel},
				}
			},
			field: "expected.tools[0].rules.rate.rel",
		},
		{
			name: "date window without width",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Args["pickupDate"] = "2026-09-01"
				c.Expected.Tools[0].Rules = map[string]Tolerance{
					"pickupDate": {Kind: ToleranceDateWindow},
				}
			},
			field: "expected.tools[0].rules.pickupDate.windowSeconds",
		},
		{
			name: "one of without values",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Rules = map[string]Tolerance{"status": {Kind: ToleranceOneOf}}
			},
			field: "expected.tools[0].rules.status.values",
		},
		{
			name: "rule without expected value",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Rules = map[string]Tolerance{"rate": {Kind: ToleranceCI}}
			},
			field: "expected.tools[0].rules.rate",
		},
		{
			name: "set comparison against a scalar",
			mutate: func(c *EvalCase) {
				c.Expected.Tools[0].Args["ids"] = "trc_1"
				c.Expected.Tools[0].Rules = map[string]Tolerance{"ids": {Kind: ToleranceSetEq}}
			},
			field: "expected.tools[0].rules.ids",
		},
		{
			name: "expected and forbidden",
			mutate: func(c *EvalCase) {
				c.Expected.ForbiddenTools = []string{"place_shipment_hold"}
			},
			field: "expected.tools[0].name",
		},
		{
			name: "refusal with tools",
			mutate: func(c *EvalCase) {
				c.Expected.ExpectRefusal = true
			},
			field: "expected.expectRefusal",
		},
		{
			name: "approved proposal without params",
			mutate: func(c *EvalCase) {
				c.Expected.Proposals = []ExpectedProposal{{ToolName: "update_rate"}}
			},
			field: "expected.proposals[0].params",
		},
		{
			name: "required and forbidden phrase",
			mutate: func(c *EvalCase) {
				c.Expected.MustNotMention = []string{"HOLD"}
			},
			field: "expected.mustNotMention",
		},
		{
			name:   "active and expects nothing",
			mutate: func(c *EvalCase) { c.Expected = Expected{ToolMode: ToolMatchAnyOrder} },
			field:  "expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evalCase := validCase()
			tt.mutate(evalCase)

			assert.Contains(t, fieldsOf(t, evalCase), tt.field)
		})
	}
}

func TestEvalCaseValidate_CandidateMayExpectNothingYet(t *testing.T) {
	t.Parallel()

	evalCase := validCase()
	evalCase.Status = CaseStatusCandidate
	evalCase.Expected = Expected{ToolMode: ToolMatchAnyOrder}

	assert.Empty(t, fieldsOf(t, evalCase))
}

func TestExpectedNormalize_TrimsAndDedupes(t *testing.T) {
	t.Parallel()

	expected := Expected{
		ForbiddenTools: []string{" cancel_shipment ", "cancel_shipment", ""},
		MustMention:    []string{"hold", " hold", ""},
		Tools:          []ExpectedTool{{Name: " get_shipment "}},
	}
	expected.Normalize()

	assert.Equal(t, ToolMatchAnyOrder, expected.ToolMode)
	assert.Equal(t, []string{"cancel_shipment"}, expected.ForbiddenTools)
	assert.Equal(t, []string{"hold"}, expected.MustMention)
	assert.Equal(t, "get_shipment", expected.Tools[0].Name)
	assert.NotNil(t, expected.Proposals)
}

func TestExpectedToolChecked_SkipsIgnoredArguments(t *testing.T) {
	t.Parallel()

	tool := ExpectedTool{
		Name: "update_rate",
		Args: map[string]any{"rate": 10, "note": "x"},
		Rules: map[string]Tolerance{
			"note":   {Kind: ToleranceIgnore},
			"reason": {Kind: TolerancePresent},
		},
	}

	assert.Equal(t, []string{"rate", "reason"}, tool.Checked())
	assert.Equal(t, ToleranceExact, tool.RuleFor("rate").Kind)
}

func TestCaseStatus_Transitions(t *testing.T) {
	t.Parallel()

	assert.True(t, CaseStatusCandidate.CanBecome(CaseStatusActive))
	assert.True(t, CaseStatusActive.CanBecome(CaseStatusQuarantined))
	assert.True(t, CaseStatusQuarantined.CanBecome(CaseStatusActive))
	assert.True(t, CaseStatusRetired.CanBecome(CaseStatusActive))
	assert.False(t, CaseStatusCandidate.CanBecome(CaseStatusQuarantined))
	assert.False(t, CaseStatusRetired.CanBecome(CaseStatusCandidate))
	assert.False(t, CaseStatusActive.CanBecome(CaseStatusActive))
	assert.True(t, CaseStatusActive.Runs())
	assert.False(t, CaseStatusQuarantined.Runs())
}

func TestCaseSource_Weight(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, CaseSourceCurated.Weight(), 1e-9)
	assert.InDelta(t, 1.0, CaseSourceDecidedProposal.Weight(), 1e-9)
	assert.InDelta(t, 0.6, CaseSourceThumbsUp.Weight(), 1e-9)
}

func TestComputeContentHash_DedupesTheSameFrozenInput(t *testing.T) {
	t.Parallel()

	first := validCase()
	second := validCase()
	second.AgentDefinitionID = first.AgentDefinitionID
	second.Title = "A different title"
	second.Expected.MustMention = []string{"something else"}
	second.Input = "  " + first.Input + "  "

	firstHash, err := first.ComputeContentHash()
	require.NoError(t, err)
	secondHash, err := second.ComputeContentHash()
	require.NoError(t, err)
	assert.Equal(t, firstHash, secondHash, "only the frozen input decides a duplicate")

	second.History = []HistoryMessage{{Role: conversation.RoleUser, Content: "earlier"}}
	changed, err := second.ComputeContentHash()
	require.NoError(t, err)
	assert.NotEqual(t, firstHash, changed)

	other := validCase()
	otherHash, err := other.ComputeContentHash()
	require.NoError(t, err)
	assert.NotEqual(t, firstHash, otherHash, "the same question to another agent is its own case")
}

func TestHistoryFromMessages_DropsProviderData(t *testing.T) {
	t.Parallel()

	history := HistoryFromMessages([]conversation.Message{{
		Role: conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{
			ID:           "call_1",
			Name:         "get_shipment",
			Arguments:    map[string]any{"id": "shp_1"},
			ProviderData: map[string]any{"signature": "opaque"},
		}},
		Reasoning: &conversation.ReasoningTrace{Text: "thinking", Encrypted: "secret"},
	}})

	require.Len(t, history, 1)
	require.Len(t, history[0].ToolCalls, 1)
	assert.Nil(t, history[0].ToolCalls[0].ProviderData)
	message := history[0].Message()
	assert.Nil(t, message.Reasoning)
	assert.Equal(t, conversation.MessageKindMessage, message.Kind)
}
