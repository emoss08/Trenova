package migrations

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
)

const (
	shipmentStageRankUp   = "20261231008170_shipment_stage_rank.tx.up.sql"
	shipmentStageRankDown = "20261231008170_shipment_stage_rank.tx.down.sql"
)

func TestShipmentStageRankMigration_MatchesTheDomainStageDefinition(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, shipmentStageRankUp))

	assert.Contains(t, up, `"stage_rank" smallint GENERATED ALWAYS AS (`+
		shipment.StageRankSQL(`"status"`)+`) STORED`)
	assert.Contains(t, up, `CREATE INDEX IF NOT EXISTS "idx_shipments_stage_rank"`)
}

func TestShipmentStageRankMigration_DownDropsTheColumn(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, shipmentStageRankDown))

	assert.Contains(t, down, `DROP INDEX IF EXISTS "idx_shipments_stage_rank"`)
	assert.Contains(t, down, `DROP COLUMN IF EXISTS "stage_rank"`)
}
