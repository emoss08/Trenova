package rateimportservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type planImportRepo struct {
	repositories.RateImportRepository
	batch   *rateimport.RateImportBatch
	updates int
}

func (r *planImportRepo) GetByID(
	context.Context,
	*repositories.GetRateImportBatchByIDRequest,
) (*rateimport.RateImportBatch, error) {
	copied := *r.batch

	return &copied, nil
}

func (r *planImportRepo) Update(
	_ context.Context,
	batch *rateimport.RateImportBatch,
) (*rateimport.RateImportBatch, error) {
	r.updates++

	return batch, nil
}

func importFixture(t *testing.T, status rateimport.Status) (*Service, *planImportRepo, *rateimport.RateImportBatch) {
	t.Helper()

	agreementID := pulid.MustNew("rag_")
	existing := &rateagreement.RateAgreementRule{
		ID:      pulid.MustNew("rarl_"),
		LaneKey: "ST:GA>ST:FL",
		Rate:    decimal.NewNullDecimal(decimal.RequireFromString("2.00")),
	}
	batch := &rateimport.RateImportBatch{
		ID:              pulid.MustNew("rib_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		RateAgreementID: agreementID,
		Status:          status,
		EffectiveFrom:   1_000,
		Rows: []*rateimport.RateImportRow{{
			RowNumber: 2,
			LaneKey:   "ST:GA>ST:FL",
			Rule: &rateagreement.RateAgreementRule{
				LaneKey: "ST:GA>ST:FL",
				Rate:    decimal.NewNullDecimal(decimal.RequireFromString("2.20")),
			},
		}},
		Changes: []pkgrateimport.Change{{
			Kind:       pkgrateimport.ChangeKindChanged,
			LaneKey:    "ST:GA>ST:FL",
			ExistingID: existing.ID.String(),
		}},
	}

	agreements := mocks.NewMockRateAgreementRepository(t)
	agreements.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&rateagreement.RateAgreement{
			ID:    agreementID,
			Rules: []*rateagreement.RateAgreementRule{existing},
		}, nil).
		Maybe()

	repo := &planImportRepo{batch: batch}

	return &Service{
		l:             zap.NewNop(),
		repo:          repo,
		agreementRepo: agreements,
		now:           func() int64 { return 2_000 },
	}, repo, batch
}

func TestPlanCommit_NamesTheAmendmentAndWritesNothing(t *testing.T) {
	t.Parallel()

	svc, repo, batch := importFixture(t, rateimport.StatusParsed)
	plan, err := svc.PlanCommit(t.Context(), &CommitRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID},
		RateImportBatchID: batch.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, batch.ID, plan.Batch.ID)
	assert.Len(t, plan.SupersededIDs, 1)
	require.Len(t, plan.Rules, 1)
	assert.Equal(t, batch.RateAgreementID, plan.Rules[0].RateAgreementID)
	assert.Zero(t, repo.updates)
}

func TestPlanCommitAndDiscard_RefuseAnImportAlreadyClosed(t *testing.T) {
	t.Parallel()

	svc, repo, batch := importFixture(t, rateimport.StatusCommitted)
	req := &CommitRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID},
		RateImportBatchID: batch.ID,
	}
	_, err := svc.PlanCommit(t.Context(), req)
	require.Error(t, err)
	_, err = svc.PlanDiscard(t.Context(), req)
	require.Error(t, err)
	assert.Zero(t, repo.updates)
}

func TestPlanDiscard_ShowsTheDiscardWithoutSavingIt(t *testing.T) {
	t.Parallel()

	svc, repo, batch := importFixture(t, rateimport.StatusParsed)
	change, err := svc.PlanDiscard(t.Context(), &CommitRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID},
		RateImportBatchID: batch.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, rateimport.StatusParsed, change.Before.Status)
	assert.Equal(t, rateimport.StatusDiscarded, change.After.Status)
	assert.Zero(t, repo.updates)
}
