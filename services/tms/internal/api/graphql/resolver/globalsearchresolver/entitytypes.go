package globalsearchresolver

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/types/search"
)

const defaultSearchLimit = 8

func parseEntityTypes(values []string) ([]search.EntityType, error) {
	entityTypes := make([]search.EntityType, 0, len(values))
	seen := make(map[search.EntityType]struct{}, len(values))
	for _, value := range values {
		entityType := search.EntityType(strings.TrimSpace(value))
		if entityType == "" {
			continue
		}
		if !isSearchableEntityType(entityType) {
			return nil, errortypes.NewValidationError(
				"entityTypes",
				errortypes.ErrInvalid,
				"Invalid global search entity type",
			)
		}
		if _, ok := seen[entityType]; ok {
			continue
		}
		seen[entityType] = struct{}{}
		entityTypes = append(entityTypes, entityType)
	}
	if len(entityTypes) == 0 {
		return nil, nil
	}

	return entityTypes, nil
}

func isSearchableEntityType(entityType search.EntityType) bool {
	switch entityType {
	case search.EntityTypeShipment,
		search.EntityTypeCustomer,
		search.EntityTypeWorker,
		search.EntityTypeDocument:
		return true
	default:
		return false
	}
}
