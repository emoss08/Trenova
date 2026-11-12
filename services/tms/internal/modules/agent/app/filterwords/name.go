package filterwords

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NameRecords writes the filters with the names of the records they narrow
// by, in one read for all of them. A record with no name, and every record
// when the read fails, stays counted: an id is never shown.
func NameRecords(
	ctx context.Context,
	labeler agentruntimeports.RecordLabeler,
	tenant pagination.TenantInfo,
	filters []Filter,
) error {
	ids := Records(filters)
	resources := make(map[string]permission.Resource, len(ids))
	refs := make(map[permission.Resource][]pulid.ID)
	for _, id := range ids {
		prefix, _, _ := strings.Cut(id, "_")
		kind, known := recordkind.OfIDPrefix(prefix + "_")
		if !known {
			continue
		}
		resource := kind.Resource()
		resources[id] = resource
		refs[resource] = append(refs[resource], pulid.ID(id))
	}
	if len(refs) == 0 {
		return nil
	}

	labels, err := labeler.Labels(ctx, tenant, refs)
	if err != nil {
		return fmt.Errorf("name the records a list was narrowed by: %w", err)
	}
	Name(filters, func(id string) string {
		return labels.Label(resources[id], pulid.ID(id))
	})

	return nil
}
