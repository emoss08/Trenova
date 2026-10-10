package aitraining

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"unicode"
)

var (
	companyAdjectives = []string{
		"Amber", "Arrow", "Bluestone", "Brightline", "Cascade", "Cedar", "Clearwater", "Copper",
		"Crescent", "Eastgate", "Evergreen", "Falcon", "Granite", "Harbor", "Highland", "Ironwood",
		"Juniper", "Keystone", "Lakeside", "Meadow", "Northwind", "Oakridge", "Pinecrest", "Prairie",
		"Redwood", "Ridgeline", "Riverbend", "Sagebrush", "Silverline", "Stonebridge", "Summit",
		"Timberline", "Westfield", "Willow",
	}
	companyNouns = []string{
		"Supply", "Foods", "Logistics", "Freight", "Distribution", "Industries", "Materials",
		"Manufacturing", "Trading", "Products", "Packaging", "Farms", "Brands", "Provisions",
		"Warehousing", "Transport", "Partners", "Goods",
	}
	firstNames = []string{
		"Alex", "Avery", "Blake", "Cameron", "Casey", "Dakota", "Drew", "Emerson", "Finley",
		"Harper", "Hayden", "Jamie", "Jordan", "Kai", "Logan", "Morgan", "Parker", "Quinn",
		"Reese", "Riley", "Rowan", "Sage", "Skyler", "Taylor",
	}
	lastNames = []string{
		"Abbott", "Barrett", "Carver", "Dalton", "Ellison", "Fletcher", "Garrison", "Hale",
		"Irwin", "Jennings", "Keller", "Langley", "Mercer", "Nolan", "Orton", "Prescott",
		"Quimby", "Ramsey", "Sutton", "Thatcher", "Underwood", "Vance", "Whitaker", "Yates",
	}
	streetNames = []string{
		"Ashford", "Birch", "Brookside", "Canyon", "Chestnut", "Cypress", "Dogwood",
		"Elm", "Fairway", "Foxhollow", "Glenview", "Hawthorn", "Hickory", "Industrial", "Juniper",
		"Laurel", "Linden", "Magnolia", "Maple", "Orchard", "Poplar", "Quarry",
		"Railway", "Sycamore", "Tanner", "Walnut",
	}
	streetSuffixes = []string{"St", "Ave", "Rd", "Blvd", "Dr", "Ln", "Way", "Pkwy", "Ct"}
	cityNames      = []string{
		"Ashbury", "Bellmont", "Briarwood", "Brookhaven", "Cedarville", "Clearfield", "Crestview",
		"Dunmore", "Eastbrook", "Elmwood", "Fairhaven", "Glendale", "Greenfield", "Hartwell",
		"Hillsdale", "Kingsford", "Lakeview", "Linwood", "Maplewood", "Millbrook", "Northfield",
		"Oakdale", "Pinehurst", "Ridgeway", "Riverton", "Rosewood", "Southport", "Stonybrook",
		"Westbury", "Woodhaven",
	}
)

func pick(rng *rand.Rand, values []string) string {
	return values[rng.IntN(len(values))]
}

func companySurrogate(rng *rand.Rand) string {
	return pick(rng, companyAdjectives) + " " + pick(rng, companyNouns)
}

func personSurrogate(rng *rand.Rand) string {
	return pick(rng, firstNames) + " " + pick(rng, lastNames)
}

func streetSurrogate(rng *rand.Rand) string {
	return fmt.Sprintf("%d %s %s", 100+rng.IntN(9800), pick(rng, streetNames), pick(rng, streetSuffixes))
}

func citySurrogate(rng *rand.Rand) string {
	return pick(rng, cityNames)
}

func postalSurrogate(rng *rand.Rand) string {
	return fmt.Sprintf("%05d", 10000+rng.IntN(89999))
}

func emailSurrogate(rng *rand.Rand) string {
	return fmt.Sprintf(
		"%s.%s%d@example.com",
		strings.ToLower(pick(rng, firstNames)),
		strings.ToLower(pick(rng, lastNames)),
		rng.IntN(100),
	)
}

func shapeSurrogate(rng *rand.Rand, original string) string {
	out := make([]rune, 0, len(original))
	for _, r := range original {
		switch {
		case r >= '0' && r <= '9':
			out = append(out, rune('0'+rng.IntN(10)))
		case unicode.IsUpper(r):
			out = append(out, rune('A'+rng.IntN(26)))
		case unicode.IsLower(r):
			out = append(out, rune('a'+rng.IntN(26)))
		default:
			out = append(out, r)
		}
	}

	return string(out)
}

func reshape(original, surrogate string) string {
	source := []rune(surrogate)
	out := make([]rune, 0, len(original))
	next := 0
	for _, r := range original {
		if !isWordRune(r) {
			out = append(out, r)
			continue
		}
		if next >= len(source) {
			out = append(out, r)
			continue
		}
		replacement := source[next]
		next++
		switch {
		case unicode.IsLower(r):
			replacement = unicode.ToLower(replacement)
		case unicode.IsUpper(r):
			replacement = unicode.ToUpper(replacement)
		}
		out = append(out, replacement)
	}

	return string(out)
}

func styleLike(surrogate, original string) string {
	hasUpper, hasLower := false, false
	for _, r := range original {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		}
	}

	switch {
	case hasUpper && !hasLower:
		return strings.ToUpper(surrogate)
	case hasLower && !hasUpper:
		return strings.ToLower(surrogate)
	default:
		return surrogate
	}
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
