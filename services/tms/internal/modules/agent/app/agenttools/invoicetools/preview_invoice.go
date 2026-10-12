package invoicetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

func (t *PostInvoiceTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
	label := common.InvoiceLabel(before)
	total := money.FormatMinor(before.TotalAmountMinor, before.CurrencyCode)
	if post.Refused() {
		return common.WarnWouldFail(toolpreview.Build(fmt.Sprintf(
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
			Version:  common.PinnedVersion(leg.Before.Version),
		}, leg.Before, leg.After, toolpreview.Only(common.FieldStatus, "billedAt"),
			toolpreview.Volatile("billedAt"))
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, legChange)
	}
	if post.QueueAfter != nil {
		queueChange, changeErr := toolpreview.Changed(
			common.QueueRecord(post.QueueBefore),
			post.QueueBefore,
			post.QueueAfter,
			toolpreview.Only(common.FieldStatus),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, queueChange)
	}
	if post.Journal != nil {
		journalChange, changeErr := journalEntryChange(
			post.Journal,
			post.Before.CurrencyCode,
			"Journal entry for "+common.InvoiceLabel(post.Before),
		)
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
		Label:    common.InvoiceLabel(post.Before),
		Version:  common.PinnedVersion(post.Before.Version),
	}, post.Before, post.After, toolpreview.Only(common.FieldStatus, "postedAt"),
		toolpreview.Volatile("postedAt"))
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		post.Before.CurrencyCode,
		agent.MoneyLine{
			Label: common.MoneyLineFreight,
			After: common.KnownAmount(post.Before.SubtotalAmount),
		},
		agent.MoneyLine{
			Label: common.MoneyLineAccessorials,
			After: common.KnownAmount(post.Before.OtherAmount),
		},
	), toolpreview.SensitiveAs("totalAmount", "subtotalAmount", "otherAmount"))

	return change, nil
}

func journalEntryChange(
	journal *kernelports.JournalPreview,
	currency string,
	label string,
) (*agent.RecordChange, error) {
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
			if view.DebitAccountID.IsNil() {
				view.DebitAccountID = line.GLAccountID
			}
			lines = append(lines, agent.MoneyLine{
				Label: "Debit",
				After: common.KnownAmount(money.DecimalFromMinor(line.DebitMinor)),
			})
		case line.CreditMinor > 0:
			if view.CreditAccountID.IsNil() {
				view.CreditAccountID = line.GLAccountID
			}
		}
	}

	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceJournalEntry,
			Label:    label,
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
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(currency, lines...),
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
		parts = append(parts, common.CountOf(len(post.Legs), "shipment")+" invoiced")
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

	return strings.Join(common.SlicesNonEmpty(sentences), " ")
}

func syncSentence(destinations []accountingsyncports.AccountingSyncDestination) string {
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

func ediSentence(plan *invoiceports.InvoiceEDISendPlan) string {
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

func (t *SendInvoiceTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		common.CountOf(len(attachments), "attachment"),
	)
	if len(plan.Parts) > 1 {
		summary += fmt.Sprintf(" It goes as %d emails, since the attachments are over the "+
			"provider's size for one.", len(plan.Parts))
	}

	if len(plan.Warnings) > 0 {
		summary += " " + strings.Join(plan.Warnings, " ")
	}

	return common.WarnWouldFail(toolpreview.Build(summary, message), send.refusal), nil
}
