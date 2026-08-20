//go:build integration

package rateagreementrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A version stores its accessorial terms keyed by charge id and nothing else;
// the names people read are resolved when the history is read. That covers the
// charge a version still carries and the charge a renegotiation dropped — the
// removed accessorial's id survives only in the change summary, and its name
// must still resolve.
func TestListVersions_ResolvesAccessorialNamesAtReadTime(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	cust := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		StateID:        data.State.ID,
		Code:           "VERSCUST",
		Name:           "Version Test Customer",
		AddressLine1:   "1 Contract Way",
		City:           "Atlanta",
		PostalCode:     "30303",
	}
	_, err := db.NewInsert().Model(cust).Exec(ctx)
	require.NoError(t, err)

	keptCharge := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           "DET",
		Description:    "Detention",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(60),
	}
	droppedCharge := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           "TONU",
		Description:    "Truck ordered, not used",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(250),
	}
	_, err = db.NewInsert().Model(&[]*accessorialcharge.AccessorialCharge{keptCharge, droppedCharge}).
		Exec(ctx)
	require.NoError(t, err)

	agreement := &rateagreement.RateAgreement{
		ID:             pulid.MustNew("ragr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PartyType:      rateagreement.PartyTypeCustomer,
		CustomerID:     &cust.ID,
		Code:           "VERSION-TEST",
		Name:           "Version Naming Agreement",
		AgreementType:  rateagreement.AgreementTypeContract,
		Status:         rateagreement.StatusDraft,
		Currency:       "USD",
		EffectiveFrom:  100,
	}
	_, err = db.NewInsert().Model(agreement).Exec(ctx)
	require.NoError(t, err)

	version := rateagreement.NewVersionFromAgreement(agreement, 2, 200, data.User.ID, "", nil)
	version.AccessorialTerms = map[string]rateagreement.AccessorialTermSnapshot{
		keptCharge.ID.String(): {
			Method: accessorialcharge.MethodFlat,
			Amount: decimal.NewFromInt(75),
		},
	}
	version.ChangeSummary = map[string]jsonutils.FieldChange{
		"accessorialTerms." + droppedCharge.ID.String(): {
			Type: jsonutils.ChangeTypeDeleted,
			Path: "accessorialTerms." + droppedCharge.ID.String(),
		},
	}

	_, err = repo.CreateVersion(ctx, version)
	require.NoError(t, err)

	listed, err := repo.ListVersions(ctx, &repositories.ListRateAgreementVersionsRequest{
		TenantInfo:      tenantInfo,
		RateAgreementID: agreement.ID,
	})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)

	names := listed.Items[0].AccessorialNames
	assert.Equal(t, "DET", names[keptCharge.ID.String()])
	assert.Equal(t, "TONU", names[droppedCharge.ID.String()])
}
