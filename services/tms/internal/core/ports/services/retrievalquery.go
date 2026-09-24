package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
)

var ErrQueryTextRequired = errors.New("a query vector needs text to embed")

type QueryVectorRequest struct {
	TenantInfo  pagination.TenantInfo
	Text        string
	Attribution AIUsageAttribution
}

type QueryVector struct {
	Available  bool
	Reason     airetrieval.UnavailableReason
	Vector     []float32
	ModelKey   string
	Dimensions int
}

func UnavailableQueryVector(reason airetrieval.UnavailableReason) QueryVector {
	return QueryVector{Reason: reason}
}

func (q QueryVector) Usable() bool {
	return q.Available && len(q.Vector) > 0 && len(q.Vector) == q.Dimensions && q.ModelKey != ""
}

type QueryVectorizer interface {
	Vectorize(ctx context.Context, req QueryVectorRequest) (QueryVector, error)
	Availability(ctx context.Context, tenant pagination.TenantInfo) (airetrieval.Availability, error)
}

func TurnQueryText(input string, history []conversation.Message) string {
	for idx := len(history) - 1; idx >= 0; idx-- {
		message := history[idx]
		if message.Role != conversation.RoleUser || message.Delegated() {
			continue
		}

		return input + "\n" + message.Content
	}

	return input
}

func TurnQueryRequest(req *RunRequest) QueryVectorRequest {
	if req == nil || req.Actor == nil {
		return QueryVectorRequest{}
	}

	attribution := AIUsageAttribution{
		UserID:   req.Actor.UserID,
		ThreadID: req.ThreadID,
		RunID:    req.RunID,
		Purpose:  req.AttributedPurpose(),
	}
	if req.Definition != nil {
		attribution.AgentDefinitionID = req.Definition.ID
	}

	return QueryVectorRequest{
		TenantInfo:  req.Actor.TenantInfo(),
		Text:        TurnQueryText(req.Input, req.History),
		Attribution: attribution,
	}
}
