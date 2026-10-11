package weatheralert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameMessage(t *testing.T) {
	t.Parallel()

	expires := int64(1_776_276_000)
	sameExpires := expires
	later := expires + 3600
	effective := int64(1_776_254_400)
	stored := &WeatherAlert{
		NWSID:         "urn:oid:1",
		MessageType:   "Alert",
		AlertCategory: AlertCategoryWinterWeather,
		Effective:     &effective,
		Expires:       &expires,
	}

	tests := []struct {
		name     string
		incoming WeatherAlert
		stored   *WeatherAlert
		want     bool
	}{
		{
			name: "same message",
			incoming: WeatherAlert{
				NWSID:         "urn:oid:1",
				MessageType:   "Alert",
				AlertCategory: AlertCategoryWinterWeather,
				Effective:     &effective,
				Expires:       &sameExpires,
				Headline:      "text is not compared",
			},
			stored: stored,
			want:   true,
		},
		{
			name:     "nothing stored",
			incoming: WeatherAlert{NWSID: "urn:oid:1"},
			stored:   nil,
			want:     false,
		},
		{
			name: "expiry moved",
			incoming: WeatherAlert{
				NWSID:         "urn:oid:1",
				MessageType:   "Alert",
				AlertCategory: AlertCategoryWinterWeather,
				Effective:     &effective,
				Expires:       &later,
			},
			stored: stored,
			want:   false,
		},
		{
			name: "cancelled",
			incoming: WeatherAlert{
				NWSID:         "urn:oid:1",
				MessageType:   "Cancel",
				AlertCategory: AlertCategoryWinterWeather,
				Effective:     &effective,
				Expires:       &expires,
			},
			stored: stored,
			want:   false,
		},
		{
			name: "category remapped",
			incoming: WeatherAlert{
				NWSID:         "urn:oid:1",
				MessageType:   "Alert",
				AlertCategory: AlertCategoryOther,
				Effective:     &effective,
				Expires:       &expires,
			},
			stored: stored,
			want:   false,
		},
		{
			name: "onset added",
			incoming: WeatherAlert{
				NWSID:         "urn:oid:1",
				MessageType:   "Alert",
				AlertCategory: AlertCategoryWinterWeather,
				Effective:     &effective,
				Onset:         &effective,
				Expires:       &expires,
			},
			stored: stored,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.incoming.IsSameMessage(tt.stored))
		})
	}
}
