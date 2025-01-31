package pcmiler

// SingleSearchURL is the URL for the single search API
var SingleSearchURL = "https://singlesearch.alk.com/NA/api/search"

type SingleSearchRequest struct {
	Query string `json:"query"`
}

// Address struct represents address details
type Address struct {
	StreetAddress   string  `json:"StreetAddress"`
	LocalArea       string  `json:"LocalArea"`
	City            string  `json:"City"`
	State           string  `json:"State"`
	StateName       string  `json:"StateName"`
	Zip             string  `json:"Zip"`
	County          string  `json:"County"`
	Country         string  `json:"Country"`
	CountryFullName string  `json:"CountryFullName"`
	SPLC            *string `json:"SPLC"`
}

// Coords struct represents latitude and longitude
type Coords struct {
	Lat string `json:"Lat"`
	Lon string `json:"Lon"`
}

// Region indicates the region of the location.
type Region int

const (
	RegionUnknown    = Region(0) // Unknown
	RegionAF         = Region(1) // Africa
	RegionAS         = Region(2) // Asia
	RegionEU         = Region(3) // Europe
	RegionNA         = Region(4) // North America (Default)
	RegionOC         = Region(5) // Oceania
	RegionSA         = Region(6) // South America
	RegionME         = Region(7) // Middle East
	RegionDeprecated = Region(8) // Deprecated
	RegionMX         = Region(9) // Mexico
)

// ResultType indicates the type of match with the search string.
type ResultType int

const (
	ResultTypeCountry       = ResultType(0)  // Country
	ResultTypeState         = ResultType(1)  // State
	ResultTypeCounty        = ResultType(2)  // County
	ResultTypeCity          = ResultType(3)  // City
	ResultTypeZip           = ResultType(4)  // Zip
	ResultTypeSPLC          = ResultType(5)  // SPLC
	ResultTypeStreet        = ResultType(6)  // Street
	ResultTypeRouteNumber   = ResultType(7)  // RouteNumber
	ResultTypeRouteAlpha    = ResultType(8)  // RouteAlpha
	ResultTypePOI           = ResultType(9)  // POI
	ResultTypePOIStreet     = ResultType(10) // POIStreet
	ResultTypeFullPostCode  = ResultType(11) // FullPostCode
	ResultTypePOIType       = ResultType(12) // POIType
	ResultTypeCrossStreet   = ResultType(13) // CrossStreet
	ResultTypeLatLon        = ResultType(14) // LatLon
	ResultTypeCustomPlace   = ResultType(15) // CustomPlace
	ResultTypeNone          = ResultType(16) // None
	ResultTypeTrimblePlaces = ResultType(17) // TrimblePlaces
)

// Location struct represents a single location with address, coordinates, and other metadata
type Location struct {
	Address         Address `json:"Address"`
	Coords          Coords  `json:"Coords"`
	Region          int     `json:"Region"`
	POITypeID       int     `json:"POITypeID"`
	PersistentPOIID int     `json:"PersistentPOIID"`
	SiteID          int     `json:"SiteID"`
	ResultType      int     `json:"ResultType"`
	ShortString     string  `json:"ShortString"`
	TimeZone        string  `json:"TimeZone"`
}

// LocationResponse represents the entire response containing multiple locations
type LocationResponse struct {
	Err       int        `json:"Err"`
	Locations []Location `json:"Locations"`
}
