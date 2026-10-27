//go:build integration

package rateagreementrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
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

	rdb := ratingdb.NewDatabase(t)
	ctx, db, data := rdb.A.Ctx, rdb.Admin, rdb.A.Data
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	repo := New(Params{DB: rdb.Conn, Logger: zap.NewNop()})

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
	insertRow(t, ctx, db, cust)

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
	insertRow(t, ctx, db, keptCharge, droppedCharge)

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
	insertRow(t, ctx, db, agreement)

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

	_, err := repo.CreateVersion(ctx, version)
	require.NoError(t, err)

	listed, err := repo.ListVersions(ctx, &rateagreementports.ListRateAgreementVersionsRequest{
		TenantInfo:      tenantInfo,
		RateAgreementID: agreement.ID,
	})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)

	names := listed.Items[0].AccessorialNames
	assert.Equal(t, "DET", names[keptCharge.ID.String()])
	assert.Equal(t, "TONU", names[droppedCharge.ID.String()])
}
