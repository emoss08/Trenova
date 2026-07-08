//go:build integration

package edicontrolnumberrepository_test

import (
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededControlNumberOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type seededDocumentType struct {
	ID pulid.ID `bun:"id"`
}

func TestAllocateControlNumbersConcurrentAllocationsAreUnique(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	repo := edicontrolnumberrepository.New(edicontrolnumberrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	var org seededControlNumberOrg
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)
	var documentType seededDocumentType
	require.NoError(
		t,
		db.NewSelect().
			Table("edi_document_types").
			Column("id").
			Where("transaction_set = ?", edi.TransactionSet204).
			Where("direction = ?", edi.DocumentDirectionOutbound).
			Limit(1).
			Scan(ctx, &documentType),
	)

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Kind:           edi.PartnerKindExternal,
		Code:           "CTRLNUM-TEST",
		Name:           "Control Number Concurrency Partner",
	}
	_, err = db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)

	req := repositories.AllocateEDIControlNumbersRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: org.ID,
			BuID:  org.BusinessUnitID,
		},
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
