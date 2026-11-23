package gqlconnection

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestRecordListLimit_BoundsAnOrganizationWideList(t *testing.T) {
	t.Parallel()

	ten, tooMany := 10, 5000
	assert.Equal(
		t,
		pagination.MaxLimit,
		RecordListLimit(nil, true),
		"a named record's list reads at most one full page",
	)
	assert.Equal(t, pagination.DefaultLimit, RecordListLimit(nil, false))
	assert.Equal(t, ten, RecordListLimit(&ten, false))
	assert.Equal(t, pagination.MaxLimit, RecordListLimit(&tooMany, true))
}
