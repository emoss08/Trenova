package customer

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

	newCustomer := func() *Customer {
		return &Customer{
			Code:                   "Acme 01",
			Name:                   "acme trucking LLC",
			City:                   "san ANTONIO",
			StatusUpdateRecipients: "Ops@Acme.com, Track@ACME.com",
			EmailProfile: &CustomerEmailProfile{
				ToRecipients:  "AR@Acme.com",
				CCRecipients:  "Cfo@Acme.com",
				BCCRecipients: "Audit@acme.COM",
			},
		}
	}

	a := newCustomer()
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "ACME01", a.Code)
	assert.Equal(t, "Acme Trucking LLC", a.Name)
	assert.Equal(t, "san ANTONIO", a.City)
	assert.Equal(t, "ops@acme.com, track@acme.com", a.StatusUpdateRecipients)
	assert.Equal(t, "ar@acme.com", a.EmailProfile.ToRecipients)
	assert.Equal(t, "cfo@acme.com", a.EmailProfile.CCRecipients)
	assert.Equal(t, "audit@acme.com", a.EmailProfile.BCCRecipients)

	b := newCustomer()
	b.City = "san antonio"
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "acme01", b.Code)
	assert.Equal(t, "ACME TRUCKING LLC", b.Name)
	assert.Equal(t, "San Antonio", b.City)
	assert.Equal(t, "OPS@ACME.COM, TRACK@ACME.COM", b.StatusUpdateRecipients)
	assert.Equal(t, "AR@ACME.COM", b.EmailProfile.ToRecipients)
	assert.Equal(t, "CFO@ACME.COM", b.EmailProfile.CCRecipients)
	assert.Equal(t, "AUDIT@ACME.COM", b.EmailProfile.BCCRecipients)
}

func TestApplyTextFormatWithoutEmailProfile(t *testing.T) {
	t.Parallel()

	c := &Customer{Code: "acme", Name: "acme trucking"}
	c.ApplyTextFormat(textFormatA)

	assert.Nil(t, c.EmailProfile)
	assert.Equal(t, "ACME", c.Code)
	assert.Equal(t, "Acme Trucking", c.Name)
}
