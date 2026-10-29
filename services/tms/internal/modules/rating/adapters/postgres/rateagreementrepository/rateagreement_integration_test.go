//go:build integration

package rateagreementrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func insertAgreement(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant ratingdb.Tenant,
	code string,
) *rateagreement.RateAgreement {
	t.Helper()

	billTo := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: tenant.Info.OrgID,
		BusinessUnitID: tenant.Info.BuID,
		StateID:        tenant.Data.State.ID,
		Code:           code,
		Name:           code + " Customer",
		AddressLine1:   "123 Main St",
		City:           "Atlanta",
		PostalCode:     "30303",
	}
	agreement := &rateagreement.RateAgreement{
		ID:             pulid.MustNew("rag_"),
		OrganizationID: tenant.Info.OrgID,
		BusinessUnitID: tenant.Info.BuID,
		PartyType:      rateagreement.PartyTypeCustomer,
		CustomerID:     &billTo.ID,
		Code:           code,
		Name:           code + " Agreement",
		AgreementType:  rateagreement.AgreementTypeContract,
		Status:         rateagreement.StatusActive,
		Currency:       "USD",
		EffectiveFrom:  100,
	}
	insertRow(t, ctx, db, billTo, agreement)

	return agreement
}

func TestGetByIDsReturnsOnlyTheTenantsAgreements(t *testing.T) {
	t.Parallel()

	rdb := ratingdb.NewDatabase(t)
	first := insertAgreement(t, rdb.A.Ctx, rdb.Admin, rdb.A, "BYID1")
	second := insertAgreement(t, rdb.A.Ctx, rdb.Admin, rdb.A, "BYID2")
	foreign := insertAgreement(t, rdb.B.Ctx, rdb.Admin, rdb.B, "BYIDX")

	repo := New(Params{DB: rdb.Conn, Logger: zap.NewNop()})

	agreements, err := repo.GetByIDs(rdb.A.Ctx, &rateagreementports.GetRateAgreementsByIDsRequest{
		TenantInfo:       rdb.A.Info,
		RateAgreementIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("rag_")},
	})
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(agreements))
	for _, agreement := range agreements {
		codes[agreement.ID] = agreement.Code
		assert.Empty(t, agreement.Rules, "a batch read loads no lanes")
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "BYID1", second.ID: "BYID2"}, codes)

	none, err := repo.GetByIDs(
		rdb.A.Ctx,
		&rateagreementports.GetRateAgreementsByIDsRequest{TenantInfo: rdb.A.Info},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
