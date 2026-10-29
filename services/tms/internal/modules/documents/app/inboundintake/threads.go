package inboundintake

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
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

func (s *Service) threadKeyFor(ctx context.Context, lookup *threadLookup) (string, error) {
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

	earlier := candidates
	if first >= 0 {
		earlier = candidates[:first]
	}
	key, err := s.replyThreadKey(ctx, lookup.tenant, earlier)
	if err != nil || key != "" {
		return key, err
	}
	if first >= 0 {
		return known[candidates[first]], nil
	}

	return "", nil
}

func (s *Service) replyThreadKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageKeys []string,
) (string, error) {
	if s.replies == nil || len(messageKeys) == 0 {
		return "", nil
	}

	key, found, err := s.replies.ThreadKeyForMessageKeys(ctx, tenantInfo, messageKeys)
	if err != nil || !found {
		return "", err
	}

	return key, nil
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
	key, err := s.threadKeyFor(ctx, &threadLookup{
		tenant:     inboundqueue.TenantOfMessage(message),
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

	tenantInfo := inboundqueue.TenantOfMessage(message)
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

	if _, err = s.queue.RunTransition(ctx, inboundqueue.SettleTransition(&inboundqueue.SettleParams{
		Tenant:   tenantInfo,
		IDs:      ids,
		Status:   inboundmessage.StatusActioned,
		Activity: triage.ActivityTheyReplied,
		Subject:  stringutils.FirstNonEmpty(message.FromName, message.FromAddress),
		Detail:   message.Subject,
		RecordID: message.ID,
		Eligible: func(state *inboundmessageports.QueueState) bool {
			return state.Parked == inboundmessage.ParkingAwaitingReply
		},
	})); err != nil {
		s.l.Error("could not settle the messages a reply answered",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
}
