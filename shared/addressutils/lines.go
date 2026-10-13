package addressutils

import (
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

// Mailing is a postal address as a document prints it.
type Mailing struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// Lines is the address one line per row, blank parts left out:
// street, suite, "City, ST 12345", country.
func (m Mailing) Lines() []string {
	return stringutils.FilterEmpty([]string{
		strings.TrimSpace(m.Line1),
		strings.TrimSpace(m.Line2),
		CityStatePostal(m.City, m.State, m.PostalCode),
		strings.TrimSpace(m.Country),
	})
}

// CityStatePostal joins a city with its state and postal code the way an
// envelope reads, "Dallas, TX 75201", leaving out whatever is blank.
func CityStatePostal(city, state, postalCode string) string {
	left := strings.TrimSpace(city)
	statePostal := strings.TrimSpace(
		strings.Join(stringutils.FilterEmpty([]string{
			strings.TrimSpace(state),
			strings.TrimSpace(postalCode),
		}), " "),
	)
	switch {
	case left == "":
		return statePostal
	case statePostal == "":
		return left
	default:
		return left + ", " + statePostal
	}
}
