// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package fixtures

import (
	"context"
	"log"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadResources(ctx context.Context, db *bun.DB) error {
	count, err := db.NewSelect().Model((*models.Resource)(nil)).Count(ctx)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Loading resources...")

		resources := []*models.Resource{
			{
				Type:        "BusinessUnit",
				Description: "Represents a business unit in the system.",
			},
			{
				Type:        "Organization",
				Description: "Represents an organization in the system.",
			},
			{
				Type:        "Role",
				Description: "Represents a role in the system.",
			},
			{
				Type:        "UsState",
				Description: "Represents a US State in the system.",
			},
			{
				Type:        "UserFavorite",
				Description: "Represents the user's favorite items in the system.",
			},
			{
				Type:        "UserNotification",
				Description: "Represents the user's notifications in the system.",
			},
			{
				Type:        "Permission",
				Description: "Represents a permission in the system.",
			},
			{
				Type:        "User",
				Description: "Represents a user in the system.",
			},
			{
				Type:        "TableChangeAlert",
				Description: "Represents a table change alert in the system.",
			},
			{
				Type:        "FleetCode",
				Description: "Represents a fleet code in the system.",
			},
			{
				Type:        "ChargeType",
				Description: "Represents a charge type in the system.",
			},
			{
				Type:        "CommentType",
				Description: "Represents a comment type in the system.",
			},
			{
				Type:        "DelayCode",
				Description: "Represents a delay code in the system.",
			},
			{
				Type:        "GeneralLedgerAccount",
				Description: "Represents a general ledger account in the system.",
			},
			{
				Type:        "Tag",
				Description: "Represents a tag in the system.",
			},
			{
				Type:        "LocationCategory",
				Description: "Represents a location category in the system.",
			},
			{
				Type:        "DivisionCode",
				Description: "Represents a division code in the system.",
			},
			{
				Type:        "DocumentClassification",
				Description: "Represents a document classification in the system.",
			},
			{
				Type:        "EquipmentType",
				Description: "Represents a equipment type in the system.",
			},
			{
				Type:        "RevenueCode",
				Description: "Represents a revenue code in the system.",
			},
			{
				Type:        "EquipmentManufacturer",
				Description: "Represents a equipment manufacturer in the system.",
			},
			{
				Type:        "LocationCategory",
				Description: "Represents a location category in the system.",
			},
			{
				Type:        "UserTask",
				Description: "Represents a user task in the system.",
			},
			{
				Type:        "HazardousMaterial",
				Description: "Represents a hazardous material in the system.",
			},
			{
				Type:        "Trailer",
				Description: "Represents a trailer in the system.",
			},
			{
				Type:        "ShipmentType",
				Description: "Represents a shipment type in the system.",
			},
			{
				Type:        "ReasonCode",
				Description: "Represents a reason code in the system.",
			},
			{
				Type:        "Commodity",
				Description: "Represents a commodity in the system.",
			},
			{
				Type:        "ServiceType",
				Description: "Represents a service type in the system.",
			},
			{
				Type:        "QualifierCode",
				Description: "Represents a qualifier code in the system.",
			},
			{
				Type:        "Trailer",
				Description: "Represents a trailer in the system.",
			},
			{
				Type:        "Tractor",
				Description: "Represents a tractor in the system.",
			},
			{
				Type:        "Worker",
				Description: "Represents a worker in the system.",
			},
			{
				Type:        "MasterKeyGeneration",
				Description: "Represents a master key generation in the system.",
			},
			{
				Type:        "WorkerMasterKeyGeneration",
				Description: "Represents a worker master key generation in the system.",
			},
			{
				Type:        "LocationMasterKeyGeneration",
				Description: "Represents a location master key generation in the system.",
			},
			{
				Type:        "CustomerMasterKeyGeneration",
				Description: "Represents a customer master key generation in the system.",
			},
			{
				Type:        "Location",
				Description: "Represents a location in the system.",
			},
			{
				Type:        "Shipment",
				Description: "Represents a shipment in the system.",
			},
			{
				Type:        "ShipmentMove",
				Description: "Represents a shipment move in the system.",
			},
			{
				Type:        "Stop",
				Description: "Represents a stop in the system.",
			},
			{
				Type:        "Customer",
				Description: "Represents a customer in the system.",
			},
			{
				Type:        "TractorAssignment",
				Description: "Represents a tractor assignment in the system.",
			},
			{
				Type:        "AccessorialCharge",
				Description: "Represents a accessorial charge in the system.",
			},
		}

		_, err = db.NewInsert().Model(&resources).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
