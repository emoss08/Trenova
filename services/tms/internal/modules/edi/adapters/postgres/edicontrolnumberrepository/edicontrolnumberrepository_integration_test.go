//go:build integration

package edicontrolnumberrepository_test

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seededDocumentType(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	set edi.TransactionSet,
) *edi.EDIDocumentType {
	t.Helper()

	dt := edidb.EDIDocumentTypes
	documentType, err := dt.Where(
		dt.TransactionSet.Eq(set),
		dt.Direction.Eq(edi.DocumentDirectionOutbound),
	).First(ctx, db)
	require.NoError(t, err)

	return documentType
}

func createPartner(t *testing.T, ctx context.Context, db *keel.DB, code string) *edi.EDIPartner {
	t.Helper()

	partner := &edi.EDIPartner{
		Kind: edi.PartnerKindExternal,
		Code: code,
		Name: "Control Number Partner " + code,
	}
	require.NoError(t, edidb.EDIPartners.Create(ctx, db, partner))

	return partner
}

func TestAllocateControlNumbersConcurrentAllocationsAreUnique(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	repo := edicontrolnumberrepository.New(edicontrolnumberrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	scope := editest.FirstTenant(t, ctx, db)
	ctx, tenant := scope.Ctx, scope.Info
	documentType := seededDocumentType(t, ctx, db, edi.TransactionSet204)
	partner := createPartner(t, ctx, db, "CTRLNUM-TEST")

	req := edipartnerports.AllocateEDIControlNumbersRequest{
		TenantInfo:     tenant,
		PartnerID:      partner.ID,
		DocumentTypeID: documentType.ID,
		Kinds: []edi.ControlNumberKind{
			edi.ControlNumberKindInterchange,
			edi.ControlNumberKindGroup,
			edi.ControlNumberKindTransaction,
		},
	}

	const workers = 16
	results := make([]map[edi.ControlNumberKind]int64, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = repo.AllocateControlNumbers(ctx, req)
		}(i)
	}
	wg.Wait()

	seenByKind := map[edi.ControlNumberKind]map[int64]bool{
		edi.ControlNumberKindInterchange: {},
		edi.ControlNumberKindGroup:       {},
		edi.ControlNumberKindTransaction: {},
	}
	for i := range workers {
		require.NoError(t, errs[i])
		require.Len(t, results[i], len(req.Kinds))
		for kind, value := range results[i] {
			require.False(
				t,
				seenByKind[kind][value],
				"duplicate control number %d allocated for kind %s",
				value,
				kind,
			)
			seenByKind[kind][value] = true
		}
	}
	for kind, seen := range seenByKind {
		require.Len(t, seen, workers, "expected %d unique %s control numbers", workers, kind)
	}
}
