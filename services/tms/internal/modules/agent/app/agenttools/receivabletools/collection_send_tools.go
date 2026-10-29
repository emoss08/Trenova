package receivabletools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolSendPaymentReminder   = "receivable_send_payment_reminder"
	toolSendCustomerStatement = "receivable_send_statement"
	paramStatementStart       = "startDate"
	paramStatementAsOf        = "asOfDate"
	maxCustomerEmailNoteChars = 2000
)

type reminderView struct {
	Currency          string   `json:"currency"`
	Invoices          []string `json:"invoices"`
	TotalOverdue      string   `json:"totalOverdue"`
	OldestDaysPastDue int      `json:"oldestDaysPastDue"`
}

type statementView struct {
	PeriodStart   int64  `json:"periodStart,omitempty"`
	PeriodEnd     int64  `json:"periodEnd"`
	EndingBalance string `json:"endingBalance"`
	PastDue       string `json:"pastDue,omitempty"`
	OpenItems     int    `json:"openItems"`
	Attachment    string `json:"attachment"`
}

func customerNoteProperty(what string) map[string]any {
	return toolparams.StringProperty(what, maxCustomerEmailNoteChars)
}

func NewSendPaymentReminderTool(reminders paymentreminderports.Service) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolSendPaymentReminder,
			Recipe: []string{
				toolListCollectionsWorklist,
				toolGetCustomerStatement,
				toolSendPaymentReminder,
			},
			Description: "Email a customer a reminder of every overdue invoice it has that is " +
				"not in dispute, with an optional note. It is one email per currency, to the " +
				"invoice recipients on its email profile, and it goes whatever the reminder " +
				"schedule says, " +
				"so use it when a collector asks or a promise to pay was broken, never for a " +
				"customer waiting on a promise that has not yet fallen due. The email is entered " +
				"in the customer's collections history.",
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressExternalRecipient,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Emails the customer about what it owes, which cannot be called back; a " +
				"person approves every reminder and sees the recipients and the email first.",
			Target:   collectionTarget,
			Artifact: changepreview.CustomerRecordEntity,
			Properties: map[string]any{
				toolparams.ParamCustomerID: collectionCustomerProperty(),
				paramCollectionNote: customerNoteProperty(
					"A short note the customer reads above the list of invoices. Leave it out " +
						"unless the person asked for one; never repeat the amounts the email " +
						"already lists.",
				),
			},
			Required: []string{toolparams.ParamCustomerID},
			SearchTerms: []string{
				"payment reminder",
				"dunning email",
				"overdue reminder",
				"chase payment",
			},
		},
		toolspec.ReceivablePlan[
			*paymentreminderports.SendNowRequest,
			[]*paymentreminderports.SendNowPreview,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*paymentreminderports.SendNowRequest, error) {
				customerID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamCustomerID,
				)
				if err != nil {
					return nil, err
				}
				note, err := toolparams.BoundedText(
					params.Params,
					paramCollectionNote,
					maxCustomerEmailNoteChars,
				)
				if err != nil {
					return nil, err
				}
				return &paymentreminderports.SendNowRequest{
					TenantInfo: toolparams.TenantFrom(*params),
					CustomerID: customerID,
					Note:       note,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *paymentreminderports.SendNowRequest,
				params *agenttoolports.ToolExecuteParams,
			) ([]*paymentreminderports.SendNowPreview, error) {
				return reminders.PreviewSendNow(ctx, req, params.Actor)
			},
			Refused: func(*paymentreminderports.SendNowRequest) string {
				return "Would email the customer a payment reminder."
			},
			Render: renderReminders,
			Run: func(
				ctx context.Context,
				req *paymentreminderports.SendNowRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				sent, err := reminders.SendNow(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				states := make([]string, 0, len(sent))
				for _, reminder := range sent {
					states = append(states, fmt.Sprintf("%s for %s sent to %s",
						reminder.TotalOverdue,
						strings.Join(reminder.InvoiceNumbers, ", "),
						strings.Join(reminder.SentTo, ", ")))
				}
				return &agentrecord.ToolExecutionResult{
					Action: "sent",
					Kind:   "payment reminder",
					IDs:    map[string]string{toolparams.ParamCustomerID: req.CustomerID.String()},
					Record: customerRecord(req.CustomerID),
					State:  strings.Join(states, "; "),
				}, nil
			},
		},
	)
}

func renderReminders(
	req *paymentreminderports.SendNowRequest,
	previews []*paymentreminderports.SendNowPreview,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(previews))
	totals := make([]string, 0, len(previews))
	invoices := 0
	name := "the customer"
	for _, preview := range previews {
		name = preview.CustomerName
		invoices += len(preview.InvoiceNumbers)
		totals = append(totals, preview.TotalOverdue)
		change, err := toolpreview.Create(toolpreview.Record{
			Resource: permission.ResourceCustomer,
			ID:       req.CustomerID,
			Label: fmt.Sprintf(
				"Payment reminder to %s (%s)",
				preview.CustomerName,
				preview.CurrencyCode,
			),
		}, &reminderView{
			Currency:          preview.CurrencyCode,
			Invoices:          preview.InvoiceNumbers,
			TotalOverdue:      preview.TotalOverdue,
			OldestDaysPastDue: preview.OldestDaysPastDue,
		})
		if err != nil {
			return nil, err
		}
		change.Operation = agentpreview.PreviewOperationSend
		change.Message = outboundpreview.PlannedEmailPreview(preview.Email)
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would email %s a payment reminder for %s totalling %s.",
		name, changepreview.CountOf(invoices, "overdue invoice"), strings.Join(totals, " and "))

	return toolpreview.Build(summary, changes...), nil
}

func NewSendCustomerStatementTool(
	statements customerstatementports.Service,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolSendCustomerStatement,
			Recipe: []string{
				toolGetCustomerStatement,
				toolSendCustomerStatement,
			},
			Description: "Email a customer its statement of account as a PDF, to the invoice " +
				"recipients on its email profile, with an optional note. The statement runs from " +
				"startDate (or the account's first transaction) to asOfDate (or today). Read " +
				"it with receivable_get_statement first so you know what it says. The email is " +
				"entered in the customer's collections history.",
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressExternalRecipient,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Emails the customer its balance and open items, which cannot be " +
				"called back; a person approves every statement and sees the recipients and the " +
				"email first.",
			Target:   collectionTarget,
			Artifact: changepreview.CustomerRecordEntity,
			Properties: map[string]any{
				toolparams.ParamCustomerID: collectionCustomerProperty(),
				paramStatementStart: agenttoolschema.Date(
					"The first day the statement covers. Leave it out to start from the " +
						"account's first transaction.",
				),
				paramStatementAsOf: agenttoolschema.Date(
					"The statement date, today or earlier. Leave it out for today.",
				),
				paramCollectionNote: customerNoteProperty(
					"A short note the customer reads in the email. Leave it out unless the " +
						"person asked for one.",
				),
			},
			Required: []string{toolparams.ParamCustomerID},
			SearchTerms: []string{
				"email statement",
				"send statement",
				"statement pdf",
			},
		},
		toolspec.ReceivablePlan[
			*customerstatementports.SendRequest,
			*customerstatementports.SendPreview,
		]{
			Request: statementRequest,
			Plan: func(
				ctx context.Context,
				req *customerstatementports.SendRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*customerstatementports.SendPreview, error) {
				return statements.PreviewSend(ctx, req, params.Actor)
			},
			Refused: func(*customerstatementports.SendRequest) string {
				return "Would email the customer its statement of account."
			},
			Render: renderStatement,
			Run: func(
				ctx context.Context,
				req *customerstatementports.SendRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				sent, err := statements.Send(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return &agentrecord.ToolExecutionResult{
					Action: "sent",
					Kind:   "customer statement",
					Name:   sent.FileName,
					IDs:    map[string]string{toolparams.ParamCustomerID: req.CustomerID.String()},
					Record: customerRecord(req.CustomerID),
					State: fmt.Sprintf("%s due, sent to %s",
						sent.EndingBalance, strings.Join(sent.SentTo, ", ")),
				}, nil
			},
		},
	)
}

func statementRequest(
	params *agenttoolports.ToolExecuteParams,
) (*customerstatementports.SendRequest, error) {
	customerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCustomerID)
	if err != nil {
		return nil, err
	}
	note, err := toolparams.BoundedText(
		params.Params,
		paramCollectionNote,
		maxCustomerEmailNoteChars,
	)
	if err != nil {
		return nil, err
	}

	zone := timeutils.LoadLocation(params.Timezone)
	req := &customerstatementports.SendRequest{
		StatementRequest: customerstatementports.StatementRequest{
			TenantInfo: toolparams.TenantFrom(*params),
			CustomerID: customerID,
		},
		Note: note,
	}
	if raw := optionalText(params.Params, paramStatementStart); raw != "" {
		start, parseErr := timeutils.ParseCalendarDate(raw, zone)
		if parseErr != nil {
			return nil, toolparams.DateRefusal(paramStatementStart, raw, toolschema.FormatDate)
		}
		req.StartDate = start
	}
	if raw := optionalText(params.Params, paramStatementAsOf); raw != "" {
		day, parseErr := timeutils.ParseCalendarDate(raw, zone)
		if parseErr != nil {
			return nil, toolparams.DateRefusal(paramStatementAsOf, raw, toolschema.FormatDate)
		}
		req.AsOfDate = timeutils.NextDayStart(day, zone) - 1
	}

	return req, nil
}

func optionalText(params map[string]any, key string) string {
	return strings.TrimSpace(toolparams.OptionalString(params, key))
}

func renderStatement(
	req *customerstatementports.SendRequest,
	preview *customerstatementports.SendPreview,
) (*agentpreview.ToolPreview, error) {
	view := &statementView{
		PeriodStart:   preview.PeriodStart,
		PeriodEnd:     preview.PeriodEnd,
		EndingBalance: money.FormatMinor(preview.EndingBalanceMinor, preview.CurrencyCode),
		OpenItems:     preview.OpenItemCount,
		Attachment:    preview.FileName,
	}
	if preview.PastDueMinor > 0 {
		view.PastDue = money.FormatMinor(preview.PastDueMinor, preview.CurrencyCode)
	}
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceCustomer,
		ID:       req.CustomerID,
		Label:    "Statement of account for " + preview.CustomerName,
	}, view, toolpreview.Types(map[string]assistantartifact.DisplayType{
		"periodStart": assistantartifact.DisplayDate,
		"periodEnd":   assistantartifact.DisplayDate,
	}))
	if err != nil {
		return nil, err
	}
	change.Operation = agentpreview.PreviewOperationSend
	change.Message = outboundpreview.PlannedEmailPreview(preview.Email)

	summary := fmt.Sprintf("Would email %s its statement of account showing %s due.",
		preview.CustomerName, view.EndingBalance)
	if view.PastDue != "" {
		summary = fmt.Sprintf("Would email %s its statement of account showing %s due, %s of it "+
			"past due.", preview.CustomerName, view.EndingBalance, view.PastDue)
	}

	return toolpreview.Build(summary, change), nil
}
