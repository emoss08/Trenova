package geo

import (
	"math"
	"strings"
)

type City struct {
	Name  string
	State string
	Zip   string
	Area  string
	Zone  string
	Lat   float64
	Lon   float64
}

var Cities = []City{
	{"ATLANTA", "GA", "30318", "404", "America/New_York", 33.749, -84.388},
	{"AURORA", "IL", "60504", "630", "America/Chicago", 41.761, -88.320},
	{"BIRMINGHAM", "AL", "35211", "205", "America/Chicago", 33.521, -86.802},
	{"BUFFALO", "NY", "14206", "716", "America/New_York", 42.886, -78.878},
	{"CEDAR RAPIDS", "IA", "52404", "319", "America/Chicago", 41.977, -91.666},
	{"CHARLOTTE", "NC", "28208", "704", "America/New_York", 35.227, -80.843},
	{"CHICAGO", "IL", "60632", "773", "America/Chicago", 41.878, -87.630},
	{"CINCINNATI", "OH", "45215", "513", "America/New_York", 39.103, -84.512},
	{"CLEVELAND", "OH", "44135", "216", "America/New_York", 41.499, -81.694},
	{"COLUMBUS", "OH", "43219", "614", "America/New_York", 39.961, -82.999},
	{"DALLAS", "TX", "75247", "214", "America/Chicago", 32.777, -96.797},
	{"DAVENPORT", "IA", "52806", "563", "America/Chicago", 41.524, -90.578},
	{"DENVER", "CO", "80216", "303", "America/Denver", 39.739, -104.990},
	{"DES MOINES", "IA", "50313", "515", "America/Chicago", 41.587, -93.625},
	{"DETROIT", "MI", "48209", "313", "America/Detroit", 42.331, -83.046},
	{"ELKHART", "IN", "46516", "574", "America/Indiana/Indianapolis", 41.682, -85.976},
	{"FORT WAYNE", "IN", "46803", "260", "America/Indiana/Indianapolis", 41.079, -85.139},
	{"GRAND RAPIDS", "MI", "49548", "616", "America/Detroit", 42.963, -85.668},
	{"GREEN BAY", "WI", "54304", "920", "America/Chicago", 44.519, -88.020},
	{"HOUSTON", "TX", "77029", "713", "America/Chicago", 29.760, -95.370},
	{"INDIANAPOLIS", "IN", "46241", "317", "America/Indiana/Indianapolis", 39.768, -86.158},
	{"JOLIET", "IL", "60436", "815", "America/Chicago", 41.525, -88.082},
	{"KANSAS CITY", "MO", "64120", "816", "America/Chicago", 39.100, -94.579},
	{"LOUISVILLE", "KY", "40213", "502", "America/Kentucky/Louisville", 38.253, -85.759},
	{"MADISON", "WI", "53704", "608", "America/Chicago", 43.073, -89.401},
	{"MEMPHIS", "TN", "38118", "901", "America/Chicago", 35.149, -90.049},
	{"MILWAUKEE", "WI", "53204", "414", "America/Chicago", 43.039, -87.906},
	{"MINNEAPOLIS", "MN", "55413", "612", "America/Chicago", 44.978, -93.265},
	{"NASHVILLE", "TN", "37210", "615", "America/Chicago", 36.163, -86.781},
	{"OMAHA", "NE", "68117", "402", "America/Chicago", 41.257, -95.935},
	{"PEORIA", "IL", "61604", "309", "America/Chicago", 40.694, -89.589},
	{"PITTSBURGH", "PA", "15205", "412", "America/New_York", 40.441, -79.996},
	{"ROCKFORD", "IL", "61109", "815", "America/Chicago", 42.271, -89.094},
	{"SAINT LOUIS", "MO", "63147", "314", "America/Chicago", 38.627, -90.199},
	{"SOUTH BEND", "IN", "46628", "574", "America/Indiana/Indianapolis", 41.676, -86.252},
	{"SPRINGFIELD", "IL", "62703", "217", "America/Chicago", 39.781, -89.650},
	{"TOLEDO", "OH", "43607", "419", "America/New_York", 41.654, -83.537},
	{"TULSA", "OK", "74107", "918", "America/Chicago", 36.154, -95.993},
	{"WICHITA", "KS", "67213", "316", "America/Chicago", 37.687, -97.330},
	{"YORK", "PA", "17406", "717", "America/New_York", 39.963, -76.728},
}

var StreetNames = []string{
	"INDUSTRIAL", "COMMERCE", "DISTRIBUTION", "ENTERPRISE", "LOGISTICS", "FOUNDRY", "MILL",
	"RAILROAD", "AIRPORT", "WAREHOUSE", "CENTRAL", "MARKET", "RIVER", "LAKE", "PRAIRIE",
	"MERIDIAN", "OAK", "MAPLE", "CEDAR", "HARBOR", "GATEWAY", "PIONEER", "TECHNOLOGY",
}

var StreetKinds = []string{"DR", "RD", "AVE", "PKWY", "BLVD", "ST", "CT", "WAY"}

func Find(name, state string) (City, bool) {
	for _, candidate := range Cities {
		if strings.EqualFold(candidate.Name, name) && strings.EqualFold(candidate.State, state) {
			return candidate, true
		}
	}
	return City{}, false
}

func RoadMiles(from, to City) int {
	const earthMiles = 3958.8
	const roadFactor = 1.18
	lat1 := from.Lat * math.Pi / 180
	lat2 := to.Lat * math.Pi / 180
	dLat := lat2 - lat1
	dLon := (to.Lon - from.Lon) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	miles := 2 * earthMiles * math.Asin(math.Sqrt(h)) * roadFactor
	return max(int(math.Round(miles)), 10)
}
