package masterdatatools

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const fieldAccountingDate = "accountingDate"

const fieldFiscalPeriodID = "fiscalPeriodId"

const sideDebit = "debit"

const sideCredit = "credit"

var ledgerDateTypes = map[string]assistantartifact.DisplayType{
	fieldAccountingDate: assistantartifact.DisplayDate,
}

var ledgerRefs = map[string]permission.Resource{
	fieldFiscalPeriodID:      permission.ResourceFiscalPeriod,
	"originalJournalEntryId": permission.ResourceJournalEntry,
}

type manualJournalView struct {
	Status         string   `json:"status"`
	Description    string   `json:"description"`
	Reason         string   `json:"reason"`
	AccountingDate int64    `json:"accountingDate"`
	FiscalPeriodID pulid.ID `json:"fiscalPeriodId"`
	CurrencyCode   string   `json:"currencyCode"`
	LineCount      int      `json:"lineCount"`
	Balanced       bool     `json:"balanced"`
	CancelReason   string   `json:"cancelReason"`
}

type reversalView struct {
	Status                 string   `json:"status"`
	OriginalJournalEntryID pulid.ID `json:"originalJournalEntryId"`
	AccountingDate         int64    `json:"accountingDate"`
	FiscalPeriodID         pulid.ID `json:"fiscalPeriodId"`
	ReasonCode             string   `json:"reasonCode"`
	ReasonText             string   `json:"reasonText"`
	CancelReason           string   `json:"cancelReason"`
}

type bookedEntryView struct {
	AccountingDate int64    `json:"accountingDate"`
	FiscalPeriodID pulid.ID `json:"fiscalPeriodId"`
	EntryStatus    string   `json:"entryStatus"`
	LineCount      int      `json:"lineCount"`
}

type reversedEntryView struct {
	Status string `json:"status"`
}

func manualJournalViewOf(entity *manualjournal.Request) *manualJournalView {
	lines := 0
	for _, line := range entity.Lines {
		if line != nil {
			lines++
		}
	}

	return &manualJournalView{
		Status:         string(entity.Status),
		Description:    entity.Description,
		Reason:         entity.Reason,
		AccountingDate: entity.AccountingDate,
		FiscalPeriodID: entity.RequestedFiscalPeriodID,
		CurrencyCode:   entity.CurrencyCode,
		LineCount:      lines,
		Balanced:       entity.IsBalanced(),
		CancelReason:   entity.CancelReason,
	}
}

func reversalViewOf(entity *journalreversal.Reversal) *reversalView {
	return &reversalView{
		Status:                 string(entity.Status),
		OriginalJournalEntryID: entity.OriginalJournalEntryID,
		AccountingDate:         entity.RequestedAccountingDate,
		FiscalPeriodID:         entity.ResolvedFiscalPeriodID,
		ReasonCode:             entity.ReasonCode,
		ReasonText:             entity.ReasonText,
		CancelReason:           entity.CancelReason,
	}
}

func journalLineLabel(labels map[pulid.ID]string, line *kernelports.JournalLinePreview) string {
	account, ok := labels[line.GLAccountID]
	if !ok {
		account = line.GLAccountID.String()
	}
	side := sideDebit
	if line.CreditMinor > 0 {
		side = sideCredit
	}

	return account + " (" + side + ")"
}

func signedMinor(line *kernelports.JournalLinePreview) decimal.Decimal {
	return money.DecimalFromMinor(line.DebitMinor - line.CreditMinor)
}

func journalMoney(
	currency string,
	labels map[pulid.ID]string,
	before, after []kernelports.JournalLinePreview,
) *agent.MoneyPreview {
	order := make([]string, 0, len(before)+len(after))
	byLabel := make(map[string]*agent.MoneyLine, len(before)+len(after))
	lineFor := func(label string) *agent.MoneyLine {
		line, ok := byLabel[label]
		if !ok {
			line = &agent.MoneyLine{Label: label}
			byLabel[label] = line
			order = append(order, label)
		}

		return line
	}
	accumulate := func(total decimal.NullDecimal, amount decimal.Decimal) decimal.NullDecimal {
		return decimal.NewNullDecimal(total.Decimal.Add(amount))
	}

	for idx := range before {
		line := lineFor(journalLineLabel(labels, &before[idx]))
		line.Before = accumulate(line.Before, signedMinor(&before[idx]))
	}
	for idx := range after {
		line := lineFor(journalLineLabel(labels, &after[idx]))
		line.After = accumulate(line.After, signedMinor(&after[idx]))
	}

	lines := make([]agent.MoneyLine, 0, len(order))
	for _, label := range order {
		lines = append(lines, *byLabel[label])
	}

	return toolpreview.MoneyBlock(currency, lines...)
}

func manualJournalRecord(entity *manualjournal.Request) toolpreview.Record {
	label := entity.RequestNumber
	if label == "" {
		label = entity.Description
	}
	rec := toolpreview.Record{
		Resource: permission.ResourceManualJournal,
		ID:       entity.ID,
		Label:    label,
	}
	if entity.ID.IsNotNil() {
		rec.Version = common.PinnedVersion(entity.Version)
	}

	return rec
}

func ledgerOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.WithRefs(ledgerRefs),
		toolpreview.Types(ledgerDateTypes),
	}
}

func balanceSentence(entity *manualjournal.Request) string {
	if entity.IsBalanced() {
		return "Its debits and credits balance."
	}

	return "Its debits and credits do not balance yet, so it cannot be submitted until they do."
}

func BookedEntryChange(
	journal *kernelports.JournalPreview,
	currency string,
	label string,
	labels map[pulid.ID]string,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceJournalEntry, Label: label},
		&bookedEntryView{
			AccountingDate: journal.AccountingDate,
			FiscalPeriodID: journal.FiscalPeriodID,
			EntryStatus:    journal.EntryStatus,
			LineCount:      len(journal.Lines),
		},
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, journalMoney(currency, labels, nil, journal.Lines))

	return change, nil
}

func RenderManualJournalDraft(
	_ *manualjournalports.CreateManualJournalRequest,
	plan *common.ManualJournalPlan,
) (*agent.ToolPreview, error) {
	planned := plan.After
	change, err := toolpreview.Create(
		manualJournalRecord(planned),
		manualJournalViewOf(planned),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		journalMoney(planned.CurrencyCode, plan.Accounts, nil, common.JournalLinesOf(planned)),
	)

	return toolpreview.Build(fmt.Sprintf(
		"Would draft the manual journal %q for %s with %s. %s It books nothing until a "+
			"person submits, approves and posts it.",
		planned.Description,
		common.DayLabel(planned.AccountingDate),
		common.CountOf(len(common.JournalLinesOf(planned)), "line"),
		balanceSentence(planned),
	), change), nil
}

func manualJournalChangeOf(
	plan *common.ManualJournalPlan,
	withLines bool,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Changed(
		manualJournalRecord(plan.Before),
		manualJournalViewOf(plan.Before),
		manualJournalViewOf(plan.After),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	if withLines {
		toolpreview.AttachMoney(change, journalMoney(
			plan.After.CurrencyCode,
			plan.Accounts,
			common.JournalLinesOf(plan.Before),
			common.JournalLinesOf(plan.After),
		))
	}

	return change, nil
}

func RenderManualJournalRevision(
	_ *common.JournalRevision,
	plan *common.ManualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, true)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would revise draft manual journal %s. %s",
		plan.Before.RequestNumber,
		balanceSentence(plan.After),
	), change), nil
}

func RenderManualJournalSubmit(
	_ *manualjournalports.GetManualJournalRequest,
	plan *common.ManualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would submit manual journal %s for approval; it books nothing until a person posts it.",
		plan.Before.RequestNumber,
	)
	if plan.After.Status == manualjournal.StatusApproved {
		summary = fmt.Sprintf(
			"Manual journal approval is turned off, so submitting %s approves it as the person "+
				"who submits it; it books nothing until a person posts it.",
			plan.Before.RequestNumber,
		)
	}

	return toolpreview.Build(summary, change), nil
}

func RenderManualJournalCancel(
	_ *manualjournalports.CancelManualJournalRequest,
	plan *common.ManualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel manual journal %s (now %s); it is kept and never posted.",
		plan.Before.RequestNumber,
		plan.Before.Status,
	), change), nil
}

func RenderManualJournalPost(
	_ *manualjournalports.GetManualJournalRequest,
	plan *common.ManualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}
	entry, err := BookedEntryChange(
		plan.Journal,
		plan.After.CurrencyCode,
		"Journal entry for "+plan.Before.RequestNumber,
		plan.Accounts,
	)
	if err != nil {
		return nil, err
	}

	sentences := []string{fmt.Sprintf(
		"Would post manual journal %s to the general ledger on %s.",
		plan.Before.RequestNumber,
		common.DayLabel(plan.Journal.AccountingDate),
	)}
	if plan.Journal.AccountingDate != plan.Before.AccountingDate {
		sentences = append(sentences, fmt.Sprintf(
			"Its own day, %s, falls in a closed period, so the closed period policy moves it "+
				"to the next open one.",
			common.DayLabel(plan.Before.AccountingDate),
		))
	}
	sentences = append(sentences, "Only a reversal undoes it.")

	return toolpreview.Build(strings.Join(sentences, " "), change, entry), nil
}

func reversalRecord(entity *journalreversal.Reversal, label string) toolpreview.Record {
	rec := toolpreview.Record{
		Resource: permission.ResourceJournalReversal,
		ID:       entity.ID,
		Label:    label,
	}
	if entity.ID.IsNotNil() {
		rec.Version = common.PinnedVersion(entity.Version)
	}

	return rec
}

func originalEntryNumber(change *journalreversalports.JournalReversalChange) string {
	if change.OriginalEntry == nil {
		return "the original entry"
	}

	return "journal entry " + change.OriginalEntry.EntryNumber
}

func RenderReversalRequest(
	_ *journalreversalports.CreateJournalReversalRequest,
	plan *common.ReversalPlan,
) (*agent.ToolPreview, error) {
	after := plan.Change.After
	label := "Reversal of " + originalEntryNumber(plan.Change)
	change, err := toolpreview.Create(
		reversalRecord(after, label),
		reversalViewOf(after),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, journalMoney(
		plan.Change.CurrencyCode,
		plan.Accounts,
		nil,
		plan.Change.Journal.Lines,
	))

	next := "It waits for an approver, and post_journal_reversal books it once approved."
	if after.Status == journalreversal.StatusApproved {
		next = "Journal approval is turned off, so it is approved at once as the person who " +
			"requests it, and post_journal_reversal books it."
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would request a reversal of %s, booked on %s with every debit and credit swapped. %s",
		originalEntryNumber(plan.Change),
		common.DayLabel(after.RequestedAccountingDate),
		next,
	), change), nil
}

func RenderReversalCancel(
	_ *journalreversalports.CancelJournalReversalRequest,
	plan *common.ReversalPlan,
) (*agent.ToolPreview, error) {
	before := plan.Change.Before
	change, err := toolpreview.Changed(
		reversalRecord(before, "Journal reversal "+before.ReasonCode),
		reversalViewOf(before),
		reversalViewOf(plan.Change.After),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel the journal reversal (now %s); the original entry stands as posted.",
		before.Status,
	), change), nil
}

func RenderReversalPost(
	_ *journalreversalports.GetJournalReversalRequest,
	plan *common.ReversalPlan,
) (*agent.ToolPreview, error) {
	before := plan.Change.Before
	label := "Reversal of " + originalEntryNumber(plan.Change)
	change, err := toolpreview.Changed(
		reversalRecord(before, label),
		reversalViewOf(before),
		reversalViewOf(plan.Change.After),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	entry, err := BookedEntryChange(
		plan.Change.Journal,
		plan.Change.CurrencyCode,
		label,
		plan.Accounts,
	)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{change, entry}
	if original := plan.Change.OriginalEntry; original != nil {
		reversed, reversedErr := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceJournalEntry,
				ID:       original.ID,
				Label:    "Journal entry " + original.EntryNumber,
				Version:  common.PinnedVersion(original.Version),
			},
			&reversedEntryView{Status: string(original.Status)},
			&reversedEntryView{Status: string(journalentry.StatusReversed)},
		)
		if reversedErr != nil {
			return nil, reversedErr
		}
		changes = append(changes, reversed)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would post the reversal of %s on %s, swapping every debit and credit, and mark the "+
			"original reversed. It cannot be undone.",
		originalEntryNumber(plan.Change),
		common.DayLabel(before.RequestedAccountingDate),
	), changes...), nil
}
