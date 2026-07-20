package shipment

type FuelSurchargeDetail struct {
	ProgramID       string   `json:"programId"`
	ProgramName     string   `json:"programName"`
	ProgramCode     string   `json:"programCode"`
	Method          string   `json:"method"`
	IndexID         string   `json:"indexId"`
	IndexCode       string   `json:"indexCode"`
	IndexSource     string   `json:"indexSource"`
	IndexRegion     string   `json:"indexRegion,omitempty"`
	IndexFuelType   string   `json:"indexFuelType,omitempty"`
	EIASeriesID     string   `json:"eiaSeriesId,omitempty"`
	PriceDate       string   `json:"priceDate"`
	Price           float64  `json:"price"`
	Currency        string   `json:"currency"`
	PegPrice        *float64 `json:"pegPrice,omitempty"`
	Increment       *float64 `json:"increment,omitempty"`
	IncrementRate   *float64 `json:"incrementRate,omitempty"`
	MilesPerGallon  *float64 `json:"milesPerGallon,omitempty"`
	BandMin         *float64 `json:"bandMin,omitempty"`
	BandMax         *float64 `json:"bandMax,omitempty"`
	BandValue       *float64 `json:"bandValue,omitempty"`
	Miles           *float64 `json:"miles,omitempty"`
	RatePerMile     *float64 `json:"ratePerMile,omitempty"`
	Percent         *float64 `json:"percent,omitempty"`
	PercentBasis    string   `json:"percentBasis,omitempty"`
	LinehaulBase    *float64 `json:"linehaulBase,omitempty"`
	AccessorialBase *float64 `json:"accessorialBase,omitempty"`
	RawAmount       float64  `json:"rawAmount"`
	Amount          float64  `json:"amount"`
	CapApplied      bool     `json:"capApplied"`
	FloorApplied    bool     `json:"floorApplied"`
	StepRounding    string   `json:"stepRounding,omitempty"`
	RateRounding    string   `json:"rateRounding,omitempty"`
	RatePrecision   int16    `json:"ratePrecision,omitempty"`
	DateBasis       string   `json:"dateBasis"`
	BasisDate       string   `json:"basisDate"`
	UsedFallback    bool     `json:"usedFallback"`
	Stale           bool     `json:"stale"`
	CalculatedAt    int64    `json:"calculatedAt"`
}
