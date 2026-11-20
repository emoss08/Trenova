package fleetcode

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

	a := &FleetCode{Code: " Fl Eet ", Description: "  some   description  "}
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "FLEET", a.Code)
	assert.Equal(t, "some description", a.Description)

	b := &FleetCode{Code: " Fl Eet ", Description: "Some  Description"}
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "fleet", b.Code)
	assert.Equal(t, "Some Description", b.Description)
}
