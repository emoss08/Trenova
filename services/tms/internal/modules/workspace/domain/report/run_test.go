package report_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportRun_StorageKeysAreNotReportable(t *testing.T) {
	t.Parallel()

	run := reflect.TypeFor[report.ReportRun]()
	for _, name := range []string{"ArtifactKey", "RowsKey"} {
		field, ok := run.FieldByName(name)
		require.True(t, ok, name)
		assert.Contains(t, strings.Split(field.Tag.Get("keel"), ","), "noreport", name)
	}
}
