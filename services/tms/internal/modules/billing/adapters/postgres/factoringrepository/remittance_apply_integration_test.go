//go:build integration

package factoringrepository

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringremittanceservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type remittanceFiles struct {
	documentports.InvoiceDocumentService
	docs     map[pulid.ID]*document.Document
	contents map[pulid.ID][]byte
}

func (f *remittanceFiles) Get(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return f.docs[req.ID], nil
}

func (f *remittanceFiles) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	return &documentports.DocumentContent{
		Body: io.NopCloser(bytes.NewReader(f.contents[req.ID])),
	}, nil
}

func minorText(minor int64) string {
	return strconv.FormatInt(minor/100, 10) + "." + strconv.FormatInt(100+minor%100, 10)[1:]
}

func TestRemittanceAppliesThroughTheLedgerInOneTransaction(t *testing.T) {
	e := setupLedgerEnv(t)
	actor := testutil.NewSessionActor(e.userID, e.tenantInfo.OrgID, e.tenantInfo.BuID)
	now := time.Now().Unix()
	conn := postgres.NewTestConnection(e.db)
	remittances := NewRemittanceRepository(Params{DB: conn, Logger: zap.NewNop()})
	files := &remittanceFiles{
		docs:     map[pulid.ID]*document.Document{},
		contents: map[pulid.ID][]byte{},
	}
	svc := factoringremittanceservice.NewRemittanceService(
		factoringremittanceservice.RemittanceParams{
			DB:          conn,
			Remittances: remittances,
			Submissions: e.submissions,
			Ledger:      e.svc,
			Factors:     e.factors,
			Documents:   files,
			Audit:       &mocks.NoopAuditService{},
			Logger:      zap.NewNop(),
		},
	)

	factor := e.createFactor(t, "FRAP")
	terms := e.agreement(factor.ID, now-3600, nil)
	terms.DueFromFactorAccountID = e.accounts.DueFromFactor
	terms.FeeExpenseAccountID = e.accounts.FeeExpense
	terms.LiabilityAccountID = e.accounts.Liability
	agreement, err := e.agreements.Create(e.ctx, terms)
	require.NoError(t, err)
	first := e.postInvoice(t, 100000, &factor.ID)
	second := e.postInvoice(t, 60000, &factor.ID)
	sub, err := e.submissions.Create(
		e.ctx,
		e.submission(agreement, "FS-RA1", "key-ra1", first, second),
	)
	require.NoError(t, err)
	require.NoError(t, sub.MarkSent(now-86400, "", 1))
	sub, err = e.submissions.Update(e.ctx, sub)
	require.NoError(t, err)

	rem, err := svc.Create(e.ctx, &factoringremittanceports.CreateRemittanceRequest{
		TenantInfo:         e.tenantInfo,
		FactoringCompanyID: factor.ID,
	}, actor)
	require.NoError(t, err)

	csv := "Invoice Number,Advance Amount,Date\n"
	for _, item := range sub.Items {
		csv += item.InvoiceNumber + "," + minorText(item.ExpectedAdvanceMinor) + "," +
			time.Unix(now, 0).UTC().Format("2006-01-02") + "\n"
	}
	docID := pulid.MustNew("doc_")
	files.docs[docID] = &document.Document{
		ID:           docID,
		OriginalName: "frap.csv",
		FileSize:     int64(len(csv)),
		ResourceType: factoringremittance.RemittanceDocumentResourceType,
		ResourceID:   rem.ID.String(),
	}
	files.contents[docID] = []byte(csv)

	staged, err := svc.Stage(e.ctx, &factoringremittanceports.StageRemittanceRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: docID,
	}, actor)
	require.NoError(t, err)
	require.Equal(t, factoringremittance.RemittanceStatusParsed, staged.Status, staged.Error)
	assert.Equal(t, 2, staged.MatchedCount)

	plan, err := svc.PlanApply(e.ctx, &factoringremittanceports.RemittanceVersionRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    staged.Version,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, 2, plan.FundedCount)
	require.Len(t, plan.Lines, 2)
	for _, line := range plan.Lines {
		require.NotNil(t, line.Movement, "the preview is the journal")
		assert.NotEmpty(t, line.Movement.Lines)
	}

	applied, err := svc.Apply(e.ctx, &factoringremittanceports.RemittanceVersionRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    staged.Version,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, factoringremittance.RemittanceStatusApplied, applied.Remittance.Status)
	require.Len(t, applied.Entries, 2)
	assert.Zero(t, e.unbalancedEntries(t))

	loaded, err := remittances.Get(e.ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo:   e.tenantInfo,
		ID:           rem.ID,
		IncludeLines: true,
	})
	require.NoError(t, err)
	for _, line := range loaded.Lines {
		assert.Equal(t, factoringremittance.RemittanceLineApplied, line.MatchStatus)
		assert.NotEmpty(t, line.ReserveEntryID)
		assert.Equal(t, sub.ID, line.MatchedSubmission)
		assert.NotEmpty(t, line.MatchedInvoice)
		entries, listErr := e.entries.ListForItem(
			e.ctx,
			factoringledgerports.ListReserveEntriesForItemRequest{
				TenantInfo:       e.tenantInfo,
				SubmissionItemID: line.SubmissionItemID,
			},
		)
		require.NoError(t, listErr)
		require.Len(t, entries, 1)
		assert.Equal(t, line.ID, entries[0].RemittanceLineID,
			"each entry records the remittance line it came from")
	}

	_, err = svc.Apply(e.ctx, &factoringremittanceports.RemittanceVersionRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    applied.Remittance.Version,
	}, actor)
	require.Error(t, err, "an applied remittance books nothing twice")
}
