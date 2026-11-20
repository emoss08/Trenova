package glaccount

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

	a := &GLAccount{AccountCode: "Ar 1000", Name: "accounts receivable"}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "AR1000", a.AccountCode)
	assert.Equal(t, "Accounts Receivable", a.Name)

	b := &GLAccount{AccountCode: "Ar 1000", Name: "accounts receivable"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "ar1000", b.AccountCode)
	assert.Equal(t, "ACCOUNTS RECEIVABLE", b.Name)
}
