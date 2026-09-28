package loaders

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

const captureRecordKeySeparator = ":"

// CaptureRecordKey is the loader key for one record a capture points at. A
// capture can point at six kinds of record, so the key carries the kind.
func CaptureRecordKey(resourceType string, id pulid.ID) string {
	return resourceType + captureRecordKeySeparator + id.String()
}

type captureRecordLabeler interface {
	Labels(
		ctx context.Context,
		req *repositories.ListCaptureRecordLabelsRequest,
	) ([]*repositories.CaptureRecordLabel, error)
}

type CaptureRecordLabelLoaderFactoryParams struct {
	fx.In

	Records repositories.CaptureRecordFinder
}

type CaptureRecordLabelLoaderFactory struct {
	records captureRecordLabeler
}

func NewCaptureRecordLabelLoaderFactory(
	p CaptureRecordLabelLoaderFactoryParams,
) *CaptureRecordLabelLoaderFactory {
	return &CaptureRecordLabelLoaderFactory{records: p.Records}
}

func (f *CaptureRecordLabelLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *repositories.CaptureRecordLabel] {
	return newLoader(f.batchFunc(tenantInfo))
}

// captureRecordKeys is a page of loader keys grouped by record kind, with
// where each key sits in the page.
type captureRecordKeys struct {
	idsByKind map[string][]pulid.ID
	indexes   map[string][]int
}

func groupCaptureRecordKeys(keys []string, errs []error) captureRecordKeys {
	grouped := captureRecordKeys{
		idsByKind: make(map[string][]pulid.ID),
		indexes:   make(map[string][]int, len(keys)),
	}

	for idx, key := range keys {
		kind, rawID, ok := strings.Cut(key, captureRecordKeySeparator)
		if !ok || kind == "" {
			errs[idx] = errortypes.NewValidationError(
				"id", errortypes.ErrInvalid, "Record key is malformed")

			continue
		}
		id, err := parseLoaderID(rawID)
		if err != nil {
			errs[idx] = err

			continue
		}
		if _, seen := grouped.indexes[key]; !seen {
			grouped.idsByKind[kind] = append(grouped.idsByKind[kind], id)
		}
		grouped.indexes[key] = append(grouped.indexes[key], idx)
	}

	return grouped
}

// batchFunc names a page of records in one query per kind. A record that is
// gone loads as nil rather than failing: the capture still says where it was
// meant to go, and the row shows that without a name.
func (f *CaptureRecordLabelLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*repositories.CaptureRecordLabel] {
	return func(ctx context.Context, keys []string) ([]*repositories.CaptureRecordLabel, []error) {
		values := make([]*repositories.CaptureRecordLabel, len(keys))
		errs := make([]error, len(keys))
		grouped := groupCaptureRecordKeys(keys, errs)

		for kind, ids := range grouped.idsByKind {
			labels, err := f.records.Labels(ctx, &repositories.ListCaptureRecordLabelsRequest{
				TenantInfo:   tenantInfo,
				ResourceType: kind,
				IDs:          ids,
			})
			if err != nil {
				for _, id := range ids {
					for _, idx := range grouped.indexes[CaptureRecordKey(kind, id)] {
						errs[idx] = err
					}
				}

				continue
			}
			for _, label := range labels {
				for _, idx := range grouped.indexes[CaptureRecordKey(kind, label.ID)] {
					values[idx] = label
				}
			}
		}

		return values, errs
	}
}
