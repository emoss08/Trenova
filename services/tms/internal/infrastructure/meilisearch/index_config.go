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
	case meilisearchtype.EntityTypeInvoice:
		return getInvoiceIndexConfig()
	case meilisearchtype.EntityTypeWorker:
		return getWorkerIndexConfig()
	case meilisearchtype.EntityTypeCustomer:
		return getCustomerIndexConfig()
	case meilisearchtype.EntityTypeCommodity:
		return getCommodityIndexConfig()
	case meilisearchtype.EntityTypeLocation:
		return getLocationIndexConfig()
	case meilisearchtype.EntityTypeHazardousMaterial:
		return getHazardousMaterialIndexConfig()
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

func getInvoiceIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.invoiceNumber",
			"metadata.customerName",
			"metadata.amount",
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
			"metadata.amount",
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

func getWorkerIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.firstName",
			"metadata.lastName",
			"metadata.code",
			"metadata.email",
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
			"metadata.lastName",
			"metadata.firstName",
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
			"metadata.email",
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

func getCommodityIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.name",
			"metadata.description",
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

func getLocationIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.name",
			"metadata.code",
			"metadata.address",
			"metadata.city",
			"metadata.state",
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

func getHazardousMaterialIndexConfig() meilisearchtype.IndexConfig {
	return meilisearchtype.IndexConfig{
		SearchableAttributes: []string{
			"title",
			"content",
			"metadata.name",
			"metadata.hazardClass",
			"metadata.packingGroup",
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
