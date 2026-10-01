package selectoptionsresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

// maxCaptureDevices matches the bound captureresolver puts on a device list.
const maxCaptureDevices = 500

const (
	captureMetaName      = "name"
	captureMetaIsDefault = "isDefault"
)

func (r *Deps) resolveCaptureDeviceSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	query := req.SelectQuery.Query
	if len(req.IDs) > 0 {
		query = ""
	}

	result, err := r.CaptureService.ListDevices(ctx, &captureservice.ListDevicesRequest{
		TenantInfo: req.TenantInfo,
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: maxCaptureDevices},
			Query:      query,
		},
		Mine:   true,
		Status: capture.DeviceActive,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	items := make([]selectOptionConnectionItem, 0, len(result.Items))
	for _, device := range withIDs(result.Items, req.IDs, func(d *capture.CaptureDevice) pulid.ID {
		return d.ID
	}) {
		items = append(items, captureDeviceSelectOptionItem(device, now))
	}

	return pageSelectOptionItems(items, req)
}

func captureDeviceSelectOptionItem(
	device *capture.CaptureDevice,
	now int64,
) selectOptionConnectionItem {
	sources := make([]map[string]any, 0, len(device.Sources))
	for i := range device.Sources {
		source := &device.Sources[i]
		sources = append(sources, map[string]any{
			captureMetaName:      source.Name,
			"protocol":           source.Protocol,
			captureMetaIsDefault: source.IsDefault,
		})
	}

	return selectOptionConnectionItemFor(
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

func (r *Deps) resolveCaptureProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	profiles, err := r.CaptureService.AvailableProfiles(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	matched := withIDs(profiles, req.IDs, func(p *capture.CaptureProfile) pulid.ID {
		return p.ID
	})
	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))

	items := make([]selectOptionConnectionItem, 0, len(matched))
	for _, profile := range matched {
		if len(req.IDs) == 0 && query != "" &&
			!strings.Contains(strings.ToLower(profile.Name), query) {
			continue
		}
		items = append(items, captureProfileSelectOptionItem(profile))
	}

	return pageSelectOptionItems(items, req)
}

func captureProfileSelectOptionItem(profile *capture.CaptureProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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
	items []selectOptionConnectionItem,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		return selectOptionConnection(items, len(items), 0)
	}

	total := len(items)
	offset := min(req.SelectQuery.Pagination.SafeOffset(), total)
	end := min(offset+req.SelectQuery.Pagination.SafeLimit(), total)

	return selectOptionConnection(items[offset:end], total, offset)
}
