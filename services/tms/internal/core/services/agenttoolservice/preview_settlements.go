package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type settlementJournalView struct {
	AccountingDate   int64    `json:"accountingDate"`
	FiscalPeriodID   pulid.ID `json:"fiscalPeriodId"`
	EntryStatus      string   `json:"entryStatus"`
	RequiresApproval bool     `json:"requiresApproval"`
	Description      string   `json:"description"`
	LineCount        int      `json:"lineCount"`
}

func wouldFail(preview *agent.ToolPreview, refusal error) (*agent.ToolPreview, error) {
	return warnWouldFail(preview, refusal), nil
}

func minorAmount(minor int64) decimal.NullDecimal {
	return knownAmount(money.DecimalFromMinor(minor))
}

func settlementVerb(action settlementshared.Action) string {
	switch action {
	case settlementshared.ActionSubmit:
		return "submit for approval"
	case settlementshared.ActionApprove:
		return "approve"
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

func (t *settlementDecisionTool[E]) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	before := t.ledger.facts(plan.Before)
	summary := fmt.Sprintf(
		"Would %s %s %s%s (now %s).",
		settlementVerb(t.decision.action),
		t.ledger.noun,
		before.number,
		payeeClause(before.payee),
		before.status,
	)
	if plan.Refused() {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: t.ledger.resource,
		ID:       before.id,
		Label:    before.number,
		Version:  pinnedVersion(before.version),
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
		settlementMoney(before, t.ledger.facts(plan.After)),
		toolpreview.SensitiveAs(t.ledger.sensitive...),
	)

	changes := []*agent.RecordChange{change}
	sentences := []string{summary}
	if plan.Journal != nil {
		journal, journalErr := settlementJournalChange(plan.Journal, before)
		if journalErr != nil {
			return nil, journalErr
		}
		changes = append(changes, journal)
		sentences = append(sentences, journalSentence(plan.Journal))
	}
	if plan.AutoPost {
		sentences = append(sentences, "The settlement control posts it to the general "+
			"ledger as soon as it is approved.")
	}
	if t.ledger.effects != nil {
		sentences = append(sentences, t.ledger.effects(t.decision.action, plan)...)
	}

	return toolpreview.Build(strings.Join(slicesNonEmpty(sentences), " "), changes...), nil
}

func payeeClause(payee string) string {
	if strings.TrimSpace(payee) == "" {
		return ""
	}

	return " for " + payee
}

func settlementMoney(before, after *settlementFacts) *agent.MoneyPreview {
	lines := make([]agent.MoneyLine, 0, max(len(before.totals), len(after.totals)))
	for idx, total := range after.totals {
		line := agent.MoneyLine{Label: total.label, After: minorAmount(total.minor)}
		if idx < len(before.totals) {
			line.Before = minorAmount(before.totals[idx].minor)
		}
		lines = append(lines, line)
	}

	return toolpreview.MoneyBlock(after.currency, lines...)
}

func journalSentence(plan *settlementshared.JournalPlan) string {
	if plan.RequiresApproval {
		return fmt.Sprintf(
			"A %s journal entry would be booked and wait for its approval.",
			strings.ToLower(plan.EntryStatus),
		)
	}

	return fmt.Sprintf("A %s journal entry would be booked.", strings.ToLower(plan.EntryStatus))
}

func settlementJournalChange(
	plan *settlementshared.JournalPlan,
	settlement *settlementFacts,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceJournalEntry,
			Label:    plan.Description,
		},
		&settlementJournalView{
			AccountingDate:   plan.AccountingDate,
			FiscalPeriodID:   plan.FiscalPeriodID,
			EntryStatus:      plan.EntryStatus,
			RequiresApproval: plan.RequiresApproval,
			Description:      plan.Description,
			LineCount:        len(plan.Lines),
		},
		toolpreview.WithRefs(map[string]permission.Resource{
			"fiscalPeriodId": permission.ResourceFiscalPeriod,
		}),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			"accountingDate": assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}

	lines := make([]agent.MoneyLine, 0, len(plan.Lines))
	for _, line := range plan.Lines {
		if line.DebitMinor > 0 {
			lines = append(lines, agent.MoneyLine{
				Label: "Debit",
				After: minorAmount(line.DebitMinor),
			})
		}
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(settlement.currency, lines...),
		toolpreview.SensitiveAs("totalAmount"))

	return change, nil
}
