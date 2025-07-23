// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package notification_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/stretchr/testify/assert"
)

func TestPriority_ShouldBypassQuietHours(t *testing.T) {
	tests := []struct {
		name     string
		priority notification.Priority
		want     bool
	}{
		{
			name:     "critical priority bypasses quiet hours",
			priority: notification.PriorityCritical,
			want:     true,
		},
		{
			name:     "high priority bypasses quiet hours",
			priority: notification.PriorityHigh,
			want:     true,
		},
		{
			name:     "medium priority does not bypass quiet hours",
			priority: notification.PriorityMedium,
			want:     false,
		},
		{
			name:     "low priority does not bypass quiet hours",
			priority: notification.PriorityLow,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.priority.ShouldBypassQuietHours()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPriority_ShouldBypassBatching(t *testing.T) {
	tests := []struct {
		name     string
		priority notification.Priority
		want     bool
	}{
		{
			name:     "critical priority bypasses batching",
			priority: notification.PriorityCritical,
			want:     true,
		},
		{
			name:     "high priority bypasses batching",
			priority: notification.PriorityHigh,
			want:     true,
		},
		{
			name:     "medium priority does not bypass batching",
			priority: notification.PriorityMedium,
			want:     false,
		},
		{
			name:     "low priority does not bypass batching",
			priority: notification.PriorityLow,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.priority.ShouldBypassBatching()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPriority_CanBeBatched(t *testing.T) {
	tests := []struct {
		name     string
		priority notification.Priority
		want     bool
	}{
		{
			name:     "critical priority cannot be batched",
			priority: notification.PriorityCritical,
			want:     false,
		},
		{
			name:     "high priority cannot be batched",
			priority: notification.PriorityHigh,
			want:     false,
		},
		{
			name:     "medium priority can be batched",
			priority: notification.PriorityMedium,
			want:     true,
		},
		{
			name:     "low priority can be batched",
			priority: notification.PriorityLow,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.priority.CanBeBatched()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPriority_GetLevel(t *testing.T) {
	tests := []struct {
		name     string
		priority notification.Priority
		want     int
	}{
		{
			name:     "critical priority has level 4",
			priority: notification.PriorityCritical,
			want:     4,
		},
		{
			name:     "high priority has level 3",
			priority: notification.PriorityHigh,
			want:     3,
		},
		{
			name:     "medium priority has level 2",
			priority: notification.PriorityMedium,
			want:     2,
		},
		{
			name:     "low priority has level 1",
			priority: notification.PriorityLow,
			want:     1,
		},
		{
			name:     "unknown priority has level 0",
			priority: notification.Priority("unknown"),
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.priority.GetLevel()
			assert.Equal(t, tt.want, got)
		})
	}
}
