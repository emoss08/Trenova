package globalsearchservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"go.uber.org/zap"
)

// restrictToVisible keeps only the hits the database's read scope lets the
// person read. The index holds every record of the tenant, so a shipment,
// order, invoice or worker hit, or a document attached to one, is checked
// against the database before it is shown; a check that fails hides them.
// A narrowed group reports only what it shows, so its total cannot count
// records the person may not see.
func (s *Service) restrictToVisible(
	ctx context.Context,
	req *kernelports.GlobalSearchRequest,
	access *searchAccess,
	results []*indexResult,
) {
	visible := s.visibleRecords(ctx, req, access, scopedHitIDs(s.visibility, results))

	for _, result := range results {
		if result == nil || len(result.hits) == 0 {
			continue
		}
		kept := result.hits[:0]
		for _, hit := range result.hits {
			kind, id := hitRecord(&result.definition, hit)
			if !s.scoped(kind) {
				kept = append(kept, hit)
				continue
			}
			if _, ok := visible[kind][id]; ok {
				kept = append(kept, hit)
			}
		}
		removed := len(result.hits) - len(kept)
		result.hits = kept
		if removed > 0 || s.groupNarrowed(access, &result.definition) {
			result.total = len(kept)
		}
	}
}

func (s *Service) scoped(kind string) bool {
	return kind != "" && s.visibility != nil && s.visibility.Scoped(kind)
}

func (s *Service) groupNarrowed(access *searchAccess, definition *entitySearchDefinition) bool {
	if definition.entityType != search.EntityTypeDocument {
		return access.narrowed[definition.resource]
	}
	for resource, narrowed := range access.narrowed {
		if narrowed && s.scoped(resource.String()) {
			return true
		}
	}
	return false
}

func (s *Service) visibleRecords(
	ctx context.Context,
	req *kernelports.GlobalSearchRequest,
	access *searchAccess,
	idsByKind map[string][]pulid.ID,
) map[string]map[pulid.ID]struct{} {
	visible := make(map[string]map[pulid.ID]struct{}, len(idsByKind))
	for kind, ids := range idsByKind {
		visible[kind] = map[pulid.ID]struct{}{}
		if !access.canRead(permission.Resource(kind)) {
			continue
		}
		found, err := s.visibility.VisibleIDs(ctx, &kernelports.ListVisibleRecordsRequest{
			TenantInfo: req.TenantInfo,
			Kind:       kind,
			IDs:        uniqueIDs(ids),
		})
		if err != nil {
			s.logger.Warn(
				"failed to check which search hits are visible; hiding them",
				zap.String("kind", kind),
				zap.Error(err),
			)
			continue
		}
		for _, id := range found {
			visible[kind][id] = struct{}{}
		}
	}
	return visible
}

func scopedHitIDs(
	visibility kernelports.RecordVisibility,
	results []*indexResult,
) map[string][]pulid.ID {
	idsByKind := make(map[string][]pulid.ID)
	if visibility == nil {
		return idsByKind
	}
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, hit := range result.hits {
			kind, id := hitRecord(&result.definition, hit)
			if kind == "" || id.IsNil() || !visibility.Scoped(kind) {
				continue
			}
			idsByKind[kind] = append(idsByKind[kind], id)
		}
	}
	return idsByKind
}

// hitRecord names the record whose visibility decides whether a hit is shown:
// the hit's own record, or for a document the record it is attached to.
func hitRecord(
	definition *entitySearchDefinition,
	hit *kernelports.GlobalSearchHit,
) (string, pulid.ID) {
	kind := definition.resource.String()
	rawID := hit.ID
	if definition.entityType == search.EntityTypeDocument {
		kind = hit.Metadata["resourceType"]
		rawID = hit.Metadata["resourceId"]
	}
	id, err := pulid.Parse(rawID)
	if err != nil {
		return kind, pulid.Nil
	}
	return kind, id
}
