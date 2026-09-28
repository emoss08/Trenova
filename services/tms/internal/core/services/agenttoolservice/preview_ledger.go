package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	fieldAccountingDate = "accountingDate"
	fieldFiscalPeriodID = "fiscalPeriodId"
	sideDebit           = "debit"
	sideCredit          = "credit"
)

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

type manualJournalPlan struct {
	before   *manualjournal.Request
	after    *manualjournal.Request
	journal  *serviceports.JournalPreview
	accounts map[pulid.ID]string
}

type reversalPlan struct {
	change   *serviceports.JournalReversalChange
	accounts map[pulid.ID]string
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

func journalLinesOf(entity *manualjournal.Request) []serviceports.JournalLinePreview {
	if entity == nil {
		return nil
	}
	lines := make([]serviceports.JournalLinePreview, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, serviceports.JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return lines
}

func accountLabels(
	ctx context.Context,
	accounts ledgerAccounts,
	tenant pagination.TenantInfo,
	sets ...[]serviceports.JournalLinePreview,
) (map[pulid.ID]string, error) {
	seen := make(map[pulid.ID]struct{})
	ids := make([]pulid.ID, 0)
	for _, lines := range sets {
		for _, line := range lines {
			if line.GLAccountID.IsNil() {
				continue
			}
			if _, dup := seen[line.GLAccountID]; dup {
				continue
			}
			seen[line.GLAccountID] = struct{}{}
			ids = append(ids, line.GLAccountID)
		}
	}

	labels := make(map[pulid.ID]string, len(ids))
	if accounts == nil || len(ids) == 0 {
		return labels, nil
	}

	found, err := accounts.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
		TenantInfo:   tenant,
		GLAccountIDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read the accounts the journal posts to: %w", err)
	}
	for _, account := range found {
		if account != nil {
			labels[account.ID] = strings.TrimSpace(account.AccountCode + " " + account.Name)
		}
	}

	return labels, nil
}

func journalLineLabel(labels map[pulid.ID]string, line *serviceports.JournalLinePreview) string {
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

func signedMinor(line *serviceports.JournalLinePreview) decimal.Decimal {
	return money.DecimalFromMinor(line.DebitMinor - line.CreditMinor)
}

func journalMoney(
	currency string,
	labels map[pulid.ID]string,
	before, after []serviceports.JournalLinePreview,
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
		rec.Version = pinnedVersion(entity.Version)
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

func bookedEntryChange(
	journal *serviceports.JournalPreview,
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

func planManualJournalDraft(
	ctx context.Context,
	accounts ledgerAccounts,
	tenant pagination.TenantInfo,
	planned *manualjournal.Request,
) (*manualJournalPlan, error) {
	labels, err := accountLabels(ctx, accounts, tenant, journalLinesOf(planned))
	if err != nil {
		return nil, err
	}

	return &manualJournalPlan{after: planned, accounts: labels}, nil
}

func planManualJournalChange(
	ctx context.Context,
	accounts ledgerAccounts,
	tenant pagination.TenantInfo,
	change *serviceports.ManualJournalChange,
) (*manualJournalPlan, error) {
	sets := [][]serviceports.JournalLinePreview{
		journalLinesOf(change.Before),
		journalLinesOf(change.After),
	}
	if change.Journal != nil {
		sets = append(sets, change.Journal.Lines)
	}
	labels, err := accountLabels(ctx, accounts, tenant, sets...)
	if err != nil {
		return nil, err
	}

	return &manualJournalPlan{
		before:   change.Before,
		after:    change.After,
		journal:  change.Journal,
		accounts: labels,
	}, nil
}

func renderManualJournalDraft(
	_ *serviceports.CreateManualJournalRequest,
	plan *manualJournalPlan,
) (*agent.ToolPreview, error) {
	planned := plan.after
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
		journalMoney(planned.CurrencyCode, plan.accounts, nil, journalLinesOf(planned)),
	)

	return toolpreview.Build(fmt.Sprintf(
		"Would draft the manual journal %q for %s with %s. %s It books nothing until a "+
			"person submits, approves and posts it.",
		planned.Description,
		dayLabel(planned.AccountingDate),
		countOf(len(journalLinesOf(planned)), "line"),
		balanceSentence(planned),
	), change), nil
}

func manualJournalChangeOf(
	plan *manualJournalPlan,
	withLines bool,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Changed(
		manualJournalRecord(plan.before),
		manualJournalViewOf(plan.before),
		manualJournalViewOf(plan.after),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	if withLines {
		toolpreview.AttachMoney(change, journalMoney(
			plan.after.CurrencyCode,
			plan.accounts,
			journalLinesOf(plan.before),
			journalLinesOf(plan.after),
		))
	}

	return change, nil
}

func renderManualJournalRevision(
	_ *journalRevision,
	plan *manualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, true)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would revise draft manual journal %s. %s",
		plan.before.RequestNumber,
		balanceSentence(plan.after),
	), change), nil
}

func renderManualJournalSubmit(
	_ *serviceports.GetManualJournalRequest,
	plan *manualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would submit manual journal %s for approval; it books nothing until a person posts it.",
		plan.before.RequestNumber,
	)
	if plan.after.Status == manualjournal.StatusApproved {
		summary = fmt.Sprintf(
			"Manual journal approval is turned off, so submitting %s approves it as the person "+
				"who submits it; it books nothing until a person posts it.",
			plan.before.RequestNumber,
		)
	}

	return toolpreview.Build(summary, change), nil
}

func renderManualJournalCancel(
	_ *serviceports.CancelManualJournalRequest,
	plan *manualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel manual journal %s (now %s); it is kept and never posted.",
		plan.before.RequestNumber,
		plan.before.Status,
	), change), nil
}

func renderManualJournalPost(
	_ *serviceports.GetManualJournalRequest,
	plan *manualJournalPlan,
) (*agent.ToolPreview, error) {
	change, err := manualJournalChangeOf(plan, false)
	if err != nil {
		return nil, err
	}
	entry, err := bookedEntryChange(
		plan.journal,
		plan.after.CurrencyCode,
		"Journal entry for "+plan.before.RequestNumber,
		plan.accounts,
	)
	if err != nil {
		return nil, err
	}

	sentences := []string{fmt.Sprintf(
		"Would post manual journal %s to the general ledger on %s.",
		plan.before.RequestNumber,
		dayLabel(plan.journal.AccountingDate),
	)}
	if plan.journal.AccountingDate != plan.before.AccountingDate {
		sentences = append(sentences, fmt.Sprintf(
			"Its own day, %s, falls in a closed period, so the closed period policy moves it "+
				"to the next open one.",
			dayLabel(plan.before.AccountingDate),
		))
	}
	sentences = append(sentences, "Only a reversal undoes it.")

	return toolpreview.Build(strings.Join(sentences, " "), change, entry), nil
}

func planReversal(
	ctx context.Context,
	accounts ledgerAccounts,
	tenant pagination.TenantInfo,
	change *serviceports.JournalReversalChange,
) (*reversalPlan, error) {
	plan := &reversalPlan{change: change, accounts: map[pulid.ID]string{}}
	if change.Journal == nil {
		return plan, nil
	}
	labels, err := accountLabels(ctx, accounts, tenant, change.Journal.Lines)
	if err != nil {
		return nil, err
	}
	plan.accounts = labels

	return plan, nil
}

func reversalRecord(entity *journalreversal.Reversal, label string) toolpreview.Record {
	rec := toolpreview.Record{
		Resource: permission.ResourceJournalReversal,
		ID:       entity.ID,
		Label:    label,
	}
	if entity.ID.IsNotNil() {
		rec.Version = pinnedVersion(entity.Version)
	}

	return rec
}

func originalEntryNumber(change *serviceports.JournalReversalChange) string {
	if change.OriginalEntry == nil {
		return "the original entry"
	}

	return "journal entry " + change.OriginalEntry.EntryNumber
}

func renderReversalRequest(
	_ *serviceports.CreateJournalReversalRequest,
	plan *reversalPlan,
) (*agent.ToolPreview, error) {
	after := plan.change.After
	label := "Reversal of " + originalEntryNumber(plan.change)
	change, err := toolpreview.Create(
		reversalRecord(after, label),
		reversalViewOf(after),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, journalMoney(
		plan.change.CurrencyCode,
		plan.accounts,
		nil,
		plan.change.Journal.Lines,
	))

	next := "It waits for an approver, and post_journal_reversal books it once approved."
	if after.Status == journalreversal.StatusApproved {
		next = "Journal approval is turned off, so it is approved at once as the person who " +
			"requests it, and post_journal_reversal books it."
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would request a reversal of %s, booked on %s with every debit and credit swapped. %s",
		originalEntryNumber(plan.change),
		dayLabel(after.RequestedAccountingDate),
		next,
	), change), nil
}

func renderReversalCancel(
	_ *serviceports.CancelJournalReversalRequest,
	plan *reversalPlan,
) (*agent.ToolPreview, error) {
	before := plan.change.Before
	change, err := toolpreview.Changed(
		reversalRecord(before, "Journal reversal "+before.ReasonCode),
		reversalViewOf(before),
		reversalViewOf(plan.change.After),
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

func renderReversalPost(
	_ *serviceports.GetJournalReversalRequest,
	plan *reversalPlan,
) (*agent.ToolPreview, error) {
	before := plan.change.Before
	label := "Reversal of " + originalEntryNumber(plan.change)
	change, err := toolpreview.Changed(
		reversalRecord(before, label),
		reversalViewOf(before),
		reversalViewOf(plan.change.After),
		ledgerOptions()...,
	)
	if err != nil {
		return nil, err
	}
	entry, err := bookedEntryChange(
		plan.change.Journal,
		plan.change.CurrencyCode,
		label,
		plan.accounts,
	)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{change, entry}
	if original := plan.change.OriginalEntry; original != nil {
		reversed, reversedErr := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceJournalEntry,
				ID:       original.ID,
				Label:    "Journal entry " + original.EntryNumber,
				Version:  pinnedVersion(original.Version),
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
		originalEntryNumber(plan.change),
		dayLabel(before.RequestedAccountingDate),
	), changes...), nil
}
