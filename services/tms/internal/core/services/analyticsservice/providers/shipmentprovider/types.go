package shipmentprovider

type ActiveShipmentsCard struct {
	Count               int `json:"count"`
	ChangeFromYesterday int `json:"changeFromYesterday"`
}

type OnTimeCard struct {
	Percent     float64 `json:"percent"`
	OnTimeCount int     `json:"onTimeCount"`
	TotalCount  int     `json:"totalCount"`
}

type RevenueSparklinePoint struct {
	Hour  string  `json:"hour"`
	Value float64 `json:"value"`
}

type RevenueTodayCard struct {
	Total     float64                  `json:"total"`
	Sparkline []*RevenueSparklinePoint `json:"sparkline"`
}

type EmptyMileCard struct {
	Percent    float64 `json:"percent"`
	EmptyMiles float64 `json:"emptyMiles"`
	TotalMiles float64 `json:"totalMiles"`
}

type ReadyToDispatchCard struct {
	Count int `json:"count"`
}

type DetentionAlertsCard struct {
	Count int `json:"count"`
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
