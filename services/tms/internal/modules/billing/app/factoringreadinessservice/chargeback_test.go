package factoringreadinessservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSweepWarnsOnceAboutChargebacksComingDue(t *testing.T) {
	t.Parallel()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	submissions := factoringservicetest.NewFakeSubmissions()
	sub, err := submissions.Create(t.Context(), &factoringsubmission.Submission{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Number:         "FS2610000001",
		Status:         factoringsubmission.SubmissionStatusSent,
		Items: []*factoringsubmission.SubmissionItem{
			{
				InvoiceID:     pulid.MustNew("inv_"),
				InvoiceNumber: "INV-1",
				Status:        factoringsubmission.SubmissionItemStatusFunded,
			},
			{
				InvoiceID:     pulid.MustNew("inv_"),
				InvoiceNumber: "INV-2",
				Status:        factoringsubmission.SubmissionItemStatusRejected,
			},
		},
	})
	require.NoError(t, err)
	first, second := sub.Items[0], sub.Items[1]
	attention := &factoringservicetest.FakeAttention{Due: []*factoringwatchtowerports.ChargebackDue{
		{
			ItemID:        first.ID,
			SubmissionID:  sub.ID,
			InvoiceID:     first.InvoiceID,
			InvoiceNumber: first.InvoiceNumber,
			DeadlineAt:    factoringservicetest.Now + 3*timeutils.SecondsPerDay,
		},
		{
			ItemID:        second.ID,
			SubmissionID:  sub.ID,
			InvoiceID:     second.InvoiceID,
			InvoiceNumber: second.InvoiceNumber,
			DeadlineAt:    factoringservicetest.Now + 3*timeutils.SecondsPerDay,
		},
	}}
	events := &factoringservicetest.FakeEvents{}
	svc := NewReadinessService(ReadinessParams{
		Submissions: submissions,
		Attention:   attention,
		Events:      events,
		Logger:      zap.NewNop(),
	}).(*ReadinessService)
	svc.now = func() int64 { return factoringservicetest.Now }

	warned, err := svc.warnChargebacksDue(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 1, warned, "the rejected item is not funded and is skipped")
	require.NotNil(t, attention.Req)
	assert.True(t, attention.Req.UnwarnedOnly)
	assert.Equal(t, factoringservicetest.Now+7*timeutils.SecondsPerDay, attention.Req.Before)
	assert.Equal(t, []string{string(agenttrigger.EventFactoringChargebackDue)}, events.Kinds())

	warned, err = svc.warnChargebacksDue(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Zero(t, warned, "a warned item is not announced again")
}
