package shipmentprovider

type ActiveShipmentsCard struct {
	Count               int                       `json:"count"`
	ChangeFromYesterday int                       `json:"changeFromYesterday"`
	Sparkline           []*RevenueSparklinePoint  `json:"sparkline"`
	Breakdown           *ActiveShipmentsBreakdown `json:"breakdown"`
}

type ActiveShipmentsBreakdown struct {
	InTransit int `json:"inTransit"`
	AtRisk    int `json:"atRisk"`
	Loading   int `json:"loading"`
	Done      int `json:"done"`
}

type OnTimeCard struct {
	Percent         float64  `json:"percent"`
	OnTimeCount     int      `json:"onTimeCount"`
	TotalCount      int      `json:"totalCount"`
	Target          *float64 `json:"target,omitempty"`
	DeltaPp         float64  `json:"deltaPp"`
	SevenDayPercent float64  `json:"sevenDayPercent"`
}

type RevenueSparklinePoint struct {
	Hour  string  `json:"hour"`
	Value float64 `json:"value"`
}

type RevenueTodayCard struct {
	Total     float64                  `json:"total"`
	Sparkline []*RevenueSparklinePoint `json:"sparkline"`
	DeltaPct  float64                  `json:"deltaPct"`
	RPM       float64                  `json:"rpm"`
}

type EmptyMileCard struct {
	Percent    float64 `json:"percent"`
	EmptyMiles float64 `json:"emptyMiles"`
	TotalMiles float64 `json:"totalMiles"`
	DeltaPp    float64 `json:"deltaPp"`
}

type AtRiskCard struct {
	Count   int `json:"count"`
	Delta   int `json:"delta"`
	ETASlip int `json:"etaSlip"`
	Weather int `json:"weather"`
	Reefer  int `json:"reefer"`
}

type UnassignedCard struct {
	Count          int     `json:"count"`
	Delta          int     `json:"delta"`
	RevenueWaiting float64 `json:"revenueWaiting"`
}

type ReadyToDispatchCard struct {
	Count       int `json:"count"`
	Delta       int `json:"delta"`
	Unassigned  int `json:"unassigned"`
	DriverReady int `json:"driverReady"`
}

type DetentionWatchlistItem struct {
	ShipmentID   string `json:"shipmentId"`
	Customer     string `json:"customer"`
	DwellLabel   string `json:"dwellLabel"`
	Tone         string `json:"tone"`
	DwellSeconds int64  `json:"-"`
}

type DetentionWatchlistCard struct {
	Items []*DetentionWatchlistItem `json:"items"`
}

type CustomerMixEntry struct {
	CustomerID string  `json:"customerId"`
	Name       string  `json:"name"`
	Revenue    float64 `json:"revenue"`
	Share      float64 `json:"share"`
	Loads      int     `json:"loads"`
	Trend      float64 `json:"trend"`
}

type CustomerMixCard struct {
	WindowDays int                 `json:"windowDays"`
	Entries    []*CustomerMixEntry `json:"entries"`
}

type TomorrowPickupStatus string

const (
	TomorrowPickupStatusScheduled  TomorrowPickupStatus = "scheduled"
	TomorrowPickupStatusConfirmed  TomorrowPickupStatus = "confirmed"
	TomorrowPickupStatusTentative  TomorrowPickupStatus = "tentative"
	TomorrowPickupStatusUnassigned TomorrowPickupStatus = "unassigned"
)

type TomorrowPickup struct {
	ShipmentID        string               `json:"shipmentId"`
	ProNumber         string               `json:"proNumber"`
	PickupWindowStart int64                `json:"pickupWindowStart"`
	Customer          string               `json:"customer"`
	Origin            string               `json:"origin"`
	Destination       string               `json:"destination"`
	Driver            string               `json:"driver"`
	Status            TomorrowPickupStatus `json:"status"`
}

type TomorrowsPickupsCard struct {
	Date    string            `json:"date"`
	Pickups []*TomorrowPickup `json:"pickups"`
}

type LaneHeatmapCell struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Count       int    `json:"count"`
}

type LaneHeatmapCard struct {
	WindowDays int                `json:"windowDays"`
	Cells      []*LaneHeatmapCell `json:"cells"`
	Total      int                `json:"total"`
}
