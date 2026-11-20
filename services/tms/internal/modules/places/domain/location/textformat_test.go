package location

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

	a := &Location{Code: "Dal 01", Name: "dallas yard", City: " dALLAS "}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "DAL01", a.Code)
	assert.Equal(t, "Dallas Yard", a.Name)
	assert.Equal(t, "dALLAS", a.City)

	b := &Location{Code: "Dal 01", Name: "dallas yard", City: "dallas"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "dal01", b.Code)
	assert.Equal(t, "DALLAS YARD", b.Name)
	assert.Equal(t, "Dallas", b.City)
}
