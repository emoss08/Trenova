package volume

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/shopspring/decimal"
)

type faker struct {
	r *rand.Rand
}

func newFaker(seed uint64) *faker {
	return &faker{r: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))} //nolint:gosec // seed data
}

func (f *faker) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return f.r.IntN(n)
}

func (f *faker) between(low, high int) int {
	if high <= low {
		return low
	}
	return low + f.r.IntN(high-low+1)
}

func (f *faker) between64(low, high int64) int64 {
	if high <= low {
		return low
	}
	return low + f.r.Int64N(high-low+1)
}

func (f *faker) chance(p float64) bool {
	return f.r.Float64() < p
}

func (f *faker) float() float64 {
	return f.r.Float64()
}

func (f *faker) weighted(weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	roll := f.intn(total)
	for i, w := range weights {
		if roll < w {
			return i
		}
		roll -= w
	}
	return len(weights) - 1
}

func pick[T any](f *faker, items []T) T {
	return items[f.intn(len(items))]
}

func (f *faker) digits(n int) string {
	var b strings.Builder
	b.Grow(n)
	for range n {
		b.WriteByte(byte('0' + f.intn(10)))
	}
	return b.String()
}

func (f *faker) letters(n int) string {
	var b strings.Builder
	b.Grow(n)
	for range n {
		b.WriteByte(byte('A' + f.intn(26)))
	}
	return b.String()
}

func (f *faker) money(low, high float64) decimal.Decimal {
	return decimal.NewFromFloat(low + f.r.Float64()*(high-low)).Round(2)
}

func (f *faker) firstName() string {
	return pick(f, firstNames)
}

func (f *faker) lastName() string {
	return pick(f, lastNames)
}

func (f *faker) phone() string {
	return fmt.Sprintf("+1%d%s%s", f.between(201, 989), f.digits(3), f.digits(4))
}

func (f *faker) street() string {
	return fmt.Sprintf("%d %s %s", f.between(100, 19999), pick(f, streetNames), pick(f, streetSuffixes))
}

func (f *faker) city() *cityRecord {
	idx := f.weighted(cityWeights)
	return &cities[idx]
}

func (f *faker) cityNear(home *cityRecord, radiusMiles float64) *cityRecord {
	for range 12 {
		candidate := f.city()
		if candidate != home && haversineMiles(home, candidate) <= radiusMiles {
			return candidate
		}
	}
	return f.city()
}

func (f *faker) postalCode(c *cityRecord) string {
	return c.Zip[:3] + f.digits(2)
}

func (f *faker) companyName() string {
	switch f.intn(4) {
	case 0:
		return fmt.Sprintf("%s %s", pick(f, companyPrefixes), pick(f, companyIndustries))
	case 1:
		return fmt.Sprintf("%s %s %s", f.lastName(), pick(f, companyIndustries), pick(f, companySuffixes))
	case 2:
		return fmt.Sprintf("%s %s", pick(f, companyPrefixes), pick(f, companySuffixes))
	default:
		return fmt.Sprintf("%s & %s %s", f.lastName(), f.lastName(), pick(f, companySuffixes))
	}
}

func (f *faker) vin() string {
	const alphabet = "ABCDEFGHJKLMNPRSTUVWXYZ0123456789"
	var b strings.Builder
	b.Grow(17)
	for range 17 {
		b.WriteByte(alphabet[f.intn(len(alphabet))])
	}
	return b.String()
}

func (f *faker) plate() string {
	return f.letters(3) + f.digits(4)
}

func (f *faker) jitter(c *cityRecord) (lat, lon float64) {
	return c.Lat + (f.float()-0.5)*0.18, c.Lon + (f.float()-0.5)*0.18
}

func codeFrom(name string, size int) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		}
		if b.Len() == size {
			break
		}
	}
	for b.Len() < size {
		b.WriteByte('X')
	}
	return b.String()
}

func uniqueCode(base string, taken map[string]struct{}) string {
	candidate := base
	for n := 2; ; n++ {
		key := strings.ToLower(candidate)
		if _, ok := taken[key]; !ok {
			taken[key] = struct{}{}
			return candidate
		}
		candidate = fmt.Sprintf("%s%d", base, n)
	}
}

func haversineMiles(a, b *cityRecord) float64 {
	const earthRadiusMiles = 3958.8
	lat1, lat2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMiles * math.Asin(math.Sqrt(h))
}

func roadMiles(a, b *cityRecord) float64 {
	miles := haversineMiles(a, b) * 1.18
	if miles < 12 {
		miles = 12 + miles
	}
	return math.Round(miles*10) / 10
}

type cityRecord struct {
	Name     string
	State    string
	Lat      float64
	Lon      float64
	Zip      string
	Timezone string
	Weight   int
}

var cityWeights = func() []int {
	weights := make([]int, len(cities))
	for i := range cities {
		weights[i] = cities[i].Weight
	}
	return weights
}()

const (
	tzEastern  = "America/New_York"
	tzCentral  = "America/Chicago"
	tzMountain = "America/Denver"
	tzArizona  = "America/Phoenix"
	tzPacific  = "America/Los_Angeles"
)

var cities = []cityRecord{
	{"Atlanta", "GA", 33.749, -84.388, "30303", tzEastern, 9},
	{"Savannah", "GA", 32.0809, -81.0912, "31401", tzEastern, 5},
	{"Macon", "GA", 32.8407, -83.6324, "31201", tzEastern, 2},
	{"Charlotte", "NC", 35.2271, -80.8431, "28202", tzEastern, 7},
	{"Greensboro", "NC", 36.0726, -79.792, "27401", tzEastern, 3},
	{"Raleigh", "NC", 35.7796, -78.6382, "27601", tzEastern, 3},
	{"Charleston", "SC", 32.7765, -79.9311, "29401", tzEastern, 3},
	{"Greenville", "SC", 34.8526, -82.394, "29601", tzEastern, 3},
	{"Columbia", "SC", 34.0007, -81.0348, "29201", tzEastern, 2},
	{"Jacksonville", "FL", 30.3322, -81.6557, "32202", tzEastern, 5},
	{"Miami", "FL", 25.7617, -80.1918, "33101", tzEastern, 5},
	{"Orlando", "FL", 28.5383, -81.3792, "32801", tzEastern, 5},
	{"Tampa", "FL", 27.9506, -82.4572, "33602", tzEastern, 4},
	{"Lakeland", "FL", 28.0395, -81.9498, "33801", tzEastern, 3},
	{"Nashville", "TN", 36.1627, -86.7816, "37203", tzCentral, 6},
	{"Memphis", "TN", 35.1495, -90.049, "38103", tzCentral, 8},
	{"Knoxville", "TN", 35.9606, -83.9207, "37902", tzEastern, 3},
	{"Chattanooga", "TN", 35.0456, -85.3097, "37402", tzEastern, 3},
	{"Louisville", "KY", 38.2527, -85.7585, "40202", tzEastern, 5},
	{"Lexington", "KY", 38.0406, -84.5037, "40507", tzEastern, 2},
	{"Birmingham", "AL", 33.5186, -86.8104, "35203", tzCentral, 4},
	{"Mobile", "AL", 30.6954, -88.0399, "36602", tzCentral, 2},
	{"Huntsville", "AL", 34.7304, -86.5861, "35801", tzCentral, 2},
	{"Jackson", "MS", 32.2988, -90.1848, "39201", tzCentral, 2},
	{"New Orleans", "LA", 29.9511, -90.0715, "70112", tzCentral, 4},
	{"Baton Rouge", "LA", 30.4515, -91.1871, "70801", tzCentral, 3},
	{"Shreveport", "LA", 32.5252, -93.7502, "71101", tzCentral, 2},
	{"Little Rock", "AR", 34.7465, -92.2896, "72201", tzCentral, 3},
	{"Fort Smith", "AR", 35.3859, -94.3985, "72901", tzCentral, 2},
	{"Springdale", "AR", 36.1867, -94.1288, "72764", tzCentral, 3},
	{"Dallas", "TX", 32.7767, -96.797, "75201", tzCentral, 10},
	{"Fort Worth", "TX", 32.7555, -97.3308, "76102", tzCentral, 6},
	{"Houston", "TX", 29.7604, -95.3698, "77002", tzCentral, 10},
	{"San Antonio", "TX", 29.4241, -98.4936, "78205", tzCentral, 5},
	{"Austin", "TX", 30.2672, -97.7431, "78701", tzCentral, 4},
	{"El Paso", "TX", 31.7619, -106.485, "79901", tzMountain, 4},
	{"Laredo", "TX", 27.5306, -99.4803, "78040", tzCentral, 5},
	{"Amarillo", "TX", 35.222, -101.8313, "79101", tzCentral, 2},
	{"Lubbock", "TX", 33.5779, -101.8552, "79401", tzCentral, 2},
	{"McAllen", "TX", 26.2034, -98.23, "78501", tzCentral, 3},
	{"Corpus Christi", "TX", 27.8006, -97.3964, "78401", tzCentral, 2},
	{"Oklahoma City", "OK", 35.4676, -97.5164, "73102", tzCentral, 4},
	{"Tulsa", "OK", 36.154, -95.9928, "74103", tzCentral, 3},
	{"Kansas City", "MO", 39.0997, -94.5786, "64105", tzCentral, 6},
	{"St. Louis", "MO", 38.627, -90.1994, "63101", tzCentral, 6},
	{"Springfield", "MO", 37.209, -93.2923, "65806", tzCentral, 2},
	{"Wichita", "KS", 37.6872, -97.3301, "67202", tzCentral, 2},
	{"Omaha", "NE", 41.2565, -95.9345, "68102", tzCentral, 3},
	{"Lincoln", "NE", 40.8136, -96.7026, "68508", tzCentral, 2},
	{"Des Moines", "IA", 41.5868, -93.625, "50309", tzCentral, 3},
	{"Cedar Rapids", "IA", 41.9779, -91.6656, "52401", tzCentral, 2},
	{"Minneapolis", "MN", 44.9778, -93.265, "55401", tzCentral, 5},
	{"St. Paul", "MN", 44.9537, -93.09, "55101", tzCentral, 2},
	{"Fargo", "ND", 46.8772, -96.7898, "58102", tzCentral, 1},
	{"Sioux Falls", "SD", 43.5446, -96.7311, "57104", tzCentral, 1},
	{"Milwaukee", "WI", 43.0389, -87.9065, "53202", tzCentral, 4},
	{"Green Bay", "WI", 44.5133, -88.0133, "54301", tzCentral, 2},
	{"Madison", "WI", 43.0731, -89.4012, "53703", tzCentral, 2},
	{"Chicago", "IL", 41.8781, -87.6298, "60601", tzCentral, 10},
	{"Joliet", "IL", 41.525, -88.0817, "60432", tzCentral, 5},
	{"Rockford", "IL", 42.2711, -89.094, "61101", tzCentral, 2},
	{"Peoria", "IL", 40.6936, -89.589, "61602", tzCentral, 2},
	{"Indianapolis", "IN", 39.7684, -86.1581, "46204", tzEastern, 7},
	{"Fort Wayne", "IN", 41.0793, -85.1394, "46802", tzEastern, 2},
	{"Columbus", "OH", 39.9612, -82.9988, "43215", tzEastern, 6},
	{"Cincinnati", "OH", 39.1031, -84.512, "45202", tzEastern, 5},
	{"Cleveland", "OH", 41.4993, -81.6944, "44113", tzEastern, 5},
	{"Toledo", "OH", 41.6528, -83.5379, "43604", tzEastern, 3},
	{"Dayton", "OH", 39.7589, -84.1916, "45402", tzEastern, 2},
	{"Detroit", "MI", 42.3314, -83.0458, "48226", tzEastern, 6},
	{"Grand Rapids", "MI", 42.9634, -85.6681, "49503", tzEastern, 3},
	{"Lansing", "MI", 42.7325, -84.5555, "48933", tzEastern, 2},
	{"Pittsburgh", "PA", 40.4406, -79.9959, "15222", tzEastern, 4},
	{"Philadelphia", "PA", 39.9526, -75.1652, "19103", tzEastern, 6},
	{"Allentown", "PA", 40.6084, -75.4902, "18101", tzEastern, 5},
	{"Harrisburg", "PA", 40.2732, -76.8867, "17101", tzEastern, 4},
	{"Scranton", "PA", 41.4089, -75.6624, "18503", tzEastern, 2},
	{"Newark", "NJ", 40.7357, -74.1724, "07102", tzEastern, 6},
	{"Edison", "NJ", 40.5187, -74.4121, "08817", tzEastern, 4},
	{"Secaucus", "NJ", 40.7895, -74.0565, "07094", tzEastern, 3},
	{"New York", "NY", 40.7128, -74.006, "10001", tzEastern, 5},
	{"Buffalo", "NY", 42.8864, -78.8784, "14202", tzEastern, 3},
	{"Syracuse", "NY", 43.0481, -76.1474, "13202", tzEastern, 2},
	{"Albany", "NY", 42.6526, -73.7562, "12207", tzEastern, 2},
	{"Rochester", "NY", 43.1566, -77.6088, "14604", tzEastern, 2},
	{"Boston", "MA", 42.3601, -71.0589, "02108", tzEastern, 4},
	{"Worcester", "MA", 42.2626, -71.8023, "01608", tzEastern, 2},
	{"Hartford", "CT", 41.7658, -72.6734, "06103", tzEastern, 2},
	{"Providence", "RI", 41.824, -71.4128, "02903", tzEastern, 1},
	{"Portland", "ME", 43.6591, -70.2568, "04101", tzEastern, 1},
	{"Manchester", "NH", 42.9956, -71.4548, "03101", tzEastern, 1},
	{"Baltimore", "MD", 39.2904, -76.6122, "21202", tzEastern, 5},
	{"Hagerstown", "MD", 39.6418, -77.72, "21740", tzEastern, 2},
	{"Wilmington", "DE", 39.7391, -75.5398, "19801", tzEastern, 1},
	{"Richmond", "VA", 37.5407, -77.436, "23219", tzEastern, 3},
	{"Norfolk", "VA", 36.8508, -76.2859, "23510", tzEastern, 4},
	{"Roanoke", "VA", 37.271, -79.9414, "24011", tzEastern, 2},
	{"Charleston", "WV", 38.3498, -81.6326, "25301", tzEastern, 1},
	{"Denver", "CO", 39.7392, -104.9903, "80202", tzMountain, 5},
	{"Colorado Springs", "CO", 38.8339, -104.8214, "80903", tzMountain, 2},
	{"Salt Lake City", "UT", 40.7608, -111.891, "84101", tzMountain, 4},
	{"Ogden", "UT", 41.223, -111.9738, "84401", tzMountain, 2},
	{"Boise", "ID", 43.615, -116.2023, "83702", tzMountain, 2},
	{"Billings", "MT", 45.7833, -108.5007, "59101", tzMountain, 1},
	{"Cheyenne", "WY", 41.14, -104.8202, "82001", tzMountain, 1},
	{"Albuquerque", "NM", 35.0844, -106.6504, "87102", tzMountain, 2},
	{"Phoenix", "AZ", 33.4484, -112.074, "85004", tzArizona, 6},
	{"Tucson", "AZ", 32.2226, -110.9747, "85701", tzArizona, 2},
	{"Las Vegas", "NV", 36.1699, -115.1398, "89101", tzPacific, 3},
	{"Reno", "NV", 39.5296, -119.8138, "89501", tzPacific, 3},
	{"Los Angeles", "CA", 34.0522, -118.2437, "90012", tzPacific, 10},
	{"Long Beach", "CA", 33.7701, -118.1937, "90802", tzPacific, 6},
	{"Ontario", "CA", 34.0633, -117.6509, "91761", tzPacific, 7},
	{"Riverside", "CA", 33.9806, -117.3755, "92501", tzPacific, 4},
	{"San Diego", "CA", 32.7157, -117.1611, "92101", tzPacific, 3},
	{"Fresno", "CA", 36.7378, -119.7871, "93721", tzPacific, 3},
	{"Bakersfield", "CA", 35.3733, -119.0187, "93301", tzPacific, 2},
	{"Stockton", "CA", 37.9577, -121.2908, "95202", tzPacific, 4},
	{"Sacramento", "CA", 38.5816, -121.4944, "95814", tzPacific, 3},
	{"Oakland", "CA", 37.8044, -122.2712, "94607", tzPacific, 4},
	{"San Jose", "CA", 37.3382, -121.8863, "95113", tzPacific, 2},
	{"Portland", "OR", 45.5152, -122.6784, "97204", tzPacific, 4},
	{"Eugene", "OR", 44.0521, -123.0868, "97401", tzPacific, 1},
	{"Seattle", "WA", 47.6062, -122.3321, "98101", tzPacific, 5},
	{"Tacoma", "WA", 47.2529, -122.4443, "98402", tzPacific, 4},
	{"Spokane", "WA", 47.6588, -117.426, "99201", tzPacific, 2},
	{"Yakima", "WA", 46.6021, -120.5059, "98901", tzPacific, 1},
}

var firstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda", "David",
	"Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas",
	"Karen", "Christopher", "Sarah", "Charles", "Lisa", "Daniel", "Nancy", "Matthew",
	"Sandra", "Anthony", "Betty", "Mark", "Ashley", "Donald", "Emily", "Steven", "Kimberly",
	"Andrew", "Margaret", "Paul", "Donna", "Joshua", "Michelle", "Kenneth", "Carol", "Kevin",
	"Amanda", "Brian", "Melissa", "Timothy", "Deborah", "Ronald", "Stephanie", "Jason",
	"Rebecca", "George", "Sharon", "Edward", "Laura", "Jeffrey", "Cynthia", "Ryan", "Amy",
	"Jacob", "Kathleen", "Nicholas", "Angela", "Gary", "Shirley", "Eric", "Brenda",
	"Jonathan", "Emma", "Stephen", "Anna", "Larry", "Pamela", "Justin", "Nicole", "Scott",
	"Samantha", "Brandon", "Katherine", "Benjamin", "Christine", "Samuel", "Debra",
	"Gregory", "Rachel", "Alexander", "Carolyn", "Patrick", "Janet", "Frank", "Maria",
	"Raymond", "Olivia", "Jack", "Heather", "Dennis", "Helen", "Jerry", "Catherine",
	"Tyler", "Diane", "Aaron", "Julie", "Jose", "Victoria", "Adam", "Joyce", "Nathan",
	"Lauren", "Henry", "Kelly", "Zachary", "Christina", "Douglas", "Ruth", "Peter", "Joan",
	"Kyle", "Virginia", "Noah", "Judith", "Ethan", "Evelyn", "Jeremy", "Hannah", "Walter",
	"Andrea", "Christian", "Megan", "Keith", "Cheryl", "Roger", "Jacqueline", "Terry",
	"Martha", "Austin", "Madison", "Sean", "Teresa", "Gerald", "Gloria", "Carl", "Sara",
	"Harold", "Janice", "Dylan", "Ann", "Arthur", "Kathryn", "Lawrence", "Abigail",
	"Jordan", "Sophia", "Jesse", "Frances", "Bryan", "Jean", "Billy", "Alice", "Bruce",
	"Judy", "Gabriel", "Isabella", "Joe", "Julia", "Logan", "Grace", "Alan", "Amber",
	"Juan", "Denise", "Albert", "Danielle", "Willie", "Marilyn", "Elijah", "Beverly",
	"Wayne", "Charlotte", "Randy", "Natalie", "Vincent", "Theresa", "Mason", "Diana",
	"Roy", "Brittany", "Ralph", "Doris", "Bobby", "Kayla", "Russell", "Alexis", "Bradley",
	"Lori", "Philip", "Marie", "Eugene", "Luis", "Carlos", "Miguel", "Jorge", "Rosa",
	"Ana", "Luz", "Marco", "Dmitri", "Ivan", "Andrei", "Wei", "Mei", "Hiroshi", "Priya",
	"Rahul", "Amir", "Fatima", "Omar", "Kwame", "Amara", "Tomasz", "Katarzyna",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
	"Thomas", "Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores", "Green",
	"Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter",
	"Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker", "Cruz", "Edwards",
	"Collins", "Reyes", "Stewart", "Morris", "Morales", "Murphy", "Cook", "Rogers",
	"Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson", "Bailey", "Reed", "Kelly",
	"Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson", "Watson", "Brooks", "Chavez",
	"Wood", "James", "Bennett", "Gray", "Mendoza", "Ruiz", "Hughes", "Price", "Alvarez",
	"Castillo", "Sanders", "Patel", "Myers", "Long", "Ross", "Foster", "Jimenez", "Powell",
	"Jenkins", "Perry", "Russell", "Sullivan", "Bell", "Coleman", "Butler", "Henderson",
	"Barnes", "Gonzales", "Fisher", "Vasquez", "Simmons", "Romero", "Jordan", "Patterson",
	"Alexander", "Hamilton", "Graham", "Reynolds", "Griffin", "Wallace", "Moreno", "West",
	"Cole", "Hayes", "Bryant", "Herrera", "Gibson", "Ellis", "Tran", "Medina", "Aguilar",
	"Stevens", "Murray", "Ford", "Castro", "Marshall", "Owens", "Harrison", "Fernandez",
	"McDonald", "Woods", "Washington", "Kennedy", "Wells", "Vargas", "Henry", "Chen",
	"Freeman", "Webb", "Tucker", "Guzman", "Burns", "Crawford", "Olson", "Simpson",
	"Porter", "Hunter", "Gordon", "Mendez", "Silva", "Shaw", "Snyder", "Mason", "Dixon",
	"Munoz", "Hunt", "Hicks", "Holmes", "Palmer", "Wagner", "Black", "Robertson", "Boyd",
	"Rose", "Stone", "Salazar", "Fox", "Warren", "Mills", "Meyer", "Rice", "Schmidt",
	"Garza", "Daniels", "Ferguson", "Nichols", "Stephens", "Soto", "Weaver", "Ryan",
	"Gardner", "Payne", "Grant", "Dunn", "Kowalski", "Novak", "Ivanov", "Okafor", "Mensah",
	"Singh", "Sharma", "Tanaka", "Nakamura", "Haddad", "Khalil", "Schultz", "Becker",
}

var streetNames = []string{
	"Industrial", "Commerce", "Distribution", "Logistics", "Enterprise", "Freight",
	"Warehouse", "Terminal", "Gateway", "Corporate", "Trade Center", "Port", "Rail",
	"Interstate", "Airport", "Harbor", "Mill", "Factory", "Market", "Union", "Main",
	"Oak", "Maple", "Cedar", "Pine", "Elm", "Washington", "Lincoln", "Jefferson",
	"Madison", "Franklin", "Jackson", "Highland", "Valley", "Ridge", "Lakeview",
	"Riverside", "Park", "Center", "Commercial", "Progress", "Production", "Supply",
	"Crossroads", "Midway", "Northpoint", "Southpark", "Westgate", "Eastside", "Prairie",
}

var streetSuffixes = []string{
	"Pkwy", "Blvd", "Dr", "Rd", "Ave", "St", "Way", "Ln", "Ct", "Hwy", "Loop", "Pl",
}

var companyPrefixes = []string{
	"Apex", "Summit", "Pioneer", "Heartland", "Coastal", "Frontier", "Liberty", "Keystone",
	"Evergreen", "Pinnacle", "Sterling", "Granite", "Horizon", "Northstar", "Bluewater",
	"Redwood", "Ironwood", "Silverline", "Prairie", "Lakeshore", "Riverbend", "Sunbelt",
	"Midwest", "Great Plains", "Atlantic", "Pacific", "Gulfstream", "Cascade", "Sierra",
	"Blue Ridge", "Tri-State", "Continental", "American", "United", "National", "Allied",
	"Premier", "Superior", "Crown", "Eagle", "Falcon", "Patriot", "Vanguard", "Titan",
	"Cornerstone", "Crossroads", "Harvest", "Golden Valley", "Clearwater", "Stonebridge",
}

var companyIndustries = []string{
	"Foods", "Beverage", "Paper", "Packaging", "Plastics", "Steel", "Metals", "Lumber",
	"Building Supply", "Chemicals", "Automotive", "Appliance", "Furniture", "Textiles",
	"Electronics", "Farms", "Produce", "Dairy", "Meats", "Grain", "Pharmaceuticals",
	"Home Goods", "Hardware", "Tire", "Glass", "Brewing", "Bottling", "Pet Supply",
	"Medical Supply", "Outdoor", "Sporting Goods", "Cabinetry", "Flooring", "Solar",
	"Batteries", "Pipe & Supply", "Fertilizer", "Feed", "Snacks", "Frozen Foods",
}

var companySuffixes = []string{
	"Inc", "LLC", "Co", "Corp", "Group", "Industries", "Holdings", "Partners",
	"Distribution", "Manufacturing", "Supply Co", "Enterprises", "Brands", "International",
}
