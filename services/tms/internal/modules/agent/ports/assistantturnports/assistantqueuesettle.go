package assistantturnports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SettleQueueRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadID   pulid.ID
	UserID     pulid.ID
	Read       []pulid.ID
	Dispatch   bool
}

type QueuedTurn struct {
	TurnID   pulid.ID `json:"turnId"`
	QueuedID pulid.ID `json:"queuedId"`
	Input    string   `json:"input"`
}

type AssistantQueueSettler interface {
	SettleQueue(ctx context.Context, req *SettleQueueRequest) *QueuedTurn
}
