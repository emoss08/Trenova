//go:build integration

package edicontrolnumberrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartControlNumber_CreatesASequenceAndNeverMovesItBack(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var err error

	repo := edicontrolnumberrepository.New(edicontrolnumberrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	var org seededControlNumberOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var documentType seededDocumentType
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("id").
		Where("transaction_set = ?", edi.TransactionSet214).
		Where("direction = ?", edi.DocumentDirectionOutbound).
		Limit(1).
		Scan(ctx, &documentType))
	partner := &edi.EDIPartner{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Kind:           edi.PartnerKindExternal,
		Code:           "CTRLNUM-START",
		Name:           "Control Number Start Partner",
	}
	_, err = db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)

	start := func(next int64) *edi.EDIControlNumberSequence {
		sequence, startErr := repo.StartControlNumber(ctx, &ediports.ResetEDIControlNumberRequest{
			TenantInfo:     pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
			PartnerID:      partner.ID,
			DocumentTypeID: documentType.ID,
			Kind:           edi.ControlNumberKindInterchange,
			NextValue:      next,
		})
		require.NoError(t, startErr)
		return sequence
	}

	assert.Equal(t, int64(1000), start(1000).NextValue, "a new sequence starts where the partner asks")
	assert.Equal(t, int64(2000), start(2000).NextValue, "a later start moves it forward")
	assert.Equal(t, int64(2000), start(1500).NextValue, "an earlier start never reissues numbers")

	count, err := db.NewSelect().
		Model((*edi.EDIControlNumberSequence)(nil)).
		Where("edi_partner_id = ?", partner.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}
