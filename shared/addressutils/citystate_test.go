package addressutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCityState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want CityState
		ok   bool
	}{
		"comma and code":     {in: "Toledo, OH", want: CityState{City: "Toledo", State: "OH"}, ok: true},
		"with a postal code": {in: "Toledo, OH 43604", want: CityState{City: "Toledo", State: "OH"}, ok: true},
		"no comma":           {in: "Lansing MI", want: CityState{City: "Lansing", State: "MI"}, ok: true},
		"full state name":    {in: "Columbus, Ohio", want: CityState{City: "Columbus", State: "OH"}, ok: true},
		"two word state":     {in: "Albany New York", want: CityState{City: "Albany", State: "NY"}, ok: true},
		"lower case code":    {in: "louisville, ky", want: CityState{City: "louisville", State: "KY"}, ok: true},
		"zip plus four":      {in: "Grand Rapids, MI 49503-1234", want: CityState{City: "Grand Rapids", State: "MI"}, ok: true},
		"dotted code":        {in: "Dallas, T.X.", want: CityState{City: "Dallas", State: "TX"}, ok: true},
		"no state":           {in: "Toledo"},
		"empty":              {in: "  "},
		"state only":         {in: "OH"},
		"not a state":        {in: "Toledo, XX"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseCityState(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCityStateSameAs(t *testing.T) {
	t.Parallel()

	assert.True(t, CityState{City: "Toledo", State: "OH"}.SameAs(CityState{City: "toledo", State: "OH"}))
	assert.False(t, CityState{City: "Toledo", State: "OH"}.SameAs(CityState{City: "Toledo", State: "WA"}))
}
