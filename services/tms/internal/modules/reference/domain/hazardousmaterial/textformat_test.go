package hazardousmaterial

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

	a := &HazardousMaterial{Code: "Un1203", Name: "gasoline blend"}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "UN1203", a.Code)
	assert.Equal(t, "Gasoline Blend", a.Name)

	b := &HazardousMaterial{Code: "Un1203", Name: "gasoline blend"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "un1203", b.Code)
	assert.Equal(t, "GASOLINE BLEND", b.Name)
}
