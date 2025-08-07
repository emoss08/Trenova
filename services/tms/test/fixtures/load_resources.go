/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package fixtures

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/resource"
	"github.com/fatih/color"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var ErrResourceDefinitionAlreadyExists = eris.New("resource definition already exists")

var resrouceDefs = []*resource.ResourceDefinition{
	{
		ResourceType:       permission.ResourceUser,
		DisplayName:        "User",
		TableName:          "users",
		Description:        "User Management and Authentication",
		AllowCustomFields:  false,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceBusinessUnit,
		DisplayName:        "Business Unit",
		TableName:          "business_units",
		Description:        "Business Unit Management",
		AllowCustomFields:  false,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceOrganization,
		DisplayName:        "Organization",
		TableName:          "organizations",
		Description:        "Organization Management",
		AllowCustomFields:  false,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceWorker,
		DisplayName:        "Worker",
		TableName:          "workers",
		Description:        "Worker Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceTractor,
		DisplayName:        "Tractor",
		TableName:          "tractors",
		Description:        "Tractor Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceTrailer,
		DisplayName:        "Trailer",
		TableName:          "trailers",
		Description:        "Trailer Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceFleetCode,
		DisplayName:        "Fleet Code",
		TableName:          "fleet_codes",
		Description:        "Fleet Code Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceCommodity,
		DisplayName:        "Commodity",
		TableName:          "commodities",
		Description:        "Commodity Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceHazardousMaterial,
		DisplayName:        "Hazardous Material",
		TableName:          "hazardous_materials",
		Description:        "Hazardous Material Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceLocation,
		DisplayName:        "Location",
		TableName:          "locations",
		Description:        "Location Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceLocationCategory,
		DisplayName:        "Location Category",
		TableName:          "location_categories",
		Description:        "Location Category Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceEquipmentType,
		DisplayName:        "Equipment Type",
		TableName:          "equipment_types",
		Description:        "Equipment Type Management",
		AllowCustomFields:  true,
		AllowAutomations:   true,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceCustomer,
		DisplayName:        "Customer",
		TableName:          "customers",
		Description:        "Customer Management",
		AllowCustomFields:  true,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceShipment,
		DisplayName:        "Shipment",
		TableName:          "shipments",
		Description:        "Shipment Management",
		AllowCustomFields:  true,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceShipmentMove,
		DisplayName:        "Shipment Move",
		TableName:          "shipment_moves",
		Description:        "Shipment Move Management",
		AllowCustomFields:  true,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
	{
		ResourceType:       permission.ResourceAssignment,
		DisplayName:        "Assignment",
		TableName:          "assignments",
		Description:        "Assign a worker to a movement",
		AllowCustomFields:  true,
		AllowAutomations:   false,
		AllowNotifications: true,
	},
}

func LoadResourceDefinition(ctx context.Context, db *bun.DB) error {
	exists, err := db.NewSelect().Model((*resource.ResourceDefinition)(nil)).Exists(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to check if resource definitions exist")
	}

	if exists {
		return ErrResourceDefinitionAlreadyExists
	}

	// Validate all tables exist before inserting any resource definitions
	for _, def := range resrouceDefs {
		// Check if table exists in the database
		var tableExists bool
		err = db.QueryRow(`
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_schema = 'public' 
                AND table_name = ?
            )
        `, def.TableName).Scan(&tableExists)
		if err != nil {
			return eris.Wrapf(err, "failed to check if table %s exists", def.TableName)
		}

		if !tableExists {
			return eris.Errorf("table %s does not exist in the database", def.TableName)
		}
	}

	// All tables exist, proceed with insertion
	if _, err = db.NewInsert().Model(&resrouceDefs).Exec(ctx); err != nil {
		return eris.Wrap(err, "failed to bulk insert resource definitions")
	}

	color.Green("Successfully loaded %d resource definitions", len(resrouceDefs))
	return nil
}
