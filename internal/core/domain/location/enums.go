/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package location

type Category string

const (
	CategoryTerminal            = Category("Terminal")
	CategoryWarehouse           = Category("Warehouse")
	CategoryDistributionCenter  = Category("DistributionCenter")
	CategoryTruckStop           = Category("TruckStop")
	CategoryRestArea            = Category("RestArea")
	CategoryCustomerLocation    = Category("CustomerLocation")
	CategoryPort                = Category("Port")
	CategoryRailYard            = Category("RailYard")
	CategoryMaintenanceFacility = Category("MaintenanceFacility")
)

type FacilityType string

const (
	FacilityTypeCrossDock          = FacilityType("CrossDock")
	FacilityTypeStorageWarehouse   = FacilityType("StorageWarehouse")
	FacilityTypeColdStorage        = FacilityType("ColdStorage")
	FacilityTypeHazmatFacility     = FacilityType("HazmatFacility")
	FacilityTypeIntermodalFacility = FacilityType("IntermodalFacility")
)
