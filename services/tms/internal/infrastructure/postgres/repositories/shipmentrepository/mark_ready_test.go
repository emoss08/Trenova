package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readyToInvoiceEntity() *shipment.Shipment {
	markedAt := int64(1_790_000_000)
	return &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		Status:              shipment.StatusReadyToInvoice,
		MarkedReadyToBillAt: &markedAt,
		Version:             7,
		AdditionalCharges:   []*shipment.AdditionalCharge{{ID: pulid.MustNew("ac_")}},
	}
}

func TestMarkReadyToInvoice_WritesOnlyTheStatusColumnsUnderTheVersion(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	entity := readyToInvoiceEntity()

	mock.ExpectQuery(
		`^UPDATE "shipments" AS "sp" SET "status" = 'ReadyToInvoice', "marked_ready_to_bill_at" = 1790000000, "version" = 8, "updated_at" = \d+ ` +
			`WHERE .*sp\.id = '` + entity.ID.String() + `'.*sp\.version = 7.*RETURNING \*$`,
	).WillReturnRows(sqlmock.NewRows([]string{"id", "status", "version"}).
		AddRow(entity.ID, shipment.StatusReadyToInvoice, 8))

	updated, err := repo.MarkReadyToInvoice(t.Context(), entity)

	require.NoError(t, err)
	assert.Same(t, entity, updated)
	assert.Equal(t, int64(8), updated.Version)
	assert.Len(t, updated.AdditionalCharges, 1, "loaded details are kept, never rewritten")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkReadyToInvoice_AStaleVersionIsAConflictAndKeepsTheVersion(t *testing.T) {
	t.Parallel()

	repo, mock := newCancelTestRepository(t)
	entity := readyToInvoiceEntity()

	mock.ExpectQuery(`^UPDATE "shipments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "version"}))

	_, err := repo.MarkReadyToInvoice(t.Context(), entity)

	require.Error(t, err)
	assert.Equal(t, int64(7), entity.Version)
	require.NoError(t, mock.ExpectationsWereMet())
}
