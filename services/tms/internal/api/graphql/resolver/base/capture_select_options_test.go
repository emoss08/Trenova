package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func selectRequest(limit, offset int, ids ...pulid.ID) *SelectOptionsRequest {
	return &SelectOptionsRequest{
		IDs: ids,
		SelectQuery: &pagination.SelectQueryRequest{
			Pagination: pagination.Info{Limit: limit, Offset: offset},
		},
	}
}

func deviceItems(t *testing.T, count int) []selectOptionConnectionItem {
	t.Helper()
	items := make([]selectOptionConnectionItem, 0, count)
	for i := range count {
		device := &capture.CaptureDevice{
			ID:        pulid.MustNew("cdev_"),
			Name:      "Computer",
			Status:    capture.DeviceActive,
			CreatedAt: int64(i + 1),
		}
		items = append(items, captureDeviceSelectOptionItem(device, 0))
	}
	return items
}

func TestPageSelectOptionItemsPagesAndReportsTheTotal(t *testing.T) {
	items := deviceItems(t, 5)

	first, err := pageSelectOptionItems(items, selectRequest(2, 0))
	require.NoError(t, err)
	require.Len(t, first.Edges, 2)
	assert.Equal(t, 5, *first.TotalCount)
	assert.True(t, first.PageInfo.HasNextPage)

	last, err := pageSelectOptionItems(items, selectRequest(2, 4))
	require.NoError(t, err)
	require.Len(t, last.Edges, 1)
	assert.Equal(t, items[4].option.ID, last.Edges[0].Node.ID)
	assert.False(t, last.PageInfo.HasNextPage)

	past, err := pageSelectOptionItems(items, selectRequest(2, 50))
	require.NoError(t, err)
	assert.Empty(t, past.Edges)
	assert.Equal(t, 5, *past.TotalCount)
}

func TestWithIDsKeepsOnlyTheAskedForEntitiesAndNothingElse(t *testing.T) {
	a := &capture.CaptureProfile{ID: pulid.MustNew("cprf_"), Name: "A"}
	b := &capture.CaptureProfile{ID: pulid.MustNew("cprf_"), Name: "B"}
	c := &capture.CaptureProfile{ID: pulid.MustNew("cprf_"), Name: "C"}
	idOf := func(p *capture.CaptureProfile) pulid.ID { return p.ID }

	all := []*capture.CaptureProfile{a, b, c}
	assert.Equal(t, all, withIDs(all, nil, idOf))
	assert.Equal(
		t,
		[]*capture.CaptureProfile{a, c},
		withIDs(all, []pulid.ID{c.ID, a.ID}, idOf),
	)
	assert.Empty(t, withIDs(all, []pulid.ID{pulid.MustNew("cprf_")}, idOf))
}

func TestCaptureDeviceOptionCarriesItsScannersAndWhetherItIsOnline(t *testing.T) {
	seen := int64(1_000)
	device := &capture.CaptureDevice{
		ID:          pulid.MustNew("cdev_"),
		Name:        "Front desk",
		MachineName: "DISPATCH-07",
		Status:      capture.DeviceActive,
		LastSeenAt:  &seen,
		Sources: []capture.SourceInfo{
			{Name: "fi-8170", Protocol: capture.SourceProtocol("Twain"), IsDefault: true},
			{Name: "DS-530", Protocol: capture.SourceProtocol("Wia")},
		},
	}

	online := captureDeviceSelectOptionItem(device, seen+1).option
	assert.Equal(t, "Front desk", online.Label)
	assert.Equal(t, "DISPATCH-07", *online.Description)
	assert.Equal(t, true, online.Meta["isOnline"])
	sources, ok := online.Meta["sources"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, sources, 2)
	assert.Equal(t, "fi-8170", sources[0]["name"])
	assert.Equal(t, true, sources[0]["isDefault"])

	stale := captureDeviceSelectOptionItem(device, seen+capture.OnlineWindowSeconds+1).option
	assert.Equal(t, false, stale.Meta["isOnline"])
}

func TestCaptureProfileOptionNamesItsSettings(t *testing.T) {
	profile := &capture.CaptureProfile{
		ID:          pulid.MustNew("cprf_"),
		Name:        "Paperwork",
		Description: "Bills of lading",
		IsDefault:   true,
		DPI:         300,
		PixelType:   capture.PixelType("BlackWhite"),
		Duplex:      true,
	}

	option := captureProfileSelectOptionItem(profile).option
	assert.Equal(t, "Paperwork", option.Label)
	assert.Equal(t, "Bills of lading", *option.Description)
	assert.Equal(t, true, option.Meta["isDefault"])
	assert.Equal(t, 300, option.Meta["dpi"])
	assert.Equal(t, true, option.Meta["duplex"])
}
