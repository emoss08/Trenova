package assistantturnservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// TurnsResource is the realtime resource a reply starting or ending is
// announced under.
const TurnsResource = "assistant_turns"

const (
	turnActionStarted  = "started"
	turnActionFinished = "finished"
)

// announceTimeout bounds telling the person's other tabs about a turn. It is a
// hint riding on a request or an activity that has its own work to finish.
const announceTimeout = 5 * time.Second

// TurnAnnouncement is what the person's other tabs are told about a reply
// starting or ending. It carries identifiers and the status and nothing the
// conversation said: a conversation is its owner's alone, and a tab reads the
// rest from the live-turns endpoint, scoped to them. The announcement itself
// is addressed to the owner, so no one else's stream carries it.
type TurnAnnouncement struct {
	TurnID   pulid.ID                         `json:"turnId"`
	ThreadID pulid.ID                         `json:"threadId"`
	UserID   pulid.ID                         `json:"userId"`
	Status   conversation.AssistantTurnStatus `json:"status"`
	Origin   conversation.AssistantTurnOrigin `json:"origin"`
}

// announce tells the person's other tabs a reply started or ended, so none of
// them has to poll to learn it. It is best effort: the record is what is
// true, and a lost announcement costs a tab a refresh, never the turn.
func (s *Service) announce(
	ctx context.Context,
	turn *conversation.AssistantTurn,
	action string,
	status conversation.AssistantTurnStatus,
) {
	if s.realtime == nil {
		return
	}

	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), announceTimeout)
	defer cancel()

	err := s.realtime.PublishResourceInvalidation(
		publishCtx,
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: turn.OrganizationID,
			BusinessUnitID: turn.BusinessUnitID,
			AudienceUserID: turn.UserID,
			Resource:       TurnsResource,
			Action:         action,
			RecordID:       turn.ID,
			Entity: TurnAnnouncement{
				TurnID:   turn.ID,
				ThreadID: turn.ThreadID,
				UserID:   turn.UserID,
				Status:   status,
				Origin:   turn.Origin,
			},
		},
	)
	if err != nil {
		s.l.Warn("could not announce an assistant turn",
			zap.String("turn", turn.ID.String()),
			zap.String("action", action),
			zap.Error(err),
		)
	}
}
