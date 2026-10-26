package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func slowWaitConnection(t *testing.T, after time.Duration) (*Connection, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.WarnLevel)
	cfg := &config.Config{}
	cfg.Database.SlowWaitWarning = after

	return &Connection{
		cfg:      cfg,
		logger:   observability.NewContextLogger(zap.New(core)),
		settings: connectionSettings{component: "postgres"},
	}, logs
}

func TestSlowWaits_AreOffWithoutAThreshold(t *testing.T) {
	t.Parallel()

	conn, _ := slowWaitConnection(t, 0)
	cfg := conn.slowWaits(scopePoolTenant)

	assert.Zero(t, cfg.After)
	assert.Nil(t, cfg.Report)
}

func TestSlowWaits_LogATenantSlotWaitWithItsHolders(t *testing.T) {
	t.Parallel()

	conn, logs := slowWaitConnection(t, time.Second)
	cfg := conn.slowWaits(scopePoolTenant)
	require.Equal(t, time.Second, cfg.After)

	stack := make([]string, 20)
	for i := range stack {
		stack[i] = "frame"
	}
	cfg.Report(context.Background(), &keel.SlowWait{
		Kind:    keel.WaitTenantSlot,
		Tenant:  "org|bu",
		Waited:  6 * time.Second,
		Stack:   stack,
		Limit:   2,
		Waiters: 3,
		Holders: []keel.SlotHolder{
			{Held: 40 * time.Second, Stack: []string{"rolerepository.GetUserRoleAssignments role.go:264"}},
		},
	})

	entries := logs.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, "tenant_slot", fields["kind"])
	assert.Equal(t, "tenant", fields["pool"])
	assert.Equal(t, "org|bu", fields["tenant"])
	assert.Equal(t, int64(2), fields["limit"])
	assert.Equal(t, int64(3), fields["waiters"])
	assert.Len(t, fields["stack"], slowWaitStackFrames, "a stack is cut to its first frames")

	holders, ok := fields["holders"].([]any)
	require.True(t, ok, "holders: %T", fields["holders"])
	require.Len(t, holders, 1)
	holder, ok := holders[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 40*time.Second, holder["held"])
	assert.Equal(t, []any{"rolerepository.GetUserRoleAssignments role.go:264"}, holder["stack"])
}

func TestSlowWaits_LogAConnectionWaitWithThePool(t *testing.T) {
	t.Parallel()

	conn, logs := slowWaitConnection(t, time.Second)
	conn.slowWaits(scopePoolSystem).Report(context.Background(), &keel.SlowWait{
		Kind:   keel.WaitConnection,
		Waited: 2 * time.Second,
		Pool:   keel.PoolState{Acquired: 8, Idle: 0, Total: 8, Max: 8},
	})

	entries := logs.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, "connection", fields["kind"])
	assert.Equal(t, "system", fields["pool"])
	assert.Equal(t, int32(8), fields["pool_acquired"])
	assert.Equal(t, int32(8), fields["pool_max"])
}

func TestSlowWaits_LogEachKindAtMostOncePerInterval(t *testing.T) {
	t.Parallel()

	conn, logs := slowWaitConnection(t, time.Second)
	report := conn.slowWaits(scopePoolTenant).Report
	for range 5 {
		report(context.Background(), &keel.SlowWait{Kind: keel.WaitTenantSlot})
	}
	report(context.Background(), &keel.SlowWait{Kind: keel.WaitCacheFill})
	assert.Len(t, logs.All(), 2, "a burst of waits of one kind logs once")

	l := &slowWaitLog{last: map[keel.WaitKind]time.Time{}, skipped: map[keel.WaitKind]int{}}
	start := time.Now()
	_, ok := l.admit(keel.WaitConnection, start)
	require.True(t, ok)
	_, ok = l.admit(keel.WaitConnection, start.Add(time.Second))
	require.False(t, ok)
	suppressed, ok := l.admit(keel.WaitConnection, start.Add(slowWaitLogInterval))
	require.True(t, ok)
	assert.Equal(t, 1, suppressed, "the next warning counts the ones it held back")
}
