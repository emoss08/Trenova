package apikeyprovider

type TotalKeysCard struct {
	Count        int `json:"count"`
	NewThisMonth int `json:"newThisMonth"`
}

type ActiveKeysCard struct {
	Count          int     `json:"count"`
	PercentOfTotal float64 `json:"percentOfTotal"`
}

type RevokedKeysCard struct {
	Count          int     `json:"count"`
	PercentOfTotal float64 `json:"percentOfTotal"`
}

type SparklinePoint struct {
	Day   string `json:"day"`
	Value int64  `json:"value"`
}

type Requests30dCard struct {
	Total     int64             `json:"total"`
	Sparkline []*SparklinePoint `json:"sparkline"`
}
