package inboundmessageservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type threadLookup struct {
	tenant     pagination.TenantInfo
	ownKey     string
	inReplyTo  string
	references []string
}

func (s *Service) threadKeyFor(ctx context.Context, lookup threadLookup) (string, error) {
	candidates := inboundmessage.ThreadCandidates(lookup.inReplyTo, lookup.references)
	if lookup.ownKey != "" {
		candidates = removeKey(candidates, lookup.ownKey)
	}
	if len(candidates) == 0 {
		return "", nil
	}

	known, err := s.queueRepo.ThreadKeysByMessageKeys(ctx, inboundmessageports.ThreadKeysRequest{
		TenantInfo:  lookup.tenant,
		MessageKeys: candidates,
	})
	if err != nil {
		return "", err
	}

	first := -1
	for idx, candidate := range candidates {
		if _, ok := known[candidate]; ok {
			first = idx
			break
		}
	}

	if s.replies != nil {
		earlier := candidates
		if first >= 0 {
			earlier = candidates[:first]
		}
		if len(earlier) > 0 {
			key, found, rErr := s.replies.ThreadKeyForMessageKeys(ctx, lookup.tenant, earlier)
			if rErr != nil {
				return "", rErr
			}
			if found && key != "" {
				return key, nil
			}
		}
	}
	if first >= 0 {
		return known[candidates[first]], nil
	}

	return "", nil
}

func removeKey(candidates []string, key string) []string {
	kept := candidates[:0]
	for _, candidate := range candidates {
		if candidate != key {
			kept = append(kept, candidate)
		}
	}

	return kept
}

func (s *Service) joinThread(ctx context.Context, message *inboundmessage.InboundMessage) {
	key, err := s.threadKeyFor(ctx, threadLookup{
		tenant:     tenantOfMessage(message),
		ownKey:     inboundmessage.NormalizeMessageID(message.MessageID),
		inReplyTo:  message.InReplyTo,
		references: message.References,
	})
	if err != nil {
		s.l.Warn("could not find the thread an inbound message belongs to; it starts its own",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
	if key == "" {
		key = inboundmessage.DeriveThreadKey(message.MessageID, message.ID)
	}
	message.ThreadKey = key
}

func (s *Service) answerAwaiting(ctx context.Context, message *inboundmessage.InboundMessage) {
	if message.ThreadKey == "" || message.Status == inboundmessage.StatusIgnored {
		return
	}
	if message.InReplyTo == "" && len(message.References) == 0 {
		return
	}

	tenantInfo := tenantOfMessage(message)
	waiting, err := s.queueRepo.AwaitingOnThread(ctx, inboundmessageports.AwaitingOnThreadRequest{
		TenantInfo: tenantInfo,
		ThreadKey:  message.ThreadKey,
		ExcludeID:  message.ID,
	})
	if err != nil {
		s.l.Warn("could not look for messages waiting on this reply",
			zap.String("messageId", message.ID.String()), zap.Error(err))

		return
	}
	if len(waiting) == 0 {
		return
	}

	ids := make([]pulid.ID, 0, len(waiting))
	for _, state := range waiting {
		ids = append(ids, state.ID)
	}

	if _, err = s.runTransition(ctx, settleTransition(settleParams{
		tenant:   tenantInfo,
		ids:      ids,
		status:   inboundmessage.StatusActioned,
		activity: triage.ActivityTheyReplied,
		subject:  stringutils.FirstNonEmpty(message.FromName, message.FromAddress),
		detail:   message.Subject,
		recordID: message.ID,
		eligible: func(state *inboundmessageports.QueueState) bool {
			return state.Parked == inboundmessage.ParkingAwaitingReply
		},
	})); err != nil {
		s.l.Error("could not settle the messages a reply answered",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
}

func (s *Service) Thread(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	threadKey string,
) ([]*inboundmessage.InboundMessage, error) {
	threadKey = strings.TrimSpace(threadKey)
	if threadKey == "" || len(threadKey) > inboundmessage.MaxThreadKeyLength {
		return nil, errortypes.NewValidationError("threadKey", errortypes.ErrInvalid,
			"Say which thread to read")
	}

	return s.queueRepo.ListThread(ctx, inboundmessageports.ListThreadRequest{
		TenantInfo: tenantInfo,
		ThreadKey:  threadKey,
		Limit:      inboundmessageports.MaxThreadMessages,
	})
}
