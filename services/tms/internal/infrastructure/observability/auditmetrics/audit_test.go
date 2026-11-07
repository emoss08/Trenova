package auditmetrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewAudit_DisabledRecordsNothingAndDoesNotPanic(t *testing.T) {
	t.Parallel()

	m := NewAudit(nil, zap.NewNop(), false)
	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())

	m.RecordWrite(map[string]int{"Operational": 1}, true, time.Millisecond)
	m.RecordSecurityEvent("auth_event", false)
	m.RecordRetentionDeleted("Access", 3)
	m.RecordPartitions("created", 1)
	m.RecordSeal(true)
	m.RecordVerification("Verified")
	m.RecordAlert("RoleChanged")
	m.RecordExport("CSV", true)

	var nilMetrics *Audit
	nilMetrics.RecordWrite(map[string]int{"Access": 1}, false, time.Millisecond)
}

func TestNewAudit_CountsWritesByClassAndOutcome(t *testing.T) {
	t.Parallel()

	m := NewAudit(prometheus.NewRegistry(), zap.NewNop(), true)
	m.RecordWrite(map[string]int{"Financial": 2, "Access": 1}, true, time.Millisecond)
	m.RecordWrite(map[string]int{"Financial": 1}, false, time.Millisecond)

	assert.InDelta(t, 2, testutil.ToFloat64(m.entriesWritten.WithLabelValues("Financial", "success")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.entriesWritten.WithLabelValues("Financial", "failure")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.entriesWritten.WithLabelValues("Access", "success")), 0)
}

func TestNewAudit_CountsMaintenance(t *testing.T) {
	t.Parallel()

	m := NewAudit(prometheus.NewRegistry(), zap.NewNop(), true)
	m.RecordRetentionDeleted("Access", 5)
	m.RecordPartitions("dropped", 2)
	m.RecordAlert("BulkDelete")

	assert.InDelta(t, 5, testutil.ToFloat64(m.retentionDeleted.WithLabelValues("Access")), 0)
	assert.InDelta(t, 2, testutil.ToFloat64(m.partitions.WithLabelValues("dropped")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.alerts.WithLabelValues("BulkDelete")), 0)
}
