package tariff

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestTablePinsComeFromTheTablesACheckRead(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	tariffCtx := &rules.TariffContext{Tables: []rules.TariffTable{
		{Code: "ZONES", Version: 3, MatrixID: matrixID},
		{Code: "UNREAD", Version: 1, MatrixID: pulid.MustNew("rmx_")},
		{Code: "STUDIO", Version: 0},
	}}
	three := int64(3)
	check := &rules.TariffCheck{Tables: []rules.TariffTableRead{
		{Code: "ZONES", Version: &three},
		{Code: "STUDIO"},
		{Code: "MISSING"},
	}}

	assert.Equal(t, []tarifftypes.TablePin{{
		Code:          "ZONES",
		RateMatrixID:  matrixID,
		VersionNumber: 3,
	}}, TablePins(check, tariffCtx))
	assert.Empty(t, TablePins(nil, tariffCtx))
}
