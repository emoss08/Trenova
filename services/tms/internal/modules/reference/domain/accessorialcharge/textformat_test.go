package accessorialcharge

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

	a := &AccessorialCharge{Code: " DeTention "}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "DETENTION", a.Code)

	b := &AccessorialCharge{Code: "DeTention"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "detention", b.Code)
}
