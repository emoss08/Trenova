package shipmentprovider

type ShipmentCountCard struct {
	Count           int `json:"count"`
	TrendPercentage int `json:"trendPercentage"` // * The percentage change in the number of shipments from the previous month
}
