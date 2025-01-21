package location

type LocationCategoryType string

const (
	LocationCategoryTypeTerminal            = LocationCategoryType("Terminal")
	LocationCategoryTypeWarehouse           = LocationCategoryType("Warehouse")
	LocationCategoryTypeDistributionCenter  = LocationCategoryType("DistributionCenter")
	LocationCategoryTypeTruckStop           = LocationCategoryType("TruckStop")
	LocationCategoryTypeRestArea            = LocationCategoryType("RestArea")
	LocationCategoryTypeCustomerLocation    = LocationCategoryType("CustomerLocation")
	LocationCategoryTypePort                = LocationCategoryType("Port")
	LocationCategoryTypeRailYard            = LocationCategoryType("RailYard")
	LocationCategoryTypeMaintenanceFacility = LocationCategoryType("MaintenanceFacility")
)

type FacilityType string

const (
	FacilityTypeCrossDock          = FacilityType("CrossDock")
	FacilityTypeStorageWarehouse   = FacilityType("StorageWarehouse")
	FacilityTypeColdStorage        = FacilityType("ColdStorage")
	FacilityTypeHazmatFacility     = FacilityType("HazmatFacility")
	FacilityTypeIntermodalFacility = FacilityType("IntermodalFacility")
)
