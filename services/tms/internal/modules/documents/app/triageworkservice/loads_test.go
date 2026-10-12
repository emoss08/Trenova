package triageworkservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLoadRecordsShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].Classification = inboundmessage.ClassificationTender

	updated, err := f.service.CreateLoad(t.Context(), &CreateLoadRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Fields: []LoadFieldInput{
			{Key: inboundmessage.LoadFieldOrigin, Value: " Columbus, OH "},
			{Key: inboundmessage.LoadFieldDestination, Value: "Louisville, KY"},
		},
		IdempotencyKey: "load-key",
	})
	require.NoError(t, err)

	require.Len(t, f.loads.requests, 1)
	req := f.loads.requests[0]
	assert.Equal(t, "load-key", req.IdempotencyKey)
	assert.Equal(t, f.attachment.DocumentID, req.SourceDocumentID)
	require.Len(t, req.Fields, 2)
	assert.Equal(t, "Columbus, OH", req.Fields[0].Value)
	assert.Equal(t, 1, req.Fields[1].Index)

	assert.True(t, updated.ActionState.CreatedShipmentID.IsNotNil())
	require.NotNil(t, updated.LoadDraft)
	assert.Equal(t, "S-200001", f.recorder.find(triage.ActivityLoadCreated).Subject)

	again, err := f.service.CreateLoad(t.Context(), &CreateLoadRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, updated.ActionState.CreatedShipmentID, again.ActionState.CreatedShipmentID)
	assert.Len(t, f.loads.requests, 1)
}

func TestCreateLoadFieldErrors(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.CreateLoad(t.Context(), &CreateLoadRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Fields: []LoadFieldInput{
			{Key: inboundmessage.LoadFieldKey("Color"), Value: "red"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Dayton, OH"},
			{Key: inboundmessage.LoadFieldInstructions, Value: strings.Repeat("x", maxLoadFieldRunes+1)},
		},
	})
	require.Error(t, err)
	for _, path := range []string{"fields[0].key", "fields[2].key", "fields[3].value"} {
		assert.Contains(t, err.Error(), path)
	}
	assert.Empty(t, f.loads.requests)
}

func TestDiscardLoadAfterCreateRefused(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].LoadDraft = &inboundmessage.LoadDraft{}

	discarded, err := f.service.DiscardLoad(t.Context(), &LoadRequest{TenantInfo: testTenant, MessageID: f.message.ID})
	require.NoError(t, err)
	assert.Nil(t, discarded.LoadDraft)

	_, err = f.service.CreateLoad(t.Context(), &CreateLoadRequest{
		TenantInfo: testTenant,
		MessageID:  f.message.ID,
		Fields:     []LoadFieldInput{{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"}},
	})
	require.NoError(t, err)

	_, err = f.service.DiscardLoad(t.Context(), &LoadRequest{TenantInfo: testTenant, MessageID: f.message.ID})
	require.Error(t, err)
}
