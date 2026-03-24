package assets

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListParamsValidateLimit(t *testing.T) {
	t.Parallel()

	err := ListParams{Limit: 513}.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)
}

func TestListParamsQuery(t *testing.T) {
	t.Parallel()

	updatedAfter := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	params := ListParams{
		Type:               TypeTrailer,
		After:              "cursor",
		UpdatedAfterTime:   &updatedAfter,
		IncludeExternalIDs: true,
		IncludeTags:        true,
		TagIDs:             []string{"t1", "t2"},
		ParentTagIDs:       []string{"p1", "p2"},
		IDs:                []string{"a1", "a2"},
		AttributeValueIDs:  []string{"av1", "av2"},
		Attributes:         []string{"Length:range(8,10)", "Date:range(2025-01-01,2025-01-31)"},
		Limit:              100,
	}

	query := params.Query()
	assert.Equal(t, "trailer", query.Get("type"))
	assert.Equal(t, "cursor", query.Get("after"))
	assert.Equal(t, updatedAfter.Format(time.RFC3339), query.Get("updatedAfterTime"))
	assert.Equal(t, "true", query.Get("includeExternalIds"))
	assert.Equal(t, "true", query.Get("includeTags"))
	assert.Equal(t, "t1,t2", query.Get("tagIds"))
	assert.Equal(t, "p1,p2", query.Get("parentTagIds"))
	assert.Equal(t, "a1,a2", query.Get("ids"))
	assert.Equal(t, "av1,av2", query.Get("attributeValueIds"))
	assert.Equal(t, "100", query.Get("limit"))
	assert.Equal(
		t,
		[]string{"Length:range(8,10)", "Date:range(2025-01-01,2025-01-31)"},
		query["attributes"],
	)
}

func TestLocationStreamParamsValidate(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)
	err := LocationStreamParams{
		StartTime: &start,
		EndTime:   &end,
	}.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLocationWindowInvalid)

	err = LocationStreamParams{
		StartTime:                     &start,
		IncludeGeofenceLookup:         true,
		IncludeHighFrequencyLocations: true,
	}.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrHighFrequencyWithGeofence)
}

func TestLocationStreamParamsQuery(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	query := LocationStreamParams{
		After:                         "next",
		Limit:                         200,
		StartTime:                     &start,
		EndTime:                       &end,
		IDs:                           []string{"a1", "a2"},
		IncludeSpeed:                  true,
		IncludeReverseGeo:             true,
		IncludeGeofenceLookup:         true,
		IncludeExternalIDs:            true,
		IncludeHighFrequencyLocations: false,
	}.Query()

	assert.Equal(t, "next", query.Get("after"))
	assert.Equal(t, "200", query.Get("limit"))
	assert.Equal(t, start.Format(time.RFC3339), query.Get("startTime"))
	assert.Equal(t, end.Format(time.RFC3339), query.Get("endTime"))
	assert.Equal(t, "a1,a2", query.Get("ids"))
	assert.Equal(t, "true", query.Get("includeSpeed"))
	assert.Equal(t, "true", query.Get("includeReverseGeo"))
	assert.Equal(t, "true", query.Get("includeGeofenceLookup"))
	assert.Equal(t, "true", query.Get("includeExternalIds"))
	assert.Equal(t, "", query.Get("includeHighFrequencyLocations"))
}
