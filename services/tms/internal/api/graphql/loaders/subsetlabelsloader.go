package loaders

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

const subsetKeySeparator = "|"

type SubsetLabelsLoaderFactoryParams struct {
	fx.In

	Labeler services.RecordLabeler `optional:"true"`
}

// SubsetLabelsLoaderFactory names the records the proposals on a page offer
// a person to untick. Each record-subset field asks once for all of its
// records, and every field in a batch is read together: one query per
// resource for each MaxRecordLabelsPerResource records, however many
// proposals list them.
type SubsetLabelsLoaderFactory struct {
	labeler services.RecordLabeler
}

func NewSubsetLabelsLoaderFactory(p SubsetLabelsLoaderFactoryParams) *SubsetLabelsLoaderFactory {
	return &SubsetLabelsLoaderFactory{labeler: p.Labeler}
}

// SubsetLabelsKey is a record-subset field's key in the loader: the resource
// its records belong to and their ids.
func SubsetLabelsKey(resource permission.Resource, ids []string) string {
	size := len(resource)
	for _, id := range ids {
		size += len(subsetKeySeparator) + len(id)
	}

	var key strings.Builder
	key.Grow(size)
	key.WriteString(resource.String())
	for _, id := range ids {
		key.WriteString(subsetKeySeparator)
		key.WriteString(id)
	}

	return key.String()
}

func (f *SubsetLabelsLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, services.RecordLabels] {
	return newLoader(f.batchFunc(tenantInfo))
}

// batchFunc answers each field's key with the labels of its records that
// have one; a record that is gone, or an id that names no record, has none.
func (f *SubsetLabelsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[services.RecordLabels] {
	return func(ctx context.Context, keys []string) ([]services.RecordLabels, []error) {
		values := make([]services.RecordLabels, len(keys))
		errs := make([]error, len(keys))
		if f.labeler == nil {
			return values, errs
		}

		parsed := make([]subsetKey, len(keys))
		refs := make(map[permission.Resource][]pulid.ID, 1)
		seen := make(map[pulid.ID]struct{})
		for idx, key := range keys {
			parsed[idx] = parseSubsetKey(key)
			for _, id := range parsed[idx].ids {
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				refs[parsed[idx].resource] = append(refs[parsed[idx].resource], id)
			}
		}
		if len(refs) == 0 {
			return values, errs
		}

		labels, err := f.read(ctx, tenantInfo, refs)
		if err != nil {
			fillMissingErrors(errs, err)

			return values, errs
		}

		for idx := range parsed {
			values[idx] = labels.For(parsed[idx].resource, parsed[idx].ids)
		}

		return values, errs
	}
}

// read labels refs a MaxRecordLabelsPerResource slice of each resource at a
// time, since one Labels read names no more than that of any resource.
func (f *SubsetLabelsLoaderFactory) read(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (services.RecordLabels, error) {
	labels := make(services.RecordLabels, len(refs))
	for offset := 0; ; offset += services.MaxRecordLabelsPerResource {
		chunk := make(map[permission.Resource][]pulid.ID, len(refs))
		for resource, ids := range refs {
			if offset < len(ids) {
				chunk[resource] = ids[offset:min(offset+services.MaxRecordLabelsPerResource, len(ids))]
			}
		}
		if len(chunk) == 0 {
			return labels, nil
		}

		read, err := f.labeler.Labels(ctx, tenantInfo, chunk)
		if err != nil {
			return nil, err
		}
		labels.Merge(read)
	}
}

type subsetKey struct {
	resource permission.Resource
	ids      []pulid.ID
}

func parseSubsetKey(key string) subsetKey {
	parts := strings.Split(key, subsetKeySeparator)
	parsed := subsetKey{
		resource: permission.Resource(parts[0]),
		ids:      make([]pulid.ID, 0, len(parts)-1),
	}
	if parsed.resource == "" {
		return parsed
	}

	for _, part := range parts[1:] {
		if id, err := pulid.Parse(part); err == nil && id.IsNotNil() {
			parsed.ids = append(parsed.ids, id)
		}
	}

	return parsed
}
