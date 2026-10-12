package dbhelper_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/stretchr/testify/assert"
)

func TestNowEpoch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "extract(epoch from current_timestamp)::bigint", dbhelper.NowEpoch)
}

func TestMonthStartEpoch(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"extract(epoch from date_trunc('month', to_timestamp(wsev.occurred_at) at time zone 'UTC'))::bigint",
		dbhelper.MonthStartEpoch("wsev.occurred_at"),
	)
}
