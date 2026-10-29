package migrations

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
)

const (
	shipmentStageRankUp = "20261231008170_shipment_stage_rank"
)

func TestShipmentStageRankMigration_MatchesTheDomainStageDefinition(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, shipmentStageRankUp))

	assert.Contains(t, up, `"stage_rank" smallint GENERATED ALWAYS AS (`+
		shipmentvocab.StageRankSQL(`"status"`)+`) STORED`)
	assert.Contains(t, up, `CREATE INDEX IF NOT EXISTS "idx_shipments_stage_rank"`)
}
