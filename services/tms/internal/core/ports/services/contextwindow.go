package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ChatContextWindowRequest struct {
	TenantInfo          pagination.TenantInfo
	PreferredProviderID pulid.ID
	Pinned              bool
}

type ChatContextWindow struct {
	Tokens      int `json:"tokens"`
	ReplyTokens int `json:"replyTokens"`
}

func (w ChatContextWindow) Known() bool { return w.Tokens > 0 }

type ChatContextWindowResolver interface {
	ChatContextWindow(ctx context.Context, req ChatContextWindowRequest) (ChatContextWindow, error)
}

type HistorySummary struct {
	Content         string `json:"content"`
	Tainted         bool   `json:"tainted"`
	ThroughSequence int    `json:"throughSequence"`
}

type ContextUsage struct {
	WindowTokens        int  `json:"windowTokens"`
	ReplyTokens         int  `json:"replyTokens"`
	PromptTokens        int  `json:"promptTokens"`
	HistoryBudgetTokens int  `json:"historyBudgetTokens"`
	HistoryTokens       int  `json:"historyTokens"`
	DroppedTurns        int  `json:"droppedTurns,omitempty"`
	Summarized          bool `json:"summarized,omitempty"`
}

func (u *ContextUsage) Fitted() bool { return u != nil && u.HistoryBudgetTokens > 0 }
