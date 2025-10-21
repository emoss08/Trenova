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
