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
	}
	for in, want := range cases {
		assert.Equal(t, want, HumanizeCamelCaseSentence(in), in)
	}
}
