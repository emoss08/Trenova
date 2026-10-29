package documentsselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// maxCaptureDevices matches the bound captureresolver puts on a device list.
const maxCaptureDevices = 500

const (
	captureMetaName      = "name"
	captureMetaIsDefault = "isDefault"
)

func captureDeviceSelectOptionItem(
	device *capture.CaptureDevice,
	now int64,
) selectoption.Item {
	sources := make([]map[string]any, 0, len(device.Sources))
	for i := range device.Sources {
		source := &device.Sources[i]
		sources = append(sources, map[string]any{
			captureMetaName:      source.Name,
			"protocol":           source.Protocol,
			captureMetaIsDefault: source.IsDefault,
		})
	}

	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          device.ID.String(),
			Label:       device.Name,
			Description: stringutils.Ptr(device.MachineName),
			Meta: map[string]any{
				"isOnline":   device.IsOnline(now),
				"lastSeenAt": device.LastSeenAt,
				"sources":    sources,
			},
		},
		device.CreatedAt,
		device.ID,
	)
}

func captureProfileSelectOptionItem(
	profile *captureprofile.CaptureProfile,
) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          profile.ID.String(),
			Label:       profile.Name,
			Description: stringutils.Ptr(profile.Description),
			Meta: map[string]any{
				captureMetaIsDefault: profile.IsDefault,
				"dpi":                profile.DPI,
				"pixelType":          profile.PixelType,
				"duplex":             profile.Duplex,
			},
		},
		profile.CreatedAt,
		profile.ID,
	)
}

func withIDs[T any](entities []T, ids []pulid.ID, idOf func(T) pulid.ID) []T {
	if len(ids) == 0 {
		return entities
	}

	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	kept := make([]T, 0, len(ids))
	for _, entity := range entities {
		if _, ok := wanted[idOf(entity)]; ok {
			kept = append(kept, entity)
		}
	}

	return kept
}

func pageSelectOptionItems(
	items []selectoption.Item,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		return selectoption.Connection(items, len(items), 0)
	}

	total := len(items)
	offset := min(req.SelectQuery.Pagination.SafeOffset(), total)
	end := min(offset+req.SelectQuery.Pagination.SafeLimit(), total)

	return selectoption.Connection(items[offset:end], total, offset)
}
