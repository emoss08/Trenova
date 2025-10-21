package m2msync

import (
	"context"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

// Example usage patterns for the many-to-many sync utility

// Example 1: Syncing user permissions (direct IDs)
func ExampleUserPermissions(
	ctx context.Context,
	tx bun.IDB,
	syncer *Syncer,
	userID pulid.ID,
	permissionIDs []pulid.ID,
) error {
	config := Config{
		Table:       "user_permissions",
		SourceField: "user_id",
		TargetField: "permission_id",
		AdditionalFields: map[string]any{
			"granted_at": "NOW()",
		},
	}

	return syncer.SyncIDs(ctx, tx, config, userID, permissionIDs)
}

// Example 2: Syncing product categories
func ExampleProductCategories(
	ctx context.Context,
	tx bun.IDB,
	syncer *Syncer,
	productID pulid.ID,
	orgID pulid.ID,
	categories []Category, // Assuming Category has an ID field
) error {
	config := Config{
		Table:       "product_categories",
		SourceField: "product_id",
		TargetField: "category_id",
		AdditionalFields: map[string]any{
			"organization_id": orgID,
		},
	}

	return syncer.SyncEntities(ctx, tx, config, productID, categories)
}

// Example 3: Syncing team members
func ExampleTeamMembers(
	ctx context.Context,
	tx bun.IDB,
	syncer *Syncer,
	teamID pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
	memberIDs []pulid.ID,
) error {
	config := Config{
		Table:       "team_members",
		SourceField: "team_id",
		TargetField: "member_id",
		AdditionalFields: map[string]any{
			"organization_id":  orgID,
			"business_unit_id": buID,
			"joined_at":        "NOW()",
		},
	}

	return syncer.SyncIDs(ctx, tx, config, teamID, memberIDs)
}

// Example 4: Syncing shipment commodities with complex metadata
func ExampleShipmentCommodities(
	ctx context.Context,
	tx bun.IDB,
	syncer *Syncer,
	shipmentID pulid.ID,
	orgID pulid.ID,
	commodityData []CommodityAssignment,
) error {
	// Extract commodity IDs from assignment data
	commodityIDs := make([]pulid.ID, 0, len(commodityData))
	for _, assignment := range commodityData {
		commodityIDs = append(commodityIDs, assignment.CommodityID)
	}

	config := Config{
		Table:       "shipment_commodities",
		SourceField: "shipment_id",
		TargetField: "commodity_id",
		AdditionalFields: map[string]any{
			"organization_id": orgID,
			"loaded_at":       "NOW()",
		},
	}

	return syncer.SyncIDs(ctx, tx, config, shipmentID, commodityIDs)
}

// Example 5: Syncing document tags
func ExampleDocumentTags(
	ctx context.Context,
	tx bun.IDB,
	syncer *Syncer,
	documentID pulid.ID,
	tags []Tag, // Assuming Tag has an ID field
) error {
	config := Config{
		Table:       "document_tags",
		SourceField: "document_id",
		TargetField: "tag_id",
		// No additional fields needed for simple tagging
	}

	return syncer.SyncEntities(ctx, tx, config, documentID, tags)
}

// Types for examples (these would be your actual domain models)
type Category struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type Tag struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type CommodityAssignment struct {
	CommodityID pulid.ID `json:"commodityId"`
	Quantity    int      `json:"quantity"`
	Weight      float64  `json:"weight"`
}
