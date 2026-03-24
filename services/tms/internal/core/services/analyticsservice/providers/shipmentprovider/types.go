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
