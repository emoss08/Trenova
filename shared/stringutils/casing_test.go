package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanizeCamelCaseSentence(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                   "",
		"windowDays":         "Window days",
		"shipmentId":         "Shipment ID",
		"profileId":          "Profile ID",
		"credentialTypeCode": "Credential type code",
		"message":            "Message",
		"dotNumber":          "DOT number",
		"lastMvrCheck":       "Last MVR check",
		"cdlClass":           "CDL class",
		"mcNumber":           "MC number",
		"unNumber":           "UN number",
		"iftaReturnId":       "IFTA return ID",
	}
	for in, want := range cases {
		assert.Equal(t, want, HumanizeCamelCaseSentence(in), in)
	}
}

func TestLowerFirst(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                  "",
		"Send the invoice.": "send the invoice.",
		"already lower":     "already lower",
		"Ünited":            "ünited",
	}
	for in, want := range cases {
		if got := LowerFirst(in); got != want {
			t.Errorf("LowerFirst(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLowerCamel(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                "",
		"Piece Count":     "pieceCount",
		"Pickup Stop":     "pickupStop",
		"PO number":       "poNumber",
		"  bill-to / ref": "billToRef",
		"Weight":          "weight",
	}
	for in, want := range cases {
		assert.Equal(t, want, LowerCamel(in), in)
	}
}
