package pcmiler

import "strconv"

type Config struct {
	APIKey  string
	BaseURL string
	Timeout int
}

type RouteOptions struct {
	DataVersion         string
	Region              string
	RoutingType         string
	DistanceUnits       string
	VehicleType         string
	LocationGranularity string
	ProfileName         string
	HighwayOnly         bool
	TollRoads           bool
	BordersOpen         bool
	VehicleHeight       float64
	VehicleLength       float64
	VehicleWidth        float64
	VehicleWeight       float64
	DimensionUnits      string
	WeightUnits         string
	Axles               int
	Hazmat              []string
	IncludeTollData     bool
	StateReport         bool
}

type Stop struct {
	City           string
	State          string
	PostalCode     string
	AddressLine    string
	Latitude       *float64
	Longitude      *float64
	TrimblePlaceID string
	Country        string
}

type RouteRequest struct {
	RouteID string
	Stops   []Stop
	Options RouteOptions
}

type JurisdictionDistance struct {
	Country  string
	Code     string
	Distance float64
	Toll     float64
	Ferry    float64
}

type RouteMileage struct {
	RouteID               string
	Distance              float64
	DataVersion           string
	Warnings              []string
	RawSummary            map[string]any
	JurisdictionDistances []JurisdictionDistance
}

type Version struct {
	Name string `json:"name"`
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return "PC*Miler returned status " + strconv.Itoa(e.StatusCode) + ": " + e.Message
}
