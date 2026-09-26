package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

// journalView is the ledger entry posting books, as a person reads it.
type journalView struct {
	AccountingDate   int64    `json:"accountingDate"`
	FiscalPeriodID   pulid.ID `json:"fiscalPeriodId"`
	EntryStatus      string   `json:"entryStatus"`
	RequiresApproval bool     `json:"requiresApproval"`
	DebitAccountID   pulid.ID `json:"debitAccountId"`
	CreditAccountID  pulid.ID `json:"creditAccountId"`
	Description      string   `json:"description"`
}

func (t *postInvoiceTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := invoiceRequest(t, &params)
	if err != nil {
		return nil, err
	}

	post, err := t.invoices.PreviewPost(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}

	before := post.Before
	label := invoiceLabel(before)
	total := money.FormatMinor(before.TotalAmountMinor, before.CurrencyCode)
	if post.Refusal != nil {
		return warnWouldFail(toolpreview.Build(fmt.Sprintf(
			"Would post %s to %s for %s.", label, before.BillToName, total,
		)), post.Refusal), nil
	}

	changes := make([]*agent.RecordChange, 0, len(post.Legs)+3)
	if !post.AlreadyPosted {
		invoiceChange, changeErr := postedInvoiceChange(post)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, invoiceChange)
	}
	for _, leg := range post.Legs {
		legChange, changeErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       leg.Before.ID,
			Label:    leg.Before.ProNumber,
			Version:  pinnedVersion(leg.Before.Version),
		}, leg.Before, leg.After, toolpreview.Only(fieldStatus, "billedAt"),
			toolpreview.Volatile("billedAt"))
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, legChange)
	}
	if post.QueueAfter != nil {
		queueChange, changeErr := toolpreview.Changed(
			queueRecord(post.QueueBefore),
			post.QueueBefore,
			post.QueueAfter,
			toolpreview.Only(fieldStatus),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, queueChange)
	}
	if post.Journal != nil {
		journalChange, changeErr := journalEntryChange(post)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, journalChange)
	}

	return toolpreview.Build(postSummary(post, label, total), changes...), nil
}

func postedInvoiceChange(post *invoiceservice.PostPreview) (*agent.RecordChange, error) {
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceInvoice,
		ID:       post.Before.ID,
		Label:    invoiceLabel(post.Before),
		Version:  pinnedVersion(post.Before.Version),
	}, post.Before, post.After, toolpreview.Only(fieldStatus, "postedAt"),
		toolpreview.Volatile("postedAt"))
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		post.Before.CurrencyCode,
		agent.MoneyLine{Label: "Freight", After: knownAmount(post.Before.SubtotalAmount)},
		agent.MoneyLine{Label: "Accessorials", After: knownAmount(post.Before.OtherAmount)},
	), toolpreview.SensitiveAs("totalAmount", "subtotalAmount", "otherAmount"))

	return change, nil
}

// journalEntryChange is the entry posting books: the receivable debited and
// revenue credited (the other way round for a credit memo), in the period it
// lands in.
func journalEntryChange(post *invoiceservice.PostPreview) (*agent.RecordChange, error) {
	journal := post.Journal
	view := &journalView{
		AccountingDate:   journal.AccountingDate,
		FiscalPeriodID:   journal.FiscalPeriodID,
		EntryStatus:      journal.EntryStatus,
		RequiresApproval: journal.RequiresApproval,
	}
	lines := make([]agent.MoneyLine, 0, len(journal.Lines))
	for _, line := range journal.Lines {
		view.Description = line.Description
		switch {
		case line.DebitMinor > 0:
			view.DebitAccountID = line.GLAccountID
			lines = append(lines, agent.MoneyLine{
				Label: "Debit",
				After: knownAmount(money.DecimalFromMinor(line.DebitMinor)),
			})
		case line.CreditMinor > 0:
			view.CreditAccountID = line.GLAccountID
		}
	}

	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceJournalEntry,
			Label:    "Journal entry for " + invoiceLabel(post.Before),
		},
		view,
		toolpreview.WithRefs(map[string]permission.Resource{
			"fiscalPeriodId":  permission.ResourceFiscalPeriod,
			"debitAccountId":  permission.ResourceGeneralLedgerAccount,
			"creditAccountId": permission.ResourceGeneralLedgerAccount,
		}),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			"accountingDate": assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(post.Before.CurrencyCode, lines...),
		toolpreview.SensitiveAs("totalAmount"))

	return change, nil
}

func postSummary(post *invoiceservice.PostPreview, label, total string) string {
	if post.AlreadyPosted {
		return fmt.Sprintf(
			"%s is already posted; posting it again only settles its billing queue item.",
			label,
		)
	}

	parts := []string{fmt.Sprintf(
		"Would post %s to %s for %s", label, post.Before.BillToName, total,
	)}
	if len(post.Legs) > 0 {
		parts = append(parts, countOf(len(post.Legs), "shipment")+" invoiced")
	}
	if post.QueueAfter != nil {
		parts = append(parts, "billing queue item "+post.QueueBefore.Number+" posted")
	}
	if post.Journal == nil {
		parts = append(parts, "no ledger entry, as the accounting settings create none for it")
	} else {
		parts = append(parts, fmt.Sprintf("a %s journal entry booked", post.Journal.EntryStatus))
	}

	sentences := []string{strings.Join(parts, "; ") + "."}
	sentences = append(sentences, syncSentence(post.AccountingSync), ediSentence(post.EDI))

	return strings.Join(slicesNonEmpty(sentences), " ")
}

func syncSentence(destinations []serviceports.AccountingSyncDestination) string {
	if len(destinations) == 0 {
		return "No accounting connection takes it."
	}

	names := make([]string, 0, len(destinations))
	for _, destination := range destinations {
		name := destination.Integration
		if destination.Company != "" {
			name += " (" + destination.Company + ")"
		}
		if destination.AwaitsRelease {
			name += ", held until someone releases it"
		}
		names = append(names, name)
	}

	return "It is queued for the accounting system: " + strings.Join(names, "; ") + "."
}

func ediSentence(plan *serviceports.InvoiceEDISendPlan) string {
	switch {
	case plan == nil || !plan.Enabled:
		return ""
	case len(plan.Blockers) > 0:
		return "The customer takes EDI invoices, but no 210 would go: " + plan.Blockers[0] + "."
	case plan.AutoSend:
		return "An EDI 210 would be sent to " + plan.PartnerName + "."
	default:
		return "The customer takes EDI invoices; the 210 waits for someone to send it."
	}
}

func slicesNonEmpty(values []string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}

	return kept
}

func (t *sendInvoiceTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	send, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	plan := send.plan
	attachments := make([]string, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		if part == nil {
			continue
		}
		for _, attachment := range part.Attachments {
			if attachment != nil {
				attachments = append(attachments, attachment.FileName)
			}
		}
	}

	message := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceInvoice,
		ID:       plan.InvoiceID,
	}, &agent.MessagePreview{
		Channel:     agent.MessageChannelEmail,
		From:        plan.FromEmail,
		To:          plan.Recipients.To,
		Cc:          plan.Recipients.CC,
		Bcc:         plan.Recipients.BCC,
		Attachments: attachments,
		Subject:     plan.Subject,
		Body:        plan.Body,
	})

	summary := fmt.Sprintf(
		"Would email the invoice to %s with %s.",
		strings.Join(plan.Recipients.To, ", "),
		countOf(len(attachments), "attachment"),
	)
	if len(plan.Parts) > 1 {
		summary += fmt.Sprintf(" It goes as %d emails, since the attachments are over the "+
			"provider's size for one.", len(plan.Parts))
	}

	if len(plan.Warnings) > 0 {
		summary += " " + strings.Join(plan.Warnings, " ")
	}

	return warnWouldFail(toolpreview.Build(summary, message), send.refusal), nil
}
