package formulatemplate

// * Category represents the type of rate calculation formula
type Category string

const (
	CategoryBaseRate          Category = "BaseRate"
	CategoryDistanceBased     Category = "DistanceBased"
	CategoryWeightBased       Category = "WeightBased"
	CategoryDimensionalWeight Category = "DimensionalWeight"
	CategoryFuelSurcharge     Category = "FuelSurcharge"
	CategoryAccessorial       Category = "Accessorial"
	CategoryTimeBasedRate     Category = "TimeBasedRate"
	CategoryZoneBased         Category = "ZoneBased"
	CategoryCustom            Category = "Custom"
)

// * String returns the string representation of the category
func (c Category) String() string {
	return string(c)
}

// * IsValid checks if the category is valid
func (c Category) IsValid() bool {
	switch c {
	case CategoryBaseRate, CategoryDistanceBased, CategoryWeightBased,
		CategoryDimensionalWeight, CategoryFuelSurcharge, CategoryAccessorial,
		CategoryTimeBasedRate, CategoryZoneBased, CategoryCustom:
		return true
	}
	return false
}

// * GetDescription returns a human-readable description of the category
func (c Category) GetDescription() string {
	switch c {
	case CategoryBaseRate:
		return "Basic flat rate calculation"
	case CategoryDistanceBased:
		return "Rate based on distance traveled"
	case CategoryWeightBased:
		return "Rate based on shipment weight"
	case CategoryDimensionalWeight:
		return "Rate based on dimensional weight calculation"
	case CategoryFuelSurcharge:
		return "Fuel surcharge calculation"
	case CategoryAccessorial:
		return "Additional service charges"
	case CategoryTimeBasedRate:
		return "Rate based on transit time"
	case CategoryZoneBased:
		return "Rate based on shipping zones"
	case CategoryCustom:
		return "Custom formula for complex calculations"
	default:
		return "Unknown category"
	}
}
