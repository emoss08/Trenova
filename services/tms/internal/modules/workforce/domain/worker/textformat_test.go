package worker

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

	newWorker := func() *Worker {
		return &Worker{
			FirstName:            " jane ",
			LastName:             "mcDonald",
			EmergencyContactName: "john  doe",
			City:                 "portland",
			Email:                "Jane.McDonald@Example.COM",
		}
	}

	a := newWorker()
	a.ApplyTextFormat(textFormatA)
	assert.Equal(t, "Jane", a.FirstName)
	assert.Equal(t, "McDonald", a.LastName)
	assert.Equal(t, "John Doe", a.EmergencyContactName)
	assert.Equal(t, "portland", a.City)
	assert.Equal(t, "jane.mcdonald@example.com", a.Email)

	b := newWorker()
	b.ApplyTextFormat(textFormatB)
	assert.Equal(t, "JANE", b.FirstName)
	assert.Equal(t, "MCDONALD", b.LastName)
	assert.Equal(t, "JOHN DOE", b.EmergencyContactName)
	assert.Equal(t, "Portland", b.City)
	assert.Equal(t, "JANE.MCDONALD@EXAMPLE.COM", b.Email)
}
