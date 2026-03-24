package workerptoservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateChartRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1735689600,
			StartDateTo:   1736294400,
			Type:          "all",
		})
		require.NoError(t, err)
	})

	t.Run("rejects invalid range and invalid type", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1736294400,
			StartDateTo:   1735689600,
			Type:          "NotAType",
		})
		require.Error(t, err)

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)

		assert.True(t, multiErr.HasErrors())
		assert.GreaterOrEqual(t, len(multiErr.Errors), 2)
	})

	t.Run("rejects range over 366 days", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1704067200,
			StartDateTo:   1735776002,
			Type:          "Vacation",
		})
		require.Error(t, err)
	})

	t.Run("rejects invalid worker ID", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1735689600,
			StartDateTo:   1736294400,
			WorkerID:      "bad-id",
		})
		require.Error(t, err)
	})
}
