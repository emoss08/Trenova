package meilisearch

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/meilisearchtype"
)

func GetIndexName(prefix, orgID, buID string, entityType meilisearchtype.EntityType) string {
	if prefix == "" {
		prefix = "trenova"
	}
	return fmt.Sprintf("%s_%s_%s_%s", prefix, orgID, buID, entityType)
}

func GetIndexConfig(entityType meilisearchtype.EntityType) meilisearchtype.IndexConfig {
	switch entityType {
	case meilisearchtype.EntityTypeShipment:
		return getShipmentIndexConfig()
	case meilisearchtype.EntityTypeCustomer:
		return getCustomerIndexConfig()
	default:
		return getDefaultIndexConfig()
	}
}

func getShipmentIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.proNumber",
			"metadata.bol",
			"metadata.customerName",
			"metadata.status",
		},
		FilterableAttributes: []string{
			"entityType",
			"organizationId",
			"businessUnitId",
			"metadata.status",
			"createdAt",
			"updatedAt",
		},
		SortableAttributes: []string{
			"createdAt",
			"updatedAt",
		},
		DisplayedAttributes: []string{
			"id",
			"entityType",
			"title",
			"subtitle",
			"metadata",
			"createdAt",
			"updatedAt",
		},
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
		StopWords: []string{},
	}
}

func getCustomerIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.name",
			"metadata.code",
		},
		FilterableAttributes: []string{
			"entityType",
			"organizationId",
			"businessUnitId",
			"metadata.status",
			"createdAt",
			"updatedAt",
		},
		SortableAttributes: []string{
			"createdAt",
			"updatedAt",
			"metadata.name",
		},
		DisplayedAttributes: []string{
			"id",
			"entityType",
			"title",
			"subtitle",
			"metadata",
			"createdAt",
			"updatedAt",
		},
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
		StopWords: []string{},
	}
}

func getDefaultIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
		},
		FilterableAttributes: []string{
			"entityType",
			"organizationId",
			"businessUnitId",
			"createdAt",
			"updatedAt",
		},
		SortableAttributes: []string{
			"createdAt",
			"updatedAt",
		},
		DisplayedAttributes: []string{
			"id",
			"entityType",
			"title",
			"subtitle",
			"metadata",
			"createdAt",
			"updatedAt",
		},
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
		StopWords: []string{},
	}
}
