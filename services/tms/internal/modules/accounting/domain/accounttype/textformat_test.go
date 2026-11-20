package accounttype

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

	a := &AccountType{Code: "Rev", Name: "operating revenue"}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "REV", a.Code)
	assert.Equal(t, "Operating Revenue", a.Name)

	b := &AccountType{Code: "Rev", Name: "operating revenue"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "rev", b.Code)
	assert.Equal(t, "OPERATING REVENUE", b.Name)
}
