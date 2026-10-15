package ediconnectiondraft

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreview_ShowsEachRecordAsItIsAndAsItWouldBe(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	existing := existingAcme(req)
	h.repo.partners = []*ediports.EDIPartnerSetup{existing}

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.True(t, plan.Draft.Ready)
	assert.Equal(t, ediports.ChangeKindChange, plan.Partner.Change)
	require.NotNil(t, plan.Partner.Before)
	assert.Equal(t, "Lee Park", plan.Partner.Before.ContactName)
	assert.Equal(t, "Dana Whitfield", plan.Partner.After.ContactName)
	assert.Equal(t, existing.Partner.Version, plan.Partner.After.Version, "the write is pinned")

	require.NotNil(t, plan.Transport)
	assert.Equal(t, ediports.ChangeKindSame, plan.Transport.Change)
	assert.NotContains(t, plan.Transport.After.Config, "privateKey", "secrets never enter a plan")

	byKey := map[string]ediports.PlannedDocumentProfile{}
	for _, profile := range plan.DocumentProfiles {
		byKey[profileKey(profile.After.TransactionSet, profile.After.Direction)] = profile
	}
	require.Len(t, byKey, 4)
	status := byKey[profileKey(edi.TransactionSet214, edi.DocumentDirectionOutbound)]
	assert.Equal(t, ediports.ChangeKindSame, status.Change)
	assert.Same(t, existing.DocumentProfiles[0], status.Before)
	tender := byKey[profileKey(edi.TransactionSet204, edi.DocumentDirectionInbound)]
	assert.Equal(t, ediports.ChangeKindAdd, tender.Change)
	assert.Nil(t, tender.Before)
	assert.Equal(t, "007912345", tender.After.Envelope.InterchangeSenderID)

	require.Len(t, plan.ControlNumbers, 2, "the 990 and 210 start at 1000; the 214 already passed it")
	for _, control := range plan.ControlNumbers {
		assert.Equal(t, ediports.ChangeKindAdd, control.Change)
		assert.Equal(t, int64(1000), control.After.NextValue)
	}
}
