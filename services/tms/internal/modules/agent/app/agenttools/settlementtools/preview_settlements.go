package settlementtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
)

func settlementVerb(action settlementshared.Action) string {
	switch action {
	case settlementshared.ActionSubmit:
		return "submit for approval"
	case settlementshared.ActionApprove:
		return common.VerbApprove
	case settlementshared.ActionReject:
		return "send back to draft"
	case settlementshared.ActionPost:
		return "post to the general ledger"
	case settlementshared.ActionMarkPaid:
		return "mark paid"
	case settlementshared.ActionVoid:
		return "void"
	case settlementshared.ActionRecalculate:
		return "recalculate"
	case settlementshared.ActionAddAdjustment:
		return "add an adjustment line to"
	case settlementshared.ActionRemoveAdjustment:
		return "remove an adjustment line from"
	default:
		return strings.ToLower(string(action))
	}
}

func (t *SettlementDecisionTool[E]) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	before := t.ledger.Facts(plan.Before)
	summary := fmt.Sprintf(
		"Would %s %s %s%s (now %s).",
		settlementVerb(t.decision.action),
		t.ledger.Noun,
		before.Number,
		common.PayeeClause(before.Payee),
		before.Status,
	)
	if plan.Refused() {
		return common.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: t.ledger.Resource,
		ID:       before.ID,
		Label:    before.Number,
		Version:  common.PinnedVersion(before.Version),
	}, plan.Before, plan.After,
		toolpreview.Only(t.decision.fields...),
		toolpreview.Volatile(t.decision.volatile...),
		toolpreview.WithRefs(t.decision.refs),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		common.SettlementMoney(before, t.ledger.Facts(plan.After)),
		toolpreview.SensitiveAs(t.ledger.Sensitive...),
	)

	changes := []*agent.RecordChange{change}
	sentences := []string{summary}
	if plan.Journal != nil {
		journal, journalErr := common.SettlementJournalChange(plan.Journal, before)
		if journalErr != nil {
			return nil, journalErr
		}
		changes = append(changes, journal)
		sentences = append(sentences, common.JournalSentence(plan.Journal))
	}
	if plan.AutoPost {
		sentences = append(sentences, "The settlement control posts it to the general "+
			"ledger as soon as it is approved.")
	}
	if t.ledger.Effects != nil {
		sentences = append(sentences, t.ledger.Effects(t.decision.action, plan)...)
	}

	return toolpreview.Build(strings.Join(common.SlicesNonEmpty(sentences), " "), changes...), nil
}
