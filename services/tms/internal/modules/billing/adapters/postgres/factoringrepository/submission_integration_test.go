//go:build integration

package factoringrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newSubmissionRepos(e *factoringEnv) (factoringports.SubmissionRepository, factoringports.TransportRepository) {
	p := Params{DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop()}
	return NewSubmissionRepository(p), NewTransportRepository(p)
}

func (e *factoringEnv) submission(
	agreement *factoring.Agreement,
	number, key string,
	invoices ...*invoice.Invoice,
) *factoring.Submission {
	sub := &factoring.Submission{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		Number:             number,
		FactoringCompanyID: agreement.FactoringCompanyID,
		AgreementID:        agreement.ID,
		Status:             factoring.SubmissionStatusSending,
		CurrencyCode:       "USD",
		TransportMethod:    factoring.TransportMethodEmail,
		TransportTarget:    "schedules@factor.test",
		IdempotencyKey:     key,
	}
	for _, inv := range invoices {
		funding := factoring.ExpectedFunding(inv.TotalAmountMinor, agreement, 30)
		sub.Items = append(sub.Items, &factoring.SubmissionItem{
			InvoiceID:            inv.ID,
			InvoiceNumber:        inv.Number,
			CustomerID:           inv.CustomerID,
			FaceMinor:            funding.FaceMinor,
			ExpectedAdvanceMinor: funding.AdvanceMinor,
			ExpectedFeeMinor:     funding.FeeMinor,
			ExpectedReserveMinor: funding.ReserveMinor,
			ExpectedDays:         funding.Days,
		})
	}
	sub.Totalize()
	return sub
}

func TestSubmissionLifecycleAndLiveInvoices(t *testing.T) {
	e := setupFactoringEnv(t)
	submissions, _ := newSubmissionRepos(e)
	factor := e.createFactor(t, "FSUB")
	agreement, err := e.agreements.Create(e.ctx, e.agreement(factor.ID, time.Now().Add(-time.Hour).Unix(), nil))
	require.NoError(t, err)

	first := e.postInvoice(t, 100000, &factor.ID)
	second := e.postInvoice(t, 50000, &factor.ID)

	created, err := submissions.Create(e.ctx, e.submission(agreement, "FS-1", "key-1", first, second))
	require.NoError(t, err)
	require.Len(t, created.Items, 2)
	assert.Equal(t, 1, created.Items[0].Position)
	assert.Equal(t, int64(150000), created.FaceMinor)

	_, err = submissions.Create(e.ctx, e.submission(agreement, "FS-2", "key-2", second))
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	_, err = submissions.Create(e.ctx, e.submission(agreement, "FS-3", "key-1", e.postInvoice(t, 1000, &factor.ID)))
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	live, err := submissions.ListLiveItems(e.ctx, &factoringports.ListLiveItemsRequest{
		TenantInfo: e.tenantInfo,
		InvoiceIDs: []pulid.ID{first.ID, second.ID},
	})
	require.NoError(t, err)
	require.Len(t, live, 2)
	assert.Equal(t, "FS-1", live[first.ID].SubmissionNumber)

	byKey, err := submissions.GetByIdempotencyKey(e.ctx, factoringports.GetSubmissionByKeyRequest{
		TenantInfo:     e.tenantInfo,
		IdempotencyKey: "key-1",
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, byKey.ID)
	assert.Len(t, byKey.Items, 2)

	require.NoError(t, submissions.SetDocuments(e.ctx, &factoringports.SetSubmissionDocumentsRequest{
		TenantInfo:            e.tenantInfo,
		SubmissionID:          created.ID,
		SchedulePDFDocumentID: pulid.MustNew("doc_"),
		Packets:               map[pulid.ID]pulid.ID{created.Items[0].ID: pulid.MustNew("doc_")},
	}))

	loaded, err := submissions.GetByID(e.ctx, factoringports.GetSubmissionRequest{
		ID: created.ID, TenantInfo: e.tenantInfo, IncludeItems: true,
	})
	require.NoError(t, err)
	assert.False(t, loaded.SchedulePDFDocumentID.IsNil())
	assert.True(t, loaded.ScheduleCSVDocumentID.IsNil())
	assert.False(t, loaded.Items[0].PacketDocumentID.IsNil())

	require.NoError(t, loaded.MarkFailed(time.Now().Unix(), "connection refused", 6))
	failedVersion := loaded.Version
	loaded, err = submissions.Update(e.ctx, loaded)
	require.NoError(t, err)
	assert.Equal(t, failedVersion+1, loaded.Version)

	failed, err := NewWatchtowerReader(Params{DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop()}).
		ListFailedSubmissions(e.ctx, &factoringports.ListFailedSubmissionsRequest{TenantInfo: e.tenantInfo})
	require.NoError(t, err)
	require.Len(t, failed, 1)
	assert.Equal(t, "FS-1", failed[0].Number)
	assert.Equal(t, factor.Name, failed[0].FactorName)
	assert.Equal(t, "connection refused", failed[0].LastError)

	stale := *loaded
	stale.Version = failedVersion
	_, err = submissions.Update(e.ctx, &stale)
	require.Error(t, err)

	require.NoError(t, loaded.Cancel(time.Now().Unix(), "Factor asked to resend"))
	_, err = submissions.Update(e.ctx, loaded)
	require.NoError(t, err)

	live, err = submissions.ListLiveItems(e.ctx, &factoringports.ListLiveItemsRequest{
		TenantInfo: e.tenantInfo,
		InvoiceIDs: []pulid.ID{first.ID, second.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, live)

	again, err := submissions.Create(e.ctx, e.submission(agreement, "FS-4", "key-4", second))
	require.NoError(t, err)

	items, err := submissions.ListItemsBySubmissionIDs(e.ctx, &factoringports.ListItemsBySubmissionIDsRequest{
		TenantInfo:    e.tenantInfo,
		SubmissionIDs: []pulid.ID{created.ID, again.ID},
	})
	require.NoError(t, err)
	assert.Len(t, items[created.ID], 2)
	assert.Equal(t, factoring.SubmissionItemStatusReleased, items[created.ID][0].Status)
	assert.Len(t, items[again.ID], 1)

	page, err := submissions.ListConnection(e.ctx, &factoringports.ListSubmissionConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: e.tenantInfo, Query: second.Number},
		Cursor: pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
	})
	require.NoError(t, err)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 2, *page.TotalCount)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err = submissions.GetByID(e.ctx, factoringports.GetSubmissionRequest{ID: created.ID, TenantInfo: other})
	require.Error(t, err)
}

func TestTransportOnePerFactor(t *testing.T) {
	e := setupFactoringEnv(t)
	_, transports := newSubmissionRepos(e)
	factor := e.createFactor(t, "FTRN")

	entity := &factoring.Transport{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		FactoringCompanyID: factor.ID,
		Method:             factoring.TransportMethodEmail,
		EmailRecipients:    []string{"schedules@factor.test"},
		AttachPackets:      true,
	}
	created, err := transports.Create(e.ctx, entity)
	require.NoError(t, err)

	_, err = transports.Create(e.ctx, &factoring.Transport{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		FactoringCompanyID: factor.ID,
		Method:             factoring.TransportMethodEmail,
		EmailRecipients:    []string{"other@factor.test"},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	created.Method = factoring.TransportMethodSftp
	created.EmailRecipients = nil
	created.SftpHost = "sftp.factor.test"
	created.SftpPort = 22
	created.SftpUsername = "carrier"
	created.SftpAuthMethod = factoring.SftpAuthPassword
	created.SftpSecret = "sealed"
	created.SftpKnownHostKey = "ssh-ed25519 AAAA"
	created.SftpDirectory = "/in"
	updated, err := transports.Update(e.ctx, created)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.Version)

	updated.RecordTest(time.Now().Unix(), nil)
	require.NoError(t, transports.RecordTest(e.ctx, updated))

	got, err := transports.Get(e.ctx, factoringports.GetTransportRequest{
		TenantInfo: e.tenantInfo, FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, factoring.TransportMethodSftp, got.Method)
	assert.Equal(t, "sealed", got.SftpSecret)
	assert.NotNil(t, got.LastTestedAt)
}

func TestTransportKeepsPacketsOffWhenCreatedWithout(t *testing.T) {
	e := setupFactoringEnv(t)
	_, transports := newSubmissionRepos(e)
	factor := e.createFactor(t, "FNOP")

	_, err := transports.Create(e.ctx, &factoring.Transport{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		FactoringCompanyID: factor.ID,
		Method:             factoring.TransportMethodEmail,
		EmailRecipients:    []string{"schedules@factor.test"},
		AttachPackets:      false,
	})
	require.NoError(t, err)

	got, err := transports.Get(e.ctx, factoringports.GetTransportRequest{
		TenantInfo: e.tenantInfo, FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.False(t, got.AttachPackets)
}
