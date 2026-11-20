package factoringcompany

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

	a := &FactoringCompany{
		Code:  "Rts",
		Name:  "RTS financial LLC",
		City:  "oVERLAND park",
		Email: " Noa@RTS.com ",
	}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "RTS", a.Code)
	assert.Equal(t, "RTS Financial LLC", a.Name)
	assert.Equal(t, "oVERLAND park", a.City)
	assert.Equal(t, "noa@rts.com", a.Email)

	b := &FactoringCompany{
		Code:  "Rts",
		Name:  "rts financial",
		City:  "overland park",
		Email: "Noa@RTS.com",
	}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "rts", b.Code)
	assert.Equal(t, "RTS FINANCIAL", b.Name)
	assert.Equal(t, "Overland Park", b.City)
	assert.Equal(t, "NOA@RTS.COM", b.Email)
}
