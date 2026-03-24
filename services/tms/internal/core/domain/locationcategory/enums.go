package locationcategory

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

func (c Category) String() string {
	return string(c)
}

func (c Category) IsValid() bool {
	switch c {
	case CategoryTerminal, CategoryWarehouse, CategoryDistributionCenter,
		CategoryTruckStop, CategoryRestArea, CategoryCustomerLocation,
		CategoryPort, CategoryRailYard, CategoryMaintenanceFacility:
		return true
	default:
		return false
	}
}

type FacilityType string

const (
	FacilityTypeCrossDock          = FacilityType("CrossDock")
	FacilityTypeStorageWarehouse   = FacilityType("StorageWarehouse")
	FacilityTypeColdStorage        = FacilityType("ColdStorage")
	FacilityTypeHazmatFacility     = FacilityType("HazmatFacility")
	FacilityTypeIntermodalFacility = FacilityType("IntermodalFacility")
)

func (f FacilityType) String() string {
	return string(f)
}

func (f FacilityType) IsValid() bool {
	switch f {
	case FacilityTypeCrossDock, FacilityTypeStorageWarehouse,
		FacilityTypeColdStorage, FacilityTypeHazmatFacility,
		FacilityTypeIntermodalFacility:
		return true
	default:
		return false
	}
}
