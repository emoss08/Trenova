package auditrollup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrollup"
	"github.com/stretchr/testify/assert"
)

func TestTheFigureRollupsHoldEveryColumnTheOverviewReads(t *testing.T) {
	t.Parallel()

	for name, table := range map[string]*rollupread.Table{
		auditrollup.HourlyName: auditrollup.HourlyTable,
		auditrollup.DailyName:  auditrollup.DailyTable,
	} {
		for _, column := range append(
			[]string{rollupread.Bucket, auditrollup.Resource, auditrollup.Category},
			auditrollup.FigureColumns...,
		) {
			assert.True(t, table.Has(column), "%s has no column %s", name, column)
		}
	}
}

func TestTheActorRollupHoldsWhoActedAndWhatTheyMayRead(t *testing.T) {
	t.Parallel()

	for _, column := range []string{
		rollupread.Bucket,
		auditrollup.PrincipalID,
		auditrollup.PrincipalType,
		auditrollup.Resource,
		auditrollup.Category,
		auditrollup.Entries,
		auditrollup.Sensitive,
	} {
		assert.True(
			t,
			auditrollup.ActorDailyTable.Has(column),
			"audit_actor_daily has no column %s",
			column,
		)
	}
}
