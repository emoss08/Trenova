package receivabletools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
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

var shareTabs = agenttoolschema.Source("invoice.shareTab", []invoice.ShareTab{
	invoice.ShareTabOverview,
	invoice.ShareTabDelivery,
	invoice.ShareTabCharges,
	invoice.ShareTabDocuments,
	invoice.ShareTabActivity,
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

func NewAssessLateChargesTool(charges lateChargeAssessor) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
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
			common.ParamCustomerIDs: toolschema.KeepEmpty(agenttoolschema.RecordIDs(
				permission.ResourceCustomer,
				"Only these customers, from customer_list or "+
					"receivable_list_open_items.",
				maxLateCustomers,
			)),
		},
		Required: []string{paramAsOfDate},
	}, common.ReceivablePlan[*latechargeports.LateChargeAssessmentRequest, *latechargeports.LateChargeAssessmentResult]{
		Request: lateChargeRequest,
		Plan: func(
			ctx context.Context,
			req *latechargeports.LateChargeAssessmentRequest,
			params *agentports.ToolExecuteParams,
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
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := charges.Assess(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func lateChargeRequest(
	params *agentports.ToolExecuteParams,
) (*latechargeports.LateChargeAssessmentRequest, error) {
	asOf, err := common.RequireUTCDay(params.Params, paramAsOfDate)
	if err != nil {
		return nil, err
	}
	req := &latechargeports.LateChargeAssessmentRequest{
		TenantInfo: common.TenantFrom(*params),
		AsOfDate:   asOf.Unix(),
	}
	if _, ok := params.Params[common.ParamCustomerIDs]; ok {
		if req.CustomerIDs, err = common.RequirePulidSlice(
			params.Params, common.ParamCustomerIDs, maxLateCustomers,
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
				common.ParamCustomerID: permission.ResourceCustomer,
			}),
			toolpreview.Labels(map[string]string{lineCountFieldName: "Overdue periods charged"}),
		)
		if err != nil {
			return nil, err
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(customer.CurrencyCode,
			agent.MoneyLine{
				Label: "Late charge",
				After: common.KnownAmount(money.DecimalFromMinor(customer.TotalChargeMinor)),
			},
		), toolpreview.SensitiveAs("totalAmount"))
		changes = append(changes, change)
	}

	posting := "left as drafts for a person to post"
	if result.AutoPost {
		posting = "posted as it is raised, as the billing control says"
	}
	summary := fmt.Sprintf("Would raise %s for %s in all, each %s.",
		common.CountOf(len(changes), "late charge debit memo"),
		money.DecimalFromMinor(result.TotalChargeMinor).StringFixed(2),
		posting)

	return toolpreview.Build(common.WithNotes(summary, notes), changes...), nil
}

func NewShareInvoiceTool(shares invoiceSharer) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
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
			common.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The invoice, from invoice_list or invoice_get."),
			paramUserIDs: agenttoolschema.RecordIDs(
				permission.ResourceUser,
				"The teammates, from invoice_list_share_candidates.",
				invoice.MaxShareRecipients,
			),
			paramShareNote: common.StringProperty("A note for them, such as what to look at.",
				invoice.MaxShareNoteLength),
			paramShareTab: agenttoolschema.Enum("The part of the invoice the link opens on. "+
				"Defaults to overview.", shareTabs),
		},
		Required: []string{common.ParamInvoiceID, paramUserIDs},
		Target:   common.TargetInvoice,
	}, common.ReceivablePlan[*invoiceports.ShareInvoiceRequest, *invoiceports.InvoiceSharePreview]{
		Request: shareRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.ShareInvoiceRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceSharePreview, error) {
			return shares.PreviewShare(ctx, req, params.Actor)
		},
		Refused: func(req *invoiceports.ShareInvoiceRequest) string {
			return fmt.Sprintf("Would share the invoice with %s.",
				common.CountOf(len(req.UserIDs), "teammate"))
		},
		Render: renderShare,
		Run: func(
			ctx context.Context,
			req *invoiceports.ShareInvoiceRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := shares.Share(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func shareRequest(
	params *agentports.ToolExecuteParams,
) (*invoiceports.ShareInvoiceRequest, error) {
	invoiceID, err := common.RequirePulid(params.Params, common.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	userIDs, err := common.RequirePulidSlice(
		params.Params,
		paramUserIDs,
		invoice.MaxShareRecipients,
	)
	if err != nil {
		return nil, err
	}
	note, err := common.BoundedText(params.Params, paramShareNote, invoice.MaxShareNoteLength)
	if err != nil {
		return nil, err
	}
	tab, _, err := common.OptionalEnum(params.Params, paramShareTab, shareTabs.Values)
	if err != nil {
		return nil, err
	}

	return &invoiceports.ShareInvoiceRequest{
		TenantInfo: common.TenantFrom(*params),
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
) (*agent.ToolPreview, error) {
	label := common.InvoiceLabel(plan.Invoice)
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
