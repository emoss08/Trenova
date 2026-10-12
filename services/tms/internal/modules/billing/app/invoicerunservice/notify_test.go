package invoicerunservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingNotifier struct {
	requests []notificationservice.NotifyPermittedRequest
	err      error
}

func (n *recordingNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	n.requests = append(n.requests, req)
	return 1, n.err
}

func scheduledFixture() (*Service, *recordingNotifier, pagination.TenantInfo, periodKey) {
	notifier := &recordingNotifier{}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	key := periodKey{
		orgID:     tenantInfo.OrgID,
		buID:      tenantInfo.BuID,
		cycle:     customer.BillingCycleMonthly,
		periodEnd: 1_759_276_799,
	}

	return &Service{l: zap.NewNop(), notifier: notifier}, notifier, tenantInfo, key
}

func TestNotifyBuildFailedTellsWhoMayCommitOncePerDay(t *testing.T) {
	t.Parallel()

	svc, notifier, tenantInfo, key := scheduledFixture()
	svc.notifyBuildFailed(t.Context(), tenantInfo, key, errors.New("no open fiscal period"))

	require.Len(t, notifier.requests, 1)
	req := notifier.requests[0]
	assert.Equal(t, permission.ResourceInvoiceRun, req.Resource)
	assert.Equal(t, permission.OpApprove, req.Operation)
	assert.Equal(t, tenantInfo, req.Tenant)
	assert.Equal(t, req.Now-int64(scheduledRunNoticeRepeats.Seconds()), req.DedupeSince)
	require.NotNil(t, req.Notification.CorrelationID)
	assert.Contains(t, *req.Notification.CorrelationID, "build:")
	assert.Contains(t, req.Notification.Message, "no open fiscal period")
	assert.Contains(t, req.Notification.Message, "Sep 30, 2025")
	assert.Equal(t, statementsLink, req.Notification.Data["link"])
}

func TestNotifyGroupsFailedNamesTheCustomersLeftUnbilled(t *testing.T) {
	t.Parallel()

	svc, notifier, tenantInfo, key := scheduledFixture()
	runID := pulid.MustNew("invrun_")
	svc.notifyGroupsFailed(t.Context(), tenantInfo, key, &invoicerunports.CommitInvoiceRunResult{
		Run: &invoicerun.InvoiceRun{ID: runID},
		Results: []invoicerunports.CommitGroupResult{
			{GroupLabel: "Acme", Success: true},
			{GroupLabel: "Globex", Error: "missing revenue account"},
			{GroupLabel: "Initech", Skipped: true},
			{GroupLabel: "Umbrella", Error: "customer on credit hold"},
		},
		TotalCount:   4,
		SuccessCount: 1,
		SkippedCount: 1,
		ErrorCount:   2,
	})

	require.Len(t, notifier.requests, 1)
	notice := notifier.requests[0].Notification
	assert.Equal(t, "2 customers were not billed by a scheduled invoice run", notice.Title)
	assert.Contains(t, notice.Message, "billed 1 of 4 customers")
	assert.Contains(t, notice.Message, "Globex: missing revenue account; Umbrella: customer on credit hold.")
	assert.Equal(t, runID.String(), notice.Data["runId"])
	assert.Equal(t, "groups:"+runID.String(), *notice.CorrelationID)
}

func TestNotifyGroupsFailedIsQuietWhenEveryoneWasBilled(t *testing.T) {
	t.Parallel()

	svc, notifier, tenantInfo, key := scheduledFixture()
	svc.notifyGroupsFailed(t.Context(), tenantInfo, key, &invoicerunports.CommitInvoiceRunResult{
		Run:          &invoicerun.InvoiceRun{ID: pulid.MustNew("invrun_")},
		TotalCount:   2,
		SuccessCount: 2,
	})

	assert.Empty(t, notifier.requests)
}

func TestFailedGroupsSummaryCountsWhatItLeavesOut(t *testing.T) {
	t.Parallel()

	results := make([]invoicerunports.CommitGroupResult, 0, 5)
	for _, label := range []string{"A", "B", "C", "D", "E"} {
		results = append(results, invoicerunports.CommitGroupResult{GroupLabel: label, Error: "no"})
	}

	assert.Equal(t, "A: no; B: no; C: no; and 2 more.", failedGroupsSummary(results))
}

func TestNotifyIsBestEffort(t *testing.T) {
	t.Parallel()

	svc, notifier, tenantInfo, key := scheduledFixture()
	notifier.err = errors.New("notifications cannot find who holds a permission")

	assert.NotPanics(t, func() {
		svc.notifyCommitFailed(t.Context(), tenantInfo, key, pulid.MustNew("invrun_").String(), errors.New("x"))
	})

	(&Service{l: zap.NewNop()}).notifyBuildFailed(t.Context(), tenantInfo, key, errors.New("x"))
}
