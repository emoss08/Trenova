package carrierok_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMonitoringGroupsChangeTriples(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, fixture(t, "monitoring_list.json"))
	})

	result, err := client.ListMonitoring(t.Context(), carrierok.MonitoringListParams{})
	require.NoError(t, err)
	require.Len(t, result.Profiles, 2)
	assert.NotEmpty(t, result.Raw)

	authority := result.Profiles[0]
	assert.Equal(t, "568253", authority.DOTNumber)
	assert.Equal(t, "MC277622", authority.Docket)
	require.Len(t, authority.Changes, 1)
	assert.Equal(t, "authority_common", authority.Changes[0].Field)
	assert.JSONEq(t, `"INACTIVE"`, string(authority.Changes[0].Current))
	assert.JSONEq(t, `"ACTIVE"`, string(authority.Changes[0].Prior))
	assert.True(t, authority.Changes[0].Changed)
	assert.Equal(t, map[string]string{"last_changed_date": "20260910"}, authority.Meta)

	sandbox := result.Profiles[1]
	assert.Equal(t, "818175", sandbox.DOTNumber)
	assert.Equal(t, "MC277621", sandbox.Docket)
	require.Len(t, sandbox.Changes, 3)

	assert.Equal(t, "insurance_bipd_on_file", sandbox.Changes[0].Field)
	assert.JSONEq(t, `1000000`, string(sandbox.Changes[0].Current))
	assert.JSONEq(t, `"750000"`, string(sandbox.Changes[0].Prior))
	assert.True(t, sandbox.Changes[0].Changed)

	assert.Equal(t, "safety_rating_desc", sandbox.Changes[1].Field)
	assert.JSONEq(t, `"Satisfactory"`, string(sandbox.Changes[1].Current))
	assert.JSONEq(t, `"Conditional"`, string(sandbox.Changes[1].Prior))
	assert.True(t, sandbox.Changes[1].Changed)

	assert.Equal(t, "email_address", sandbox.Changes[2].Field)
	assert.JSONEq(t, `"ops@sandboxfreight.example"`, string(sandbox.Changes[2].Current))
	assert.Nil(t, sandbox.Changes[2].Prior)
	assert.False(t, sandbox.Changes[2].Changed)

	assert.Equal(t, map[string]string{
		"last_changed_date":     "20260912",
		"started_monitoring_at": "1754006400",
		"last_acknowledged_at":  "2026-09-01T12:00:00Z",
	}, sandbox.Meta)
}

func TestListMonitoringToleratesEmptyAndObjectItems(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			writeJSON(w, http.StatusOK, []byte(`{"total_count":0}`))
			return
		}
		writeJSON(w, http.StatusOK, []byte(
			`{"total_count":"1","items":{"1-MC2":{"usdot_status_current":"ACTIVE","usdot_status_changed":"N"}}}`,
		))
	})

	empty, err := client.ListMonitoring(t.Context(), carrierok.MonitoringListParams{Page: 1})
	require.NoError(t, err)
	assert.Empty(t, empty.Profiles)
	assert.Zero(t, empty.TotalCount)

	single, err := client.ListMonitoring(t.Context(), carrierok.MonitoringListParams{Page: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(1), single.TotalCount)
	require.Len(t, single.Profiles, 1)
	require.Len(t, single.Profiles[0].Changes, 1)
	assert.Equal(t, "usdot_status", single.Profiles[0].Changes[0].Field)
	assert.Nil(t, single.Profiles[0].Changes[0].Prior)
	assert.False(t, single.Profiles[0].Changes[0].Changed)
}

func TestMonitoringListParamsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params carrierok.MonitoringListParams
		valid  bool
	}{
		{name: "empty", params: carrierok.MonitoringListParams{}, valid: true},
		{name: "negative page", params: carrierok.MonitoringListParams{Page: -1}},
		{name: "page size too large", params: carrierok.MonitoringListParams{PageSize: 501}},
		{name: "bad sort order", params: carrierok.MonitoringListParams{SortOrder: "up"}},
		{
			name: "inverted dates",
			params: carrierok.MonitoringListParams{
				DateType: "last_changed_date",
				DateMin:  time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC),
				DateMax:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "dates without type",
			params: carrierok.MonitoringListParams{
				DateMin: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		err := tt.params.Validate()
		if tt.valid {
			assert.NoError(t, err, tt.name)
			continue
		}
		assert.ErrorIs(t, err, carrierok.ErrInvalidQuery, tt.name)
	}
}
