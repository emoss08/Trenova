package minio

import (
	"testing"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
)

// The bucket is shared, so the retention rule must join whatever lifecycle is
// already there rather than replace it.
func TestWithPayloadRetention_KeepsEveryoneElsesRules(t *testing.T) {
	t.Parallel()

	current := &lifecycle.Configuration{Rules: []lifecycle.Rule{
		{ID: "someone-elses", Status: "Enabled", RuleFilter: lifecycle.Filter{Prefix: "exports/"}},
	}}

	merged := withPayloadRetention(current)

	require.Len(t, merged.Rules, 2)
	assert.Equal(t, "someone-elses", merged.Rules[0].ID)
	assert.Equal(t, temporalPayloadRuleID, merged.Rules[1].ID)
	assert.Equal(t, temporalPayloadPrefix, merged.Rules[1].RuleFilter.Prefix)
	assert.Equal(
		t,
		lifecycle.ExpirationDays(temporalPayloadRetention),
		merged.Rules[1].Expiration.Days,
	)
}

// Running twice must not stack two copies of the same rule.
func TestWithPayloadRetention_ReplacesItsOwnRule(t *testing.T) {
	t.Parallel()

	once := withPayloadRetention(lifecycle.NewConfiguration())
	twice := withPayloadRetention(once)

	require.Len(t, twice.Rules, 1)
	assert.Equal(t, temporalPayloadRuleID, twice.Rules[0].ID)
}

func TestPayloadScope_FilesAPayloadUnderItsExecution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target converter.StorageDriverTargetInfo
		want   string
	}{
		{
			name: "workflow",
			target: converter.StorageDriverWorkflowInfo{
				Namespace:  "default",
				WorkflowID: "assistant-thread/athr_1",
			},
			want: "default/assistant-thread%2Fathr_1",
		},
		{
			name:   "standalone activity",
			target: converter.StorageDriverActivityInfo{Namespace: "default", ActivityID: "act-1"},
			want:   "default/activity/act-1",
		},
		{
			// A workflow ID of ".." must not climb out of the payload prefix.
			name:   "traversal",
			target: converter.StorageDriverWorkflowInfo{Namespace: "default", WorkflowID: ".."},
			want:   "default/_",
		},
		{
			name:   "no target",
			target: nil,
			want:   "unscoped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, payloadScope(tt.target))
		})
	}
}
