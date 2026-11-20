package documenttype

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

	a := &DocumentType{Code: "Po D", Name: "proof of delivery"}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "POD", a.Code)
	assert.Equal(t, "Proof Of Delivery", a.Name)

	b := &DocumentType{Code: "Po D", Name: "proof of delivery"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "pod", b.Code)
	assert.Equal(t, "PROOF OF DELIVERY", b.Name)
}
