package inboundreplyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanSaveDraft_IsTheDraftSaveDraftStores(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &SaveDraftRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Body:       "  It delivers tomorrow before noon.  ",
	}

	plan, err := f.service.PlanSaveDraft(t.Context(), req)
	require.NoError(t, err)
	assert.Nil(t, plan.Current, "the message has no draft yet")
	assert.Equal(t, "It delivers tomorrow before noon.", plan.Draft.Body)
	assert.Equal(t, "edi@partner.example", plan.Draft.ToAddress)
	assert.Equal(t, "Re: Where is my load?", plan.Draft.Subject)
	assert.Empty(t, f.replies.byStatus(triage.ReplyDraft), "a plan saves no draft")

	saved, err := f.service.SaveDraft(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, plan.Draft.Body, saved.Body)
	assert.Equal(t, plan.Draft.Subject, saved.Subject)
	assert.Equal(t, plan.Draft.ToAddress, saved.ToAddress)

	again, err := f.service.PlanSaveDraft(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, again.Current, "a second draft replaces the first")
	assert.Equal(t, saved.ID, again.Current.ID)
}

func TestPlanDraftReply_RefusesWhatDraftReplyRefuses(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &DraftReplyRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Mode:       triagenova.Interactive,
	}

	plan, err := f.service.PlanDraftReply(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.Draft.DraftedByNova)
	assert.Empty(t, plan.Draft.Body, "the body is written when the draft runs")
	assert.Empty(t, f.nova.requests, "a plan asks Nova nothing")

	f.nova.available = false
	_, err = f.service.PlanDraftReply(t.Context(), req)
	require.ErrorIs(t, err, triagenova.ErrUnavailable)
}

func TestPlanDiscardDraft_NamesTheDraftItWouldThrowAway(t *testing.T) {
	t.Parallel()

	f := newFixture()

	plan, err := f.service.PlanDiscardDraft(t.Context(), testTenant, f.message.ID)
	require.NoError(t, err)
	assert.Nil(t, plan.Current)

	saved, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Body:       "First",
	})
	require.NoError(t, err)

	plan, err = f.service.PlanDiscardDraft(t.Context(), testTenant, f.message.ID)
	require.NoError(t, err)
	require.NotNil(t, plan.Current)
	assert.Equal(t, saved.ID, plan.Current.ID)
	assert.Len(t, f.replies.byStatus(triage.ReplyDraft), 1, "a plan discards nothing")
}
