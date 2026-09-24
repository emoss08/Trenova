package completionrouter

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func documentAttribution() serviceports.AIUsageAttribution {
	return serviceports.AIUsageAttribution{
		UserID:  pulid.MustNew("usr_"),
		Feature: aiusage.FeatureDocumentIntelligenceExtract,
		Subject: aiusage.Subject{
			Type: aiusage.SubjectTypeDocument,
			ID:   pulid.MustNew("doc_").String(),
		},
	}
}

func TestCompleteStructured_RecordsTheFeatureAndSubject(t *testing.T) {
	t.Parallel()

	healthy, _ := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	usage := &fakeUsage{}
	svc := newTestService(t, openAIChatProvider("only", healthy.URL, 10))
	svc.usage = usage

	req := generalRequest()
	req.Attribution = serviceports.AIUsageAttribution{
		UserID:  pulid.MustNew("usr_"),
		Feature: aiusage.FeatureFormulaGenerate,
		Subject: aiusage.Subject{Type: aiusage.SubjectTypeFormulaSchema, ID: "shipment"},
	}
	_, err := svc.CompleteStructured(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, req.Attribution.UserID, rows[0].UserID)
	assert.Equal(t, aiusage.FeatureFormulaGenerate, rows[0].Feature)
	assert.Equal(t, aiusage.SubjectTypeFormulaSchema, rows[0].SubjectType)
	assert.Equal(t, "shipment", rows[0].SubjectID)
}

func TestCompleteChat_RecordsTheFeatureAndSubject(t *testing.T) {
	t.Parallel()

	healthy, _ := chatServer(t, http.StatusOK, "Found Acme Freight.")
	usage := &fakeUsage{}
	svc := newTestService(t, chatProvider("only", healthy.URL, 10))
	svc.usage = usage

	attribution := serviceports.AIUsageAttribution{
		UserID:  pulid.MustNew("usr_"),
		Feature: aiusage.FeatureShipmentImportChat,
		Subject: aiusage.Subject{
			Type: aiusage.SubjectTypeDocument,
			ID:   pulid.MustNew("doc_").String(),
		},
	}
	req := chatRequest(pulid.Nil)
	req.Attribution = attribution
	_, err := svc.CompleteChat(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, aiusage.SurfaceChat, rows[0].Surface)
	assert.Equal(t, attribution.UserID, rows[0].UserID)
	assert.Equal(t, aiusage.FeatureShipmentImportChat, rows[0].Feature)
	assert.Equal(t, aiusage.SubjectTypeDocument, rows[0].SubjectType)
	assert.Equal(t, attribution.Subject.ID, rows[0].SubjectID)
}

func TestRecord_LeavesOutASubjectTheTableCannotHold(t *testing.T) {
	t.Parallel()

	cases := map[string]aiusage.Subject{
		"no type":     {ID: "shipment"},
		"no id":       {Type: aiusage.SubjectTypeDocument},
		"id too long": {Type: aiusage.SubjectTypeDocument, ID: strings.Repeat("é", 101)},
	}
	for name, subject := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			healthy, _ := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
			usage := &fakeUsage{}
			svc := newTestService(t, openAIChatProvider("only", healthy.URL, 10))
			svc.usage = usage

			req := generalRequest()
			req.Attribution = serviceports.AIUsageAttribution{
				Feature: aiusage.FeatureTableQuery,
				Subject: subject,
			}
			_, err := svc.CompleteStructured(t.Context(), req)
			require.NoError(t, err)

			rows := usage.recorded(t, 1)
			require.Len(t, rows, 1)
			assert.Equal(t, aiusage.FeatureTableQuery, rows[0].Feature,
				"the feature is kept when the subject is not")
			assert.Empty(t, rows[0].SubjectType)
			assert.Empty(t, rows[0].SubjectID)
		})
	}
}

func TestSubmitBackground_AnInlineRunKeepsItsAttribution(t *testing.T) {
	t.Parallel()

	inline, _ := chatServer(t, http.StatusOK, `{"documentKind":"RateConfirmation"}`)
	provider := openAIChatProvider("self-hosted", inline.URL, 10)
	provider.Tasks = []aiprovider.Task{aiprovider.TaskDocumentExtraction}
	usage := &fakeUsage{}
	svc := newTestService(t, provider)
	svc.usage = usage

	req := extractionRequest()
	req.Attribution = documentAttribution()
	_, err := svc.SubmitBackground(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, req.Attribution.UserID, rows[0].UserID)
	assert.Equal(t, aiusage.FeatureDocumentIntelligenceExtract, rows[0].Feature)
	assert.Equal(t, req.Attribution.Subject.ID, rows[0].SubjectID)
}

func TestPollBackground_RecordsAFinishedCallOnce(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "completed", "model": "gpt-test",
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{
				{"type": "output_text", "text": `{"documentKind":"RateConfirmation"}`},
			}},
		},
		"usage": map[string]any{"input_tokens": 120, "output_tokens": 44},
	})
	usage := &fakeUsage{}
	svc.usage = usage

	req := pollRequest(provider)
	req.Task = aiprovider.TaskDocumentExtraction
	req.SubmittedAt = time.Now().Add(-90 * time.Second).Unix()
	req.Attribution = documentAttribution()

	_, err := svc.PollBackground(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	row := rows[0]
	assert.True(t, row.Succeeded)
	assert.Equal(t, aiusage.SurfaceBackground, row.Surface)
	assert.Equal(t, aiprovider.TaskDocumentExtraction, row.Task)
	assert.Equal(t, provider.ID, row.ProviderID)
	assert.Equal(t, "gpt-test", row.Model)
	assert.Equal(t, 120, row.InputTokens)
	assert.Equal(t, 44, row.OutputTokens)
	assert.GreaterOrEqual(t, row.LatencyMs, int64(89_000),
		"a background call took as long as it ran, not as long as the poll")
	assert.Equal(t, req.Attribution.UserID, row.UserID)
	assert.Equal(t, aiusage.FeatureDocumentIntelligenceExtract, row.Feature)
	assert.Equal(t, aiusage.SubjectTypeDocument, row.SubjectType)
	assert.Equal(t, req.Attribution.Subject.ID, row.SubjectID)
}

func TestPollBackground_RecordsATerminalFailureWithItsReason(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "incomplete", "model": "gpt-test",
		"incomplete_details": map[string]any{"reason": "max_output_tokens"},
	})
	usage := &fakeUsage{}
	svc.usage = usage

	req := pollRequest(provider)
	req.Attribution = documentAttribution()

	_, err := svc.PollBackground(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Succeeded)
	assert.Equal(t, "provider_error", rows[0].ErrorClass)
	assert.Contains(t, rows[0].ErrorMessage, "max_output_tokens")
	assert.Equal(t, aiprovider.TaskGeneral, rows[0].Task, "an unnamed task prices as general")
	assert.Zero(t, rows[0].LatencyMs, "no submission time means no latency to report")
	assert.Equal(t, aiusage.FeatureDocumentIntelligenceExtract, rows[0].Feature)
}

func TestPollBackground_RecordsNothingWhileTheCallRuns(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "in_progress", "model": "gpt-test",
	})
	usage := &fakeUsage{}
	svc.usage = usage

	outcome, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.NoError(t, err)
	require.Equal(t, serviceports.BackgroundPending, outcome.State)

	usage.mu.Lock()
	defer usage.mu.Unlock()
	assert.Empty(t, usage.rows, "a pending poll is not a call")
}

func TestBackgroundLatency(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_100, 0)

	assert.Equal(t, 100*time.Second, backgroundLatency(1_700_000_000, now))
	assert.Zero(t, backgroundLatency(0, now))
	assert.Zero(t, backgroundLatency(1_700_000_200, now),
		"a clock that ran backwards is not a negative wait")
}
