package carrier

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/stretchr/testify/assert"
)

var (
	textFormatA = dataentrycontrol.TextFormat{
		CodeCase:  dataentrycontrol.CaseFormatUpper,
		NameCase:  dataentrycontrol.CaseFormatTitleCase,
		EmailCase: dataentrycontrol.CaseFormatLower,
		CityCase:  dataentrycontrol.CaseFormatAsEntered,
	}
	textFormatB = dataentrycontrol.TextFormat{
		CodeCase:  dataentrycontrol.CaseFormatLower,
		NameCase:  dataentrycontrol.CaseFormatUpper,
		EmailCase: dataentrycontrol.CaseFormatUpper,
		CityCase:  dataentrycontrol.CaseFormatTitleCase,
	}
)

func TestApplyTextFormat(t *testing.T) {
	t.Parallel()

	newCarrier := func() (*Carrier, *CarrierContact) {
		contact := &CarrierContact{Name: "jane doe", Email: "Jane@Swift.COM"}
		return &Carrier{
			Code:        "Sw Ft",
			Name:        "swift transport",
			DBAName:     "swift",
			RemitToName: "swift remittance",
			City:        "phoenix",
			RemitCity:   "tempe",
			Email:       "Dispatch@Swift.COM",
			Contacts:    []*CarrierContact{contact, nil},
		}, contact
	}

	a, aContact := newCarrier()
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "SWFT", a.Code)
	assert.Equal(t, "Swift Transport", a.Name)
	assert.Equal(t, "Swift", a.DBAName)
	assert.Equal(t, "Swift Remittance", a.RemitToName)
	assert.Equal(t, "phoenix", a.City)
	assert.Equal(t, "tempe", a.RemitCity)
	assert.Equal(t, "dispatch@swift.com", a.Email)
	assert.Equal(t, "Jane Doe", aContact.Name)
	assert.Equal(t, "jane@swift.com", aContact.Email)

	b, bContact := newCarrier()
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "swft", b.Code)
	assert.Equal(t, "SWIFT TRANSPORT", b.Name)
	assert.Equal(t, "SWIFT", b.DBAName)
	assert.Equal(t, "SWIFT REMITTANCE", b.RemitToName)
	assert.Equal(t, "Phoenix", b.City)
	assert.Equal(t, "Tempe", b.RemitCity)
	assert.Equal(t, "DISPATCH@SWIFT.COM", b.Email)
	assert.Equal(t, "JANE DOE", bContact.Name)
	assert.Equal(t, "JANE@SWIFT.COM", bContact.Email)
}

func TestApplyTextFormatAsEnteredKeepsCase(t *testing.T) {
	t.Parallel()

	c := &Carrier{
		Code:  "SwFt",
		Name:  "McDonald Freight LLC",
		City:  "McAllen",
		Email: "Ops@McDonald.com",
	}
	c.ApplyTextFormat(dataentrycontrol.TextFormat{
		CodeCase:  dataentrycontrol.CaseFormatAsEntered,
		NameCase:  dataentrycontrol.CaseFormatAsEntered,
		EmailCase: dataentrycontrol.CaseFormatAsEntered,
		CityCase:  dataentrycontrol.CaseFormatAsEntered,
	})

	assert.Equal(t, "SwFt", c.Code)
	assert.Equal(t, "McDonald Freight LLC", c.Name)
	assert.Equal(t, "McAllen", c.City)
	assert.Equal(t, "Ops@McDonald.com", c.Email)
}
