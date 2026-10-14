//go:build integration

package editemplaterepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editemplaterepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededTenant struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestEnsureBaseTemplate_EveryTransactionSetCanBeSentOut(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	_, err := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org seededTenant
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))
	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	repo := editemplaterepository.New(editemplaterepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	for _, set := range []edi.TransactionSet{
		edi.TransactionSet204,
		edi.TransactionSet210,
		edi.TransactionSet214,
		edi.TransactionSet990,
		edi.TransactionSet997,
		edi.TransactionSet999,
	} {
		t.Run(string(set), func(t *testing.T) {
			template, version, err := repo.EnsureBaseTemplate(ctx, tenant, set)
			require.NoError(t, err)
			require.Equal(t, set, template.TransactionSet)
			require.Equal(t, edi.DocumentDirectionOutbound, template.Direction)
			require.True(t, version.IsActive)

			again, _, err := repo.EnsureBaseTemplate(ctx, tenant, set)
			require.NoError(t, err)
			require.Equal(t, template.ID, again.ID, "a second call finds the first template")
		})
	}
}

func TestDocumentTypeCatalog_HoldsEverySetInBothDirections(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	var rows []struct {
		TransactionSet edi.TransactionSet    `bun:"transaction_set"`
		Direction      edi.DocumentDirection `bun:"direction"`
	}
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("transaction_set", "direction").
		Where("standard = ?", edi.EDIStandardX12).
		Scan(ctx, &rows))

	present := make(map[string]bool, len(rows))
	for _, row := range rows {
		present[string(row.TransactionSet)+"/"+string(row.Direction)] = true
	}
	for _, set := range []edi.TransactionSet{
		edi.TransactionSet204, edi.TransactionSet210, edi.TransactionSet214,
		edi.TransactionSet990, edi.TransactionSet997, edi.TransactionSet999,
	} {
		for _, direction := range []edi.DocumentDirection{
			edi.DocumentDirectionInbound, edi.DocumentDirectionOutbound,
		} {
			require.Truef(t, present[string(set)+"/"+string(direction)],
				"X12 %s %s is not in the document type catalog", set, direction)
		}
	}
}
