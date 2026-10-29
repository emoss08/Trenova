package receivabletools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramAsOfDate = "asOfDate"

const paramUserIDs = "userIds"

const paramShareNote = "note"

const paramShareTab = "tab"

const maxLateCustomers = 200

const lineCountFieldName = "lineCount"

var errNothingToCharge = errors.New(
	"there is nothing to charge as of that date: every overdue period is assessed, or " +
		"what is due is under the organization's minimum",
)

var shareTabs = agenttoolschema.Source("invoice.shareTab", []invoiceshare.ShareTab{
	invoiceshare.ShareTabOverview,
	invoiceshare.ShareTabDelivery,
	invoiceshare.ShareTabCharges,
	invoiceshare.ShareTabDocuments,
	invoiceshare.ShareTabActivity,
})

type lateChargeAssessor interface {
	Assess(
		ctx context.Context,
		req *latechargeports.LateChargeAssessmentRequest,
		actor *permissionports.RequestActor,
	) (*latechargeports.LateChargeAssessmentResult, error)
	PlanAssess(
		ctx context.Context,
		req *latechargeports.LateChargeAssessmentRequest,
		actor *permissionports.RequestActor,
	) (*latechargeports.LateChargeAssessmentResult, error)
}

type invoiceSharer interface {
	Share(
		ctx context.Context,
		req *invoiceports.ShareInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.ShareInvoiceResult, error)
	PreviewShare(
		ctx context.Context,
		req *invoiceports.ShareInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceSharePreview, error)
}

func NewAssessLateChargesTool(charges lateChargeAssessor) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:        "receivable_assess_late_charges",
		SearchTerms: []string{"late fees", "finance charge"},
		Description: "Propose assessing late charges on overdue invoices as of a date, as one " +
			"debit memo per customer. Each has a line per invoice and overdue period at the " +
			"customer's rate, and no period is charged twice. Leave out customerIds to assess " +
			"every customer. A person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Bills customers for paying late; only a person assesses late charges, " +
			"and the billing control may post the memos as they are raised.",
		Properties: map[string]any{
			paramAsOfDate: agenttoolschema.Date(
				"The day overdue periods are counted to, usually today.",
			),
			toolparams.ParamCustomerIDs: toolschema.KeepEmpty(agenttoolschema.RecordIDs(
				permission.ResourceCustomer,
				"Only these customers, from customer_list or "+
					"receivable_list_open_items.",
				maxLateCustomers,
			)),
		},
		Required: []string{paramAsOfDate},
	}, toolspec.ReceivablePlan[*latechargeports.LateChargeAssessmentRequest, *latechargeports.LateChargeAssessmentResult]{
		Request: lateChargeRequest,
		Plan: func(
			ctx context.Context,
			req *latechargeports.LateChargeAssessmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*latechargeports.LateChargeAssessmentResult, error) {
			result, err := charges.PlanAssess(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			if result.TotalChargeMinor <= 0 {
				return nil, errNothingToCharge
			}
			return result, nil
		},
		Refused: func(*latechargeports.LateChargeAssessmentRequest) string {
			return "Would assess late charges."
		},
		Render: renderLateCharges,
		Run: func(
			ctx context.Context,
			req *latechargeports.LateChargeAssessmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := charges.Assess(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func lateChargeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*latechargeports.LateChargeAssessmentRequest, error) {
	asOf, err := toolparams.RequireUTCDay(params.Params, paramAsOfDate)
	if err != nil {
		return nil, err
	}
	req := &latechargeports.LateChargeAssessmentRequest{
		TenantInfo: toolparams.TenantFrom(params),
		AsOfDate:   asOf.Unix(),
	}
	if _, ok := params.Params[toolparams.ParamCustomerIDs]; ok {
		if req.CustomerIDs, err = toolparams.RequirePulidSlice(
			params.Params, toolparams.ParamCustomerIDs, maxLateCustomers,
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
	_ *latechargeports.LateChargeAssessmentRequest,
	result *latechargeports.LateChargeAssessmentResult,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(result.Customers))
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
				toolparams.ParamCustomerID: permission.ResourceCustomer,
			}),
			toolpreview.Labels(map[string]string{lineCountFieldName: "Overdue periods charged"}),
		)
		if err != nil {
			return nil, err
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(customer.CurrencyCode,
			agentpreview.MoneyLine{
				Label: "Late charge",
				After: changepreview.KnownAmount(money.DecimalFromMinor(customer.TotalChargeMinor)),
			},
		), toolpreview.SensitiveAs("totalAmount"))
		changes = append(changes, change)
	}

	posting := "left as drafts for a person to post"
	if result.AutoPost {
		posting = "posted as it is raised, as the billing control says"
	}
	summary := fmt.Sprintf("Would raise %s for %s in all, each %s.",
		changepreview.CountOf(len(changes), "late charge debit memo"),
		money.DecimalFromMinor(result.TotalChargeMinor).StringFixed(2),
		posting)

	return toolpreview.Build(changepreview.WithNotes(summary, notes), changes...), nil
}

func NewShareInvoiceTool(shares invoiceSharer) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:        "invoice_share",
		Recipe:      []string{"invoice_list_share_candidates", "invoice_share"},
		SearchTerms: []string{"loop in teammate", "teammate", "colleague"},
		Description: "Propose sharing an invoice with teammates, who get a notification and an " +
			"email with your note and a link to it. Pick them with " +
			"invoice_list_share_candidates, which lists only people who may read invoices. " +
			"It is sent in the name of the person who approves it, so a person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpRead,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Notifies and emails colleagues in the name of the person who shares it, " +
			"with a note they read as theirs; only that person sends it.",
		Properties: map[string]any{
			toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The invoice, from invoice_list or invoice_get."),
			paramUserIDs: agenttoolschema.RecordIDs(
				permission.ResourceUser,
				"The teammates, from invoice_list_share_candidates.",
				invoiceshare.MaxShareRecipients,
			),
			paramShareNote: toolparams.StringProperty("A note for them, such as what to look at.",
				invoiceshare.MaxShareNoteLength),
			paramShareTab: agenttoolschema.Enum("The part of the invoice the link opens on. "+
				"Defaults to overview.", shareTabs),
		},
		Required: []string{toolparams.ParamInvoiceID, paramUserIDs},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.ShareInvoiceRequest, *invoiceports.InvoiceSharePreview]{
		Request: shareRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.ShareInvoiceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceSharePreview, error) {
			return shares.PreviewShare(ctx, req, params.Actor)
		},
		Refused: func(req *invoiceports.ShareInvoiceRequest) string {
			return fmt.Sprintf("Would share the invoice with %s.",
				changepreview.CountOf(len(req.UserIDs), "teammate"))
		},
		Render: renderShare,
		Run: func(
			ctx context.Context,
			req *invoiceports.ShareInvoiceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := shares.Share(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func shareRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.ShareInvoiceRequest, error) {
	invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	userIDs, err := toolparams.RequirePulidSlice(
		params.Params,
		paramUserIDs,
		invoiceshare.MaxShareRecipients,
	)
	if err != nil {
		return nil, err
	}
	note, err := toolparams.BoundedText(
		params.Params,
		paramShareNote,
		invoiceshare.MaxShareNoteLength,
	)
	if err != nil {
		return nil, err
	}
	tab, _, err := toolparams.OptionalEnum(params.Params, paramShareTab, shareTabs.Values)
	if err != nil {
		return nil, err
	}

	return &invoiceports.ShareInvoiceRequest{
		TenantInfo: toolparams.TenantFrom(params),
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
	_ *invoiceports.ShareInvoiceRequest,
	plan *invoiceports.InvoiceSharePreview,
) (*agentpreview.ToolPreview, error) {
	label := invoicepreview.InvoiceLabel(plan.Invoice)
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Recipients))
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
			change.Message = &agentpreview.MessagePreview{
				Channel: agentpreview.MessageChannelEmail,
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
