package conversation

type ThreadAttention struct {
	PendingDecisions int  `json:"pendingDecisions"`
	LastTurnFailed   bool `json:"lastTurnFailed"`
	Unread           bool `json:"unread"`
}

type ThreadAttentionSignals struct {
	PendingDecisions int
	LastTurnStatus   AssistantTurnStatus
}

func (t *Thread) ApplyAttention(signals ThreadAttentionSignals) {
	t.Attention = &ThreadAttention{
		PendingDecisions: signals.PendingDecisions,
		LastTurnFailed:   signals.LastTurnStatus == AssistantTurnStatusFailed,
		Unread:           t.LastMessageAt > t.LastReadAt,
	}
}
