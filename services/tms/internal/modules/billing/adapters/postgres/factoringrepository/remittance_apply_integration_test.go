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
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
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

type remittanceApplyEnv struct {
	*ledgerEnv
	svc         factoringports.RemittanceService
	remittances factoringports.RemittanceRepository
	files       *remittanceFiles
	actor       *permissionports.RequestActor
	factor      *factoringcompany.FactoringCompany
	sub         *factoring.Submission
	now         int64
}

func setupRemittanceApplyEnv(t *testing.T, code string) *remittanceApplyEnv {
	t.Helper()
	e := setupLedgerEnv(t)
	actor := testutil.NewSessionActor(e.userID, e.tenantInfo.OrgID, e.tenantInfo.BuID)
	now := time.Now().Unix()
	conn := postgres.NewTestConnection(e.db)
	remittances := NewRemittanceRepository(Params{DB: conn, Logger: zap.NewNop()})
	files := &remittanceFiles{
		docs:     map[pulid.ID]*document.Document{},
		contents: map[pulid.ID][]byte{},
	}
	svc := factoringservice.NewRemittanceService(factoringservice.RemittanceParams{
		DB:          conn,
		Remittances: remittances,
		Submissions: e.submissions,
		Ledger:      e.svc,
		Factors:     e.factors,
		Documents:   files,
		Audit:       &mocks.NoopAuditService{},
		Logger:      zap.NewNop(),
	})

	factor := e.createFactor(t, code)
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
		e.submission(agreement, "FS-"+code, "key-"+code, first, second),
	)
	require.NoError(t, err)
	require.NoError(t, sub.MarkSent(now-86400, "", 1))
	sub, err = e.submissions.Update(e.ctx, sub)
	require.NoError(t, err)

	return &remittanceApplyEnv{
		ledgerEnv:   e,
		svc:         svc,
		remittances: remittances,
		files:       files,
		actor:       actor,
		factor:      factor,
		sub:         sub,
		now:         now,
	}
}

func (r *remittanceApplyEnv) stage(
	t *testing.T,
	kind factoring.RemittanceLineKind,
	csv string,
) *factoring.Remittance {
	t.Helper()
	rem, err := r.svc.Create(r.ctx, &factoringports.CreateRemittanceRequest{
		TenantInfo:         r.tenantInfo,
		FactoringCompanyID: r.factor.ID,
		DefaultKind:        kind,
	}, r.actor)
	require.NoError(t, err)

	docID := pulid.MustNew("doc_")
	r.files.docs[docID] = &document.Document{
		ID:           docID,
		OriginalName: "remittance.csv",
		FileSize:     int64(len(csv)),
		ResourceType: factoring.RemittanceDocumentResourceType,
		ResourceID:   rem.ID.String(),
	}
	r.files.contents[docID] = []byte(csv)

	staged, err := r.svc.Stage(r.ctx, &factoringports.StageRemittanceRequest{
		TenantInfo: r.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: docID,
	}, r.actor)
	require.NoError(t, err)
	require.Equal(t, factoring.RemittanceStatusParsed, staged.Status, staged.Error)
	return staged
}

func (r *remittanceApplyEnv) day() string {
	return time.Unix(r.now, 0).UTC().Format("2006-01-02")
}

func TestRemittanceAppliesThroughTheLedgerInOneTransaction(t *testing.T) {
	r := setupRemittanceApplyEnv(t, "FRAP")
	e, svc, actor, sub, remittances := r.ledgerEnv, r.svc, r.actor, r.sub, r.remittances

	csv := "Invoice Number,Advance Amount,Date\n"
	for _, item := range sub.Items {
		csv += item.InvoiceNumber + "," + minorText(item.ExpectedAdvanceMinor) + "," + r.day() + "\n"
	}
	staged := r.stage(t, factoring.RemittanceLineFunding, csv)
	rem := staged
	assert.Equal(t, 2, staged.MatchedCount)

	plan, err := svc.PlanApply(e.ctx, &factoringports.RemittanceVersionRequest{
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

	applied, err := svc.Apply(e.ctx, &factoringports.RemittanceVersionRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    staged.Version,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusApplied, applied.Remittance.Status)
	require.Len(t, applied.Entries, 2)
	assert.Zero(t, e.unbalancedEntries(t))

	loaded, err := remittances.Get(e.ctx, factoringports.GetRemittanceRequest{
		TenantInfo:   e.tenantInfo,
		ID:           rem.ID,
		IncludeLines: true,
	})
	require.NoError(t, err)
	for _, line := range loaded.Lines {
		assert.Equal(t, factoring.RemittanceLineApplied, line.MatchStatus)
		assert.NotEmpty(t, line.ReserveEntryID)
		assert.Equal(t, sub.ID, line.MatchedSubmission)
		assert.NotEmpty(t, line.MatchedInvoice)
		entries, listErr := e.entries.ListForItem(e.ctx, factoringports.ListReserveEntriesForItemRequest{
			TenantInfo:       e.tenantInfo,
			SubmissionItemID: line.SubmissionItemID,
		})
		require.NoError(t, listErr)
		require.Len(t, entries, 1)
		assert.Equal(t, line.ID, entries[0].RemittanceLineID,
			"each entry records the remittance line it came from")
	}

	_, err = svc.Apply(e.ctx, &factoringports.RemittanceVersionRequest{
		TenantInfo: e.tenantInfo,
		ID:         rem.ID,
		Version:    applied.Remittance.Version,
	}, actor)
	require.Error(t, err, "an applied remittance books nothing twice")
}

func TestRemittanceAppliesAReleaseOfAnEmptyReserve(t *testing.T) {
	r := setupRemittanceApplyEnv(t, "FRZR")
	item := r.sub.Items[0]
	advance := item.FaceMinor - item.ExpectedFeeMinor
	fee := item.ExpectedFeeMinor
	funded, err := r.ledgerEnv.svc.RecordFunding(r.ctx, &factoringports.RecordFundingRequest{
		TenantInfo:   r.tenantInfo,
		SubmissionID: r.sub.ID,
		Version:      r.sub.Version,
		FundedOn:     r.now,
		Items: []factoringports.FundingItemInput{{
			ItemID:       item.ID,
			Outcome:      factoring.FundingOutcomeFunded,
			AdvanceMinor: &advance,
			FeeMinor:     &fee,
		}},
	}, r.actor)
	require.NoError(t, err)
	require.Len(t, funded.Entries, 1)

	staged := r.stage(t, factoring.RemittanceLineReserveRelease,
		"Invoice Number,Cash,Date\n"+item.InvoiceNumber+",0.00,"+r.day()+"\n")
	require.Equal(t, 1, staged.MatchedCount, "a zero release matches the zero reserve held")

	applied, err := r.svc.Apply(r.ctx, &factoringports.RemittanceVersionRequest{
		TenantInfo: r.tenantInfo,
		ID:         staged.ID,
		Version:    staged.Version,
	}, r.actor)
	require.NoError(t, err, "a release that books nothing still settles the line")
	assert.Equal(t, factoring.RemittanceStatusApplied, applied.Remittance.Status)
	assert.Empty(t, applied.Entries, "an empty reserve books no journal")
	assert.Zero(t, r.unbalancedEntries(t))
}
