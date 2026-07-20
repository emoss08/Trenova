package fuelsurcharge

type EIASeriesDef struct {
	SeriesID string
	Code     string
	Name     string
	Region   string
	FuelType FuelType
}

var eiaSeriesRegistry = []EIASeriesDef{
	{
		SeriesID: "EMD_EPD2D_PTE_NUS_DPG",
		Code:     "DOE_US",
		Name:     "U.S. National Average",
		Region:   "US",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R10_DPG",
		Code:     "DOE_EAST_COAST",
		Name:     "East Coast (PADD 1)",
		Region:   "PADD 1",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R1X_DPG",
		Code:     "DOE_NEW_ENGLAND",
		Name:     "New England (PADD 1A)",
		Region:   "PADD 1A",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R1Y_DPG",
		Code:     "DOE_CENTRAL_ATLANTIC",
		Name:     "Central Atlantic (PADD 1B)",
		Region:   "PADD 1B",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R1Z_DPG",
		Code:     "DOE_LOWER_ATLANTIC",
		Name:     "Lower Atlantic (PADD 1C)",
		Region:   "PADD 1C",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R20_DPG",
		Code:     "DOE_MIDWEST",
		Name:     "Midwest (PADD 2)",
		Region:   "PADD 2",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R30_DPG",
		Code:     "DOE_GULF_COAST",
		Name:     "Gulf Coast (PADD 3)",
		Region:   "PADD 3",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R40_DPG",
		Code:     "DOE_ROCKY_MOUNTAIN",
		Name:     "Rocky Mountain (PADD 4)",
		Region:   "PADD 4",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R50_DPG",
		Code:     "DOE_WEST_COAST",
		Name:     "West Coast (PADD 5)",
		Region:   "PADD 5",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_R5XCA_DPG",
		Code:     "DOE_WEST_COAST_NO_CA",
		Name:     "West Coast less California",
		Region:   "PADD 5 (excl. CA)",
		FuelType: FuelTypeDiesel,
	},
	{
		SeriesID: "EMD_EPD2D_PTE_SCA_DPG",
		Code:     "DOE_CALIFORNIA",
		Name:     "California",
		Region:   "California",
		FuelType: FuelTypeDiesel,
	},
}

func EIASeriesRegistry() []EIASeriesDef {
	out := make([]EIASeriesDef, len(eiaSeriesRegistry))
	copy(out, eiaSeriesRegistry)
	return out
}

func EIASeriesByID(seriesID string) (EIASeriesDef, bool) {
	for _, def := range eiaSeriesRegistry {
		if def.SeriesID == seriesID {
			return def, true
		}
	}
	return EIASeriesDef{}, false
}
