//go:build integration

package edicontrolnumberrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartControlNumber_CreatesASequenceAndNeverMovesItBack(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	repo := edicontrolnumberrepository.New(edicontrolnumberrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	scope := editest.FirstTenant(t, ctx, db)
	ctx, tenant := scope.Ctx, scope.Info
	documentType := seededDocumentType(t, ctx, db, edi.TransactionSet214)
	partner := createPartner(t, ctx, db, "CTRLNUM-START")

	start := func(next int64) *edi.EDIControlNumberSequence {
		sequence, startErr := repo.StartControlNumber(ctx, &ediports.ResetEDIControlNumberRequest{
			TenantInfo:     tenant,
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

	cn := edidb.EDIControlNumberSequences
	count, err := cn.Where(cn.EDIPartnerID.Eq(partner.ID)).Count(ctx, db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}
