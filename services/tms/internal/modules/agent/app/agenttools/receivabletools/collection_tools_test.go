package receivabletools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCollections struct {
	collectionports.Service

	guard     *agenttoolstest.WriteGuard
	row       *collectionports.WorklistRow
	logged    []*collectionports.LogActivityRequest
	assigned  []*collectionports.AssignCollectorRequest
	scheduled []*collectionports.ScheduleFollowUpRequest
}

func newFakeCollections() *fakeCollections {
	return &fakeCollections{
		guard: &agenttoolstest.WriteGuard{},
		row: &collectionports.WorklistRow{
			CustomerID:    pulid.MustNew("cus_"),
			CustomerName:  "Acme Foods",
			CurrencyCode:  "USD",
			CollectorID:   pulid.MustNew("usr_"),
			CollectorName: "Dana",
		},
	}
}

func (f *fakeCollections) plan() *collectionports.AccountChangePlan {
	state := collectionports.AccountState{
		CollectorID:   f.row.CollectorID,
		CollectorName: f.row.CollectorName,
	}

	return &collectionports.AccountChangePlan{Row: f.row, Before: state, After: state}
}

func (f *fakeCollections) view() *collectionports.AccountView {
	return &collectionports.AccountView{Row: f.row}
}

func (f *fakeCollections) PlanLogActivity(
	_ context.Context,
	req *collectionports.LogActivityRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	plan := f.plan()
	activity := &collection.Activity{
		CustomerID:   req.CustomerID,
		Kind:         req.Kind,
		Summary:      req.Summary,
		CurrencyCode: "USD",
		PromisedOn:   req.PromisedOn,
		FollowUpOn:   req.FollowUpOn,
	}
	if req.PromisedAmount.Valid {
		amount := req.PromisedAmount.Decimal.Shift(2).IntPart()
		activity.PromisedAmountMinor = &amount
	}
	plan.Activity = activity
	plan.Changed = true
	if activity.IsPromise() {
		plan.After.PromisedOn = activity.PromisedOn
		plan.After.PromisedAmountMinor = activity.PromisedAmountMinor
		dayAfter := *activity.PromisedOn + 86_400
		plan.After.NextFollowUpOn = &dayAfter
	}

	return plan, nil
}

func (f *fakeCollections) LogActivity(
	_ context.Context,
	req *collectionports.LogActivityRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.logged = append(f.logged, req)

	return f.view(), nil
}

func (f *fakeCollections) PlanAssignCollector(
	_ context.Context,
	req *collectionports.AssignCollectorRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	plan := f.plan()
	plan.After.CollectorID = req.CollectorID
	plan.After.CollectorName = ""
	plan.Changed = plan.Before.CollectorID != req.CollectorID
	if plan.Changed {
		plan.Activity = &collection.Activity{
			CustomerID:  req.CustomerID,
			Kind:        collection.ActivityCollectorAssigned,
			Summary:     "Unassigned the collector",
			CollectorID: req.CollectorID,
		}
	}

	return plan, nil
}

func (f *fakeCollections) AssignCollector(
	_ context.Context,
	req *collectionports.AssignCollectorRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.assigned = append(f.assigned, req)

	return f.view(), nil
}

func (f *fakeCollections) PlanScheduleFollowUp(
	_ context.Context,
	req *collectionports.ScheduleFollowUpRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	if err := refusePastFollowUp(req); err != nil {
		return nil, err
	}
	plan := f.plan()
	plan.Activity = &collection.Activity{
		CustomerID: req.CustomerID,
		Kind:       collection.ActivityFollowUpScheduled,
		Summary:    req.Note,
		FollowUpOn: req.FollowUpOn,
	}
	plan.After.NextFollowUpOn = req.FollowUpOn
	plan.Changed = true

	return plan, nil
}

func (f *fakeCollections) ScheduleFollowUp(
	_ context.Context,
	req *collectionports.ScheduleFollowUpRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	if err := refusePastFollowUp(req); err != nil {
		return nil, err
	}
	f.scheduled = append(f.scheduled, req)

	return f.view(), nil
}

func refusePastFollowUp(req *collectionports.ScheduleFollowUpRequest) error {
	if req.FollowUpOn != nil && *req.FollowUpOn < 1_700_000_000 {
		return errortypes.NewValidationError(
			"followUpOn", errortypes.ErrInvalid, "Schedule the follow-up for today or later",
		)
	}

	return nil
}

func previewOf(
	t *testing.T,
	guard *agenttoolstest.WriteGuard,
	tool agenttoolports.AgentTool,
	params agenttoolports.ToolExecuteParams,
) *agent.ToolPreview {
	t.Helper()

	return agenttoolstest.PreviewWithoutWrites(t, guard, func() (*agent.ToolPreview, error) {
		return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	})
}

func TestLogCollectionActivity_APromiseShowsTheEntryAndTheFollowUpItMoves(t *testing.T) {
	t.Parallel()

	collections := newFakeCollections()
	tool := NewLogCollectionActivityTool(collections)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId":     collections.row.CustomerID.String(),
		"kind":           "PromiseToPay",
		"summary":        "AP said the check goes out Friday",
		"contactName":    "Pat in AP",
		"promisedAmount": "1250.00",
		"promisedOn":     "2026-10-23",
	})

	preview := previewOf(t, collections.guard, tool, params)
	require.Len(t, preview.Changes, 2, "the history entry and the account it changes")
	entry := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, entry.Operation)
	account := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationUpdate, account.Operation)
	assert.Contains(t, preview.Summary, "promise to pay")
	assert.Empty(t, collections.logged)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, collections.logged, 1)
	logged := collections.logged[0]
	assert.Equal(t, collections.row.CustomerID, logged.CustomerID)
	assert.Equal(t, collection.ActivityPromiseToPay, logged.Kind)
	assert.Equal(t, "Pat in AP", logged.ContactName)
	require.True(t, logged.PromisedAmount.Valid)
	assert.Equal(t, "1250", logged.PromisedAmount.Decimal.String())
	require.NotNil(t, logged.PromisedOn)
	assert.Equal(t, time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC).Unix(), *logged.PromisedOn)
	assert.Nil(t, logged.FollowUpOn, "no follow-up given leaves the promise to set it")
}

func TestLogCollectionActivity_RefusesAKindNobodyLogs(t *testing.T) {
	t.Parallel()

	collections := newFakeCollections()
	tool := NewLogCollectionActivityTool(collections)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": collections.row.CustomerID.String(),
		"kind":       "CollectorAssigned",
		"summary":    "Took it over",
	})

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Empty(t, collections.logged)
}

func TestLogCollectionActivity_WhatACustomerSaidIsHeldOnceTheRunIsTainted(t *testing.T) {
	t.Parallel()

	policy := NewLogCollectionActivityTool(nil).Policy()
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
	assert.True(t, policy.TaintHold.Applies(agenttoolstest.ExecuteParams(nil)))
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
}

func TestAssignCollector_LeavingTheCollectorOutTakesTheAccountOffThem(t *testing.T) {
	t.Parallel()

	collections := newFakeCollections()
	tool := NewAssignCollectorTool(collections)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": collections.row.CustomerID.String(),
	})

	preview := previewOf(t, collections.guard, tool, params)
	assert.Contains(t, preview.Summary, "off Dana")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, collections.assigned, 1)
	assert.True(t, collections.assigned[0].CollectorID.IsNil())
}

func TestAssignCollector_TheSameCollectorChangesNothing(t *testing.T) {
	t.Parallel()

	collections := newFakeCollections()
	tool := NewAssignCollectorTool(collections)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId":  collections.row.CustomerID.String(),
		"collectorId": collections.row.CollectorID.String(),
	})

	preview := previewOf(t, collections.guard, tool, params)
	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "nothing would change")
}

func TestScheduleCollectionFollowUp_ADayAlreadyOverIsRefused(t *testing.T) {
	t.Parallel()

	collections := newFakeCollections()
	tool := NewScheduleCollectionFollowUpTool(collections)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": collections.row.CustomerID.String(),
		"followUpOn": "2020-01-01",
	})

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Empty(t, collections.scheduled)

	params.Params["followUpOn"] = "2026-11-02"
	params.Params["note"] = "They pay on the first business day"
	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, collections.scheduled, 1)
	assert.Equal(t, "They pay on the first business day", collections.scheduled[0].Note)
}

type fakeReminders struct {
	paymentreminderports.Service

	guard *agenttoolstest.WriteGuard
	sent  []*paymentreminderports.SendNowRequest
}

func (f *fakeReminders) PreviewSendNow(
	_ context.Context,
	req *paymentreminderports.SendNowRequest,
	_ *permissionports.RequestActor,
) ([]*paymentreminderports.SendNowPreview, error) {
	return []*paymentreminderports.SendNowPreview{{
		CustomerID:        req.CustomerID,
		CustomerName:      "Acme Foods",
		CurrencyCode:      "USD",
		InvoiceNumbers:    []string{"INV-1001", "INV-1004"},
		TotalOverdue:      "$3,400.00",
		OldestDaysPastDue: 41,
		Email: &emailports.PlannedEmail{
			To:      []string{"ap@acme.test"},
			CC:      []string{"controller@acme.test"},
			Subject: "Overdue invoices from Trenova Freight",
			Text:    "INV-1001 and INV-1004 are past due.\n" + req.Note,
		},
	}}, nil
}

func (f *fakeReminders) SendNow(
	_ context.Context,
	req *paymentreminderports.SendNowRequest,
	_ *permissionports.RequestActor,
) ([]*paymentreminderports.SendNowResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.sent = append(f.sent, req)

	return []*paymentreminderports.SendNowResult{{
		SentTo:         []string{"ap@acme.test"},
		InvoiceNumbers: []string{"INV-1001", "INV-1004"},
		TotalOverdue:   "$3,400.00",
	}}, nil
}

func TestSendPaymentReminder_PreviewIsTheEmailAndItsRecipients(t *testing.T) {
	t.Parallel()

	reminders := &fakeReminders{guard: &agenttoolstest.WriteGuard{}}
	tool := NewSendPaymentReminderTool(reminders)
	customerID := pulid.MustNew("cus_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": customerID.String(),
		"note":       "Please call us if anything is holding these up.",
	})

	preview := previewOf(t, reminders.guard, tool, params)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationSend, change.Operation)
	require.NotNil(t, change.Message)
	assert.Equal(t, agent.MessageChannelEmail, change.Message.Channel)
	assert.Equal(t, []string{"ap@acme.test"}, change.Message.To)
	assert.Equal(t, []string{"controller@acme.test"}, change.Message.Cc)
	assert.Contains(t, change.Message.Body, "Please call us")
	assert.Contains(t, preview.Summary, "2 overdue invoices")
	assert.Empty(t, reminders.sent)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, reminders.sent, 1)
	assert.Equal(t, customerID, reminders.sent[0].CustomerID)
	assert.Contains(t, result.State, "ap@acme.test")
	require.NotNil(t, result.Record)
	assert.Equal(t, customerID.String(), result.Record.ID)
}

func TestCustomerEmails_LeaveTheOrganizationAndStopAtAskFirst(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		NewSendPaymentReminderTool(nil),
		NewSendCustomerStatementTool(nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress,
			tool.Name())
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
	}
}

type fakeStatements struct {
	customerstatementports.Service

	guard *agenttoolstest.WriteGuard
	asked []*customerstatementports.SendRequest
}

func (f *fakeStatements) PreviewSend(
	_ context.Context,
	req *customerstatementports.SendRequest,
	_ *permissionports.RequestActor,
) (*customerstatementports.SendPreview, error) {
	f.asked = append(f.asked, req)

	return &customerstatementports.SendPreview{
		CustomerID:         req.CustomerID,
		CustomerName:       "Acme Foods",
		PeriodStart:        req.StartDate,
		PeriodEnd:          req.AsOfDate,
		CurrencyCode:       "USD",
		EndingBalanceMinor: 340_000,
		PastDueMinor:       120_000,
		OpenItemCount:      3,
		FileName:           "Statement-ACME-2026-10-15.pdf",
		Email: &emailports.PlannedEmail{
			To:          []string{"ap@acme.test"},
			Subject:     "Statement of account",
			Text:        "Your statement is attached.",
			Attachments: []string{"Statement-ACME-2026-10-15.pdf"},
		},
	}, nil
}

func TestSendCustomerStatement_DatesAreWholeDaysWhereTheCallerIs(t *testing.T) {
	t.Parallel()

	statements := &fakeStatements{guard: &agenttoolstest.WriteGuard{}}
	tool := NewSendCustomerStatementTool(statements)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": pulid.MustNew("cus_").String(),
		"startDate":  "2026-10-01",
		"asOfDate":   "2026-10-15",
	})
	params.Timezone = "America/Chicago"

	preview := previewOf(t, statements.guard, tool, params)
	require.Len(t, statements.asked, 1)
	zone, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, zone).Unix(), statements.asked[0].StartDate)
	endOfDay := time.Date(2026, 10, 16, 0, 0, 0, 0, zone).Unix() - 1
	assert.Equal(t, endOfDay, statements.asked[0].AsOfDate)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, change.Message)
	assert.Equal(t, []string{"Statement-ACME-2026-10-15.pdf"}, change.Message.Attachments)
	assert.Contains(t, preview.Summary, "past due")
}

func TestSendCustomerStatement_RefusesADateItCannotRead(t *testing.T) {
	t.Parallel()

	statements := &fakeStatements{guard: &agenttoolstest.WriteGuard{}}
	tool := NewSendCustomerStatementTool(statements)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"customerId": pulid.MustNew("cus_").String(),
		"asOfDate":   "10/15/2026",
	})

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	assert.Empty(t, statements.asked)
}
