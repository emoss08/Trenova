package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramAsOfDate      = "asOfDate"
	paramUserIDs       = "userIds"
	paramShareNote     = "note"
	paramShareTab      = "tab"
	maxLateCustomers   = 200
	lineCountFieldName = "lineCount"
)

var (
	errNothingToCharge = errors.New(
		"there is nothing to charge as of that date: every overdue period is assessed, or " +
			"what is due is under the organization's minimum",
	)
	shareTabs = []invoice.ShareTab{
		invoice.ShareTabOverview,
		invoice.ShareTabDelivery,
		invoice.ShareTabCharges,
		invoice.ShareTabDocuments,
		invoice.ShareTabActivity,
	}
)

type lateChargeAssessor interface {
	Assess(
		ctx context.Context,
		req *serviceports.LateChargeAssessmentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.LateChargeAssessmentResult, error)
	PlanAssess(
		ctx context.Context,
		req *serviceports.LateChargeAssessmentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.LateChargeAssessmentResult, error)
}

type invoiceSharer interface {
	Share(
		ctx context.Context,
		req *serviceports.ShareInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ShareInvoiceResult, error)
	PreviewShare(
		ctx context.Context,
		req *serviceports.ShareInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceSharePreview, error)
}

func newAssessLateChargesTool(charges lateChargeAssessor) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "assess_late_charges",
		description: "Propose assessing late charges on overdue invoices as of a date: one debit " +
			"memo per customer with a line per invoice and overdue period, at each customer's " +
			"rate, never charging a period twice. Leave out customerIds to assess every customer. " +
			"A person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpCreate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Bills customers for paying late; only a person assesses late charges, " +
			"and the billing control may post the memos as they are raised.",
		properties: map[string]any{
			paramAsOfDate: dayProperty("The day overdue periods are counted to, usually today;"),
			paramCustomerIDs: idListProperty("Only these customers, from list_customers or "+
				"list_ar_open_items.", maxLateCustomers),
		},
		required: []string{paramAsOfDate},
	}, receivablePlan[*serviceports.LateChargeAssessmentRequest, *serviceports.LateChargeAssessmentResult]{
		request: lateChargeRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.LateChargeAssessmentRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.LateChargeAssessmentResult, error) {
			result, err := charges.PlanAssess(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			if result.TotalChargeMinor <= 0 {
				return nil, errNothingToCharge
			}
			return result, nil
		},
		refused: func(*serviceports.LateChargeAssessmentRequest) string {
			return "Would assess late charges."
		},
		render: renderLateCharges,
		run: func(
			ctx context.Context,
			req *serviceports.LateChargeAssessmentRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := charges.Assess(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func lateChargeRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.LateChargeAssessmentRequest, error) {
	asOf, err := requireUTCDay(params.Params, paramAsOfDate)
	if err != nil {
		return nil, err
	}
	req := &serviceports.LateChargeAssessmentRequest{
		TenantInfo: tenantFrom(*params),
		AsOfDate:   asOf.Unix(),
	}
	if _, ok := params.Params[paramCustomerIDs]; ok {
		if req.CustomerIDs, err = requirePulidSlice(
			params.Params, paramCustomerIDs, maxLateCustomers,
		); err != nil {
			return nil, err
		}
	}

	return req, nil
}

type lateChargeMemoView struct {
	CustomerID pulid.ID `json:"customerId"`
	LineCount  int      `json:"lineCount"`
	Invoices   string   `json:"invoices"`
}

func renderLateCharges(
	_ *serviceports.LateChargeAssessmentRequest,
	result *serviceports.LateChargeAssessmentResult,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(result.Customers))
	notes := make([]string, 0, len(result.Customers))
	for _, customer := range result.Customers {
		if customer.Skipped {
			notes = append(notes, fmt.Sprintf("%s is skipped: %s",
				customer.CustomerName, customer.SkipReason))
			continue
		}
		invoices := make([]string, 0, len(customer.Lines))
		seen := make(map[string]struct{}, len(customer.Lines))
		for _, line := range customer.Lines {
			if _, ok := seen[line.InvoiceNumber]; ok {
				continue
			}
			seen[line.InvoiceNumber] = struct{}{}
			invoices = append(invoices, line.InvoiceNumber)
		}
		change, err := toolpreview.Create(
			toolpreview.Record{
				Resource: permission.ResourceInvoice,
				Label:    "Late charge debit memo for " + customer.CustomerName,
			},
			&lateChargeMemoView{
				CustomerID: customer.CustomerID,
				LineCount:  len(customer.Lines),
				Invoices:   strings.Join(invoices, ", "),
			},
			toolpreview.WithRefs(map[string]permission.Resource{
				paramCustomerID: permission.ResourceCustomer,
			}),
			toolpreview.Labels(map[string]string{lineCountFieldName: "Overdue periods charged"}),
		)
		if err != nil {
			return nil, err
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(customer.CurrencyCode,
			agent.MoneyLine{
				Label: "Late charge",
				After: knownAmount(money.DecimalFromMinor(customer.TotalChargeMinor)),
			},
		), toolpreview.SensitiveAs("totalAmount"))
		changes = append(changes, change)
	}

	posting := "left as drafts for a person to post"
	if result.AutoPost {
		posting = "posted as it is raised, as the billing control says"
	}
	summary := fmt.Sprintf("Would raise %s for %s in all, each %s.",
		countOf(len(changes), "late charge debit memo"),
		money.DecimalFromMinor(result.TotalChargeMinor).StringFixed(2),
		posting)

	return toolpreview.Build(withNotes(summary, notes), changes...), nil
}

func newShareInvoiceTool(shares invoiceSharer) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "share_invoice",
		description: "Propose sharing an invoice with teammates, who get a notification and an " +
			"email with your note and a link to it. Pick them with " +
			"list_invoice_share_candidates, which lists only people who may read invoices. " +
			"It is sent in the name of the person who approves it, so a person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpRead,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Notifies and emails colleagues in the name of the person who shares it, " +
			"with a note they read as theirs; only that person sends it.",
		properties: map[string]any{
			paramInvoiceID: stringProperty("The invoice, from list_invoices or get_invoice.", 0),
			paramUserIDs: idListProperty("The teammates, from list_invoice_share_candidates.",
				invoice.MaxShareRecipients),
			paramShareNote: stringProperty("A note for them, such as what to look at.",
				invoice.MaxShareNoteLength),
			paramShareTab: enumProperty("The part of the invoice the link opens on. Defaults to "+
				"overview.", enumNames(shareTabs)),
		},
		required: []string{paramInvoiceID, paramUserIDs},
		target:   targetInvoice,
	}, receivablePlan[*serviceports.ShareInvoiceRequest, *serviceports.InvoiceSharePreview]{
		request: shareRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ShareInvoiceRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceSharePreview, error) {
			return shares.PreviewShare(ctx, req, params.Actor)
		},
		refused: func(req *serviceports.ShareInvoiceRequest) string {
			return fmt.Sprintf("Would share the invoice with %s.",
				countOf(len(req.UserIDs), "teammate"))
		},
		render: renderShare,
		run: func(
			ctx context.Context,
			req *serviceports.ShareInvoiceRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := shares.Share(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func shareRequest(params *serviceports.ToolExecuteParams) (*serviceports.ShareInvoiceRequest, error) {
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	userIDs, err := requirePulidSlice(params.Params, paramUserIDs, invoice.MaxShareRecipients)
	if err != nil {
		return nil, err
	}
	note, err := boundedText(params.Params, paramShareNote, invoice.MaxShareNoteLength)
	if err != nil {
		return nil, err
	}
	tab, err := optionalEnum(params.Params, paramShareTab, shareTabs)
	if err != nil {
		return nil, err
	}

	return &serviceports.ShareInvoiceRequest{
		TenantInfo: tenantFrom(*params),
		InvoiceID:  invoiceID,
		UserIDs:    userIDs,
		Note:       note,
		Tab:        tab,
	}, nil
}

type shareView struct {
	SharedWithID pulid.ID `json:"sharedWithId"`
	Tab          string   `json:"tab"`
	Note         string   `json:"note,omitempty"`
}

func renderShare(
	_ *serviceports.ShareInvoiceRequest,
	plan *serviceports.InvoiceSharePreview,
) (*agent.ToolPreview, error) {
	label := invoiceLabel(plan.Invoice)
	changes := make([]*agent.RecordChange, 0, len(plan.Recipients))
	names := make([]string, 0, len(plan.Recipients))
	for _, recipient := range plan.Recipients {
		names = append(names, recipient.Name)
		change, err := toolpreview.Create(
			toolpreview.Record{
				Resource: permission.ResourceInvoice,
				Label:    label + " shared with " + recipient.Name,
			},
			&shareView{SharedWithID: recipient.UserID, Tab: string(plan.Tab), Note: plan.Note},
			toolpreview.WithRefs(map[string]permission.Resource{
				"sharedWithId": permission.ResourceUser,
			}),
		)
		if err != nil {
			return nil, err
		}
		if recipient.Emailed {
			change.Message = &agent.MessagePreview{
				Channel: agent.MessageChannelEmail,
				To:      []string{recipient.EmailAddress},
				Subject: recipient.Subject,
				Body:    recipient.Body,
			}
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would share %s with %s in %s's name; each gets a notification",
		label, strings.Join(names, ", "), plan.SharedByName)
	if plan.EmailConfigured {
		summary += " and an email."
	} else {
		summary += "; no email goes, as outbound email is not configured."
	}

	return toolpreview.Build(summary, changes...), nil
}
