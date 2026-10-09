package assistantjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedQueue struct {
	asked []*serviceports.SettleQueueRequest
	next  *serviceports.QueuedTurn
}

func (q *recordedQueue) SettleQueue(
	_ context.Context,
	req *serviceports.SettleQueueRequest,
) *serviceports.QueuedTurn {
	q.asked = append(q.asked, req)
	if !req.Dispatch {
		return nil
	}

	return q.next
}

type recordedFollowUps struct {
	resumed int
}

func (f *recordedFollowUps) ResumeFollowUps(context.Context, serviceports.ResumeFollowUpsRequest) {
	f.resumed++
}

func continuation(status conversation.AssistantTurnStatus) *conversationContinuation {
	return &conversationContinuation{
		tenant:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		threadID: pulid.MustNew("athr_"),
		userID:   pulid.MustNew("usr_"),
		read:     []pulid.ID{pulid.MustNew("aqm_")},
		status:   status,
		resume:   true,
	}
}

func TestContinueConversation_WhatThePersonQueuedGoesFirst(t *testing.T) {
	t.Parallel()

	queue := &recordedQueue{next: &serviceports.QueuedTurn{TurnID: pulid.MustNew("atrn_")}}
	followUps := &recordedFollowUps{}
	a := &Activities{queue: queue, followUps: followUps}
	c := continuation(conversation.AssistantTurnStatusCompleted)

	next := a.continueConversation(t.Context(), c)

	require.NotNil(t, next)
	require.Len(t, queue.asked, 1)
	assert.True(t, queue.asked[0].Dispatch)
	assert.Equal(t, c.read, queue.asked[0].Read, "what the turn read is cleared from the queue")
	assert.Zero(t, followUps.resumed, "the queued turn reports the decisions when it ends")
}

func TestContinueConversation_WithNothingQueuedADecisionIsReported(t *testing.T) {
	t.Parallel()

	queue := &recordedQueue{}
	followUps := &recordedFollowUps{}
	a := &Activities{queue: queue, followUps: followUps}

	assert.Nil(t, a.continueConversation(t.Context(), continuation(conversation.AssistantTurnStatusRefused)))
	assert.Equal(t, 1, followUps.resumed)
}

func TestContinueConversation_AStoppedOrFailedTurnHoldsTheQueue(t *testing.T) {
	t.Parallel()

	for _, status := range []conversation.AssistantTurnStatus{
		conversation.AssistantTurnStatusStopped,
		conversation.AssistantTurnStatusFailed,
	} {
		queue := &recordedQueue{next: &serviceports.QueuedTurn{TurnID: pulid.MustNew("atrn_")}}
		followUps := &recordedFollowUps{}
		a := &Activities{queue: queue, followUps: followUps}

		assert.Nil(t, a.continueConversation(t.Context(), continuation(status)), status)
		require.Len(t, queue.asked, 1)
		assert.False(t, queue.asked[0].Dispatch, status)
		assert.Equal(t, 1, followUps.resumed, "a decision is still reported after %s", status)
	}
}
