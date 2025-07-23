// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package pcmiler

var (
	// SingleSearchURL is the URL for the single search API
	SingleSearchURL = "https://singlesearch.alk.com/NA/api/search"

	// RouteReportURL is the URL for the route report API
	RouteReportURL = "https://pcmiler.alk.com/apis/rest/v1.0/Service.svc/route/routeReports"
)

// Address struct represents address details
type Address struct {
	StreetAddress   string  `json:"streetAddress"`
	LocalArea       string  `json:"localArea"`
	City            string  `json:"city"`
	State           string  `json:"state"`
	StateName       string  `json:"stateName"`
	Zip             string  `json:"zip"`
	County          string  `json:"county"`
	Country         string  `json:"country"`
	CountryFullName string  `json:"countryFullName"`
	SPLC            *string `json:"splc"`
}

// Coords struct represents latitude and longitude
type Coords struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
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
	Region          int     `json:"region"`
	POITypeID       int     `json:"poiTypeId"`
	PersistentPOIID int     `json:"persistentPoiId"`
	SiteID          int     `json:"siteId"`
	ResultType      int     `json:"resultType"`
	ShortString     string  `json:"shortString"`
	TimeZone        string  `json:"timeZone"`
	Coords          Coords  `json:"coords"`
	Address         Address `json:"address"`
}

// LocationResponse represents the entire response containing multiple locations
type LocationResponse struct {
	Err       int         `json:"err"`
	Locations []*Location `json:"locations"`
}

type SingleSearchParams struct {
	// The API key for the PCMiler API (Required)
	AuthToken string `json:"authToken" url:"authToken"`

	// String indicating the text to search for (Required)
	Query string `json:"query" url:"query"`

	// Limits search results by the specified number. Must be a value between 1 and 100. (Not Required)
	MaxResults int `json:"maxResults,omitempty" url:"maxResults,omitempty"`

	// 	The current longitude and latitude, where longitude and latitude are either decimal or
	// integer coordinates. (Not Required)
	CurrentLonLat string `json:"currentLonLat,omitempty" url:"currentLonLat,omitempty"`

	// A comma-separated list of InterpTypes to include in the search response.
	// Allowed filters: Country, State, County, City, POBox, Zip, SPLC, Street, RouteNumber,
	//  RouteAlpha, POI, POIStreet, FullPostCode, POIType, CrossStreet, LatLon, CustomPlace, and None.
	// (Example: includeOnly=CustomPlace,Street. This will return only custom places and streets that match
	// with the search string.)
	// To include results from ZIP codes that only apply to post office boxes, the values POBox and Zip must be
	// included in your request. (Not Required)
	IncludeOnly string `json:"includeOnly,omitempty" url:"includeOnly,omitempty"`

	// A comma-separated list of Points of Interest (POI) category names by which you want to filter all POI results. A GET
	// call to /search/poiCategories can be used to retrieve the current list of
	// categories available for filtering. (Not Required)
	PoiCategories string `json:"poiCategories,omitempty" url:"poiCategories,omitempty"`

	// A comma-separated list of country codes by which you want to filter all results. It defaults to ISO format.
	Countries string `json:"countries,omitempty" url:"countries,omitempty"`

	// The standard for country abbreviations: ISO, FIPS, GENC2, and GENC3. (Not Required)
	CountryType string `json:"countryType,omitempty" url:"countryType,omitempty"`

	// A comma-separated list of state abbreviations by which you want to filter all results. (Not Required)
	States string `json:"states,omitempty" url:"states,omitempty"`

	// Set to include=Meta to include additional metadata in your results, such as road grid and link
	// information as well as the confidence level of the search results.
	// (See QueryConfidence in response parameters below.) (Not Required)
	Include string `json:"include,omitempty" url:"include,omitempty"`

	// If set to true, this option includes custom places in the search results where the location’s PlaceId
	// or PlaceName starts with the query string. Note that custom places can only be searched by name and ID,
	// not by address, city, state, or ZIP code. Custom place results will appear before any other search results.
	// This setting is a convenience feature that integrates a call to the Places API’s places/v1/place/search
	// endpoint into the single search call. However, it comes with a performance cost, as single search must
	// wait for the places call to complete before returning results. For optimal performance, consider leaving
	// this option off (false) and making a separate parallel call to the places endpoint.
	//  Default is false. (Not Required)
	UseCustomPlaces bool `json:"useCustomPlaces,omitempty" url:"useCustomPlaces,omitempty"`

	// Sets whether the house number should be returned as a separate field from the rest of the street address.
	// Default is false. (Not Required)
	SeparateHN bool `json:"separateHn,omitempty" url:"separateHn,omitempty"`

	// If set to true, all potential house number ranges will be returned for a particular street match.
	// Default is false. (Not Required)
	GetAllHNRanges bool `json:"getAllHnRanges,omitempty" url:"getAllHnRanges,omitempty"`

	// Set to true to return the TrimblePlaceId, PlaceName and SiteName for a location, if they exist.
	// Default is false. (Not Required)
	IncludeTrimblePlaceIDs bool `json:"includeTrimblePlaceIds,omitempty" url:"includeTrimblePlaceIds,omitempty"`

	// The language to use in results. U.S. English is the default.
	Lang string `json:"lang,omitempty" url:"lang,omitempty"`

	// Used with partial postal code queries. When provided, the center coordinates of multiple postal code
	// points will be calculated and returned. For example, a search for &includeOnly=zip&query=840 will return
	// a long list of ZIP codes starting with “840.” Adding &includeCenter instead returns a single point that is
	// roughly central to all of those ZIP codes.
	IncludeCenter bool `json:"includeCenter,omitempty" url:"includeCenter,omitempty"`

	// 	Limits the amount of difference that is allowed between the input query and the match results. This is generally better left to the default of false, but in some cases where automation is applied to results, it may limit false positives.
	StrictMatch bool `json:"strictMatch,omitempty" url:"strictMatch,omitempty"`

	// Limits search results to within a specified distance in miles from the current location, as identified by currentLonLat. A valid currentLonLat must be sent in the request for this parameter to take effect. In some cases, a short radiusFromCurrentLonLat may produce no matches.
	RadiusFromCurrentLonLat float32 `json:"radiusFromCurrentLonLat,omitempty" url:"radiusFromCurrentLonLat,omitempty"`
}
