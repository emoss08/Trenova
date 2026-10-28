//go:build integration

package editenderrecipientrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrecipientrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetTenderRecipientByID_UnderRowLevelSecurityOnlyTheTwoPartiesSeeIt(t *testing.T) {
	tender := editest.SeedInternalTender(t)
	recipient := tender.NewRecipient(t, tender.NewTransfer(t, edi.TransferStatusPendingApproval))

	bystanderOrg := seedtest.NewOrganization(tender.Source.Info.BuID, tender.StateID).
		WithName("Bystander Carrier").
		WithScacCode("BYST").
		WithDOTNumber("9990043").
		WithBucketName("test-bucket-byst").
		Build(t, tender.Ctx, tender.Admin)
	bystander := editest.TenantOf(tender.Ctx, pagination.TenantInfo{
		OrgID: bystanderOrg.ID,
		BuID:  tender.Source.Info.BuID,
	})
	elsewhere := seedtest.SeedAdditionalTenant(t, tender.Ctx, tender.Admin, "XB")
	otherUnit := editest.TenantOf(tender.Ctx, elsewhere.Info())

	repo := editenderrecipientrepository.New(editenderrecipientrepository.Params{
		DB:     dbtest.NewRLSConnection(t, tender.Admin, tender.DSN),
		Logger: zap.NewNop(),
	})
	read := func(scope editest.Tenant, claimed pagination.TenantInfo) (*edi.TenderRecipient, error) {
		return repo.GetTenderRecipientByID(
			scope.Ctx,
			editenderports.GetEDITenderRecipientByIDRequest{
				ID:         recipient.ID,
				TenantInfo: claimed,
			},
		)
	}

	for name, party := range map[string]editest.Tenant{
		"sender":   tender.Source,
		"receiver": tender.Target,
	} {
		got, err := read(party, party.Info)
		require.NoError(t, err, name)
		assert.Equal(t, recipient.ID, got.ID, name)
	}

	for name, probe := range map[string]struct {
		scope   editest.Tenant
		claimed pagination.TenantInfo
	}{
		"an organization of the business unit that is neither party": {
			scope:   bystander,
			claimed: bystander.Info,
		},
		"an organization of another business unit": {
			scope:   otherUnit,
			claimed: otherUnit.Info,
		},
		"another business unit claiming the receiver's organization": {
			scope: otherUnit,
			claimed: pagination.TenantInfo{
				OrgID: tender.Target.Info.OrgID,
				BuID:  otherUnit.Info.BuID,
			},
		},
	} {
		_, err := read(probe.scope, probe.claimed)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsNotFoundError(err), "%s: %v", name, err)
	}
}

func TestTenderRecipients_LiveInTheirSourceShipmentsBusinessUnit(t *testing.T) {
	tender := editest.SeedInternalTender(t)
	transfer := tender.NewTransfer(t, edi.TransferStatusPendingApproval)
	elsewhere := seedtest.SeedAdditionalTenant(t, tender.Ctx, tender.Admin, "XB")
	systemCtx := dbscope.WithSystem(tender.Ctx, "test a tender recipient's business unit")

	misplaced := tender.Recipient(transfer)
	misplaced.BusinessUnitID = elsewhere.BusinessUnit.ID
	err := edidb.TenderRecipients.Create(systemCtx, tender.Admin, misplaced)
	require.Error(t, err)
	assert.Equal(
		t,
		"fk_edi_tender_recipients_source_shipment",
		dberror.ExtractConstraintName(err),
		"a recipient is stored in its source shipment's business unit",
	)

	foreign := tender.Recipient(transfer)
	foreign.RecipientOrganizationID = elsewhere.Organization.ID
	require.NoError(t, edidb.TenderRecipients.Create(tender.Source.Ctx, tender.Admin, foreign))

	repo := editenderrecipientrepository.New(editenderrecipientrepository.Params{
		DB:     dbtest.NewRLSConnection(t, tender.Admin, tender.DSN),
		Logger: zap.NewNop(),
	})
	other := editest.TenantOf(tender.Ctx, elsewhere.Info())
	_, err = repo.GetTenderRecipientByID(other.Ctx, editenderports.GetEDITenderRecipientByIDRequest{
		ID:         foreign.ID,
		TenantInfo: other.Info,
	})
	require.Error(t, err)
	assert.True(
		t,
		errortypes.IsNotFoundError(err),
		"an organization of another business unit never reads a recipient row, even one naming it: %v",
		err,
	)
}
