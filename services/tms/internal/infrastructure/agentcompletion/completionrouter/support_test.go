package completionrouter

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSupport struct {
	refusal error
	asked   []pagination.TenantInfo
}

func (r *recordingSupport) AdmitModelCall(_ context.Context, tenant pagination.TenantInfo) error {
	r.asked = append(r.asked, tenant)
	return r.refusal
}

func TestCompletion_ACallTheSupportPolicyRefusesReachesNoProvider(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("only", server.URL, 10))
	support := &recordingSupport{refusal: supportaccess.ErrModelCallRefused()}
	svc.Support = support

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	var refusal *errortypes.BusinessError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Message, "AI providers")
	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured,
		"Nova's availability reads the refusal as a reason it cannot run, not as a failure")

	_, err = svc.CompleteChat(t.Context(), chatRequest(""))
	require.Error(t, err)
	assert.Zero(t, calls.Load(), "nothing is spent on the organization's provider")
}

func TestCompletion_ACallTheSupportPolicyAdmitsRunsAsBefore(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("only", server.URL, 10))
	support := &recordingSupport{}
	svc.Support = support

	req := generalRequest()
	_, err := svc.CompleteStructured(t.Context(), req)
	require.NoError(t, err)
	assert.EqualValues(t, 1, calls.Load())
	require.NotEmpty(t, support.asked)
	assert.Equal(t, req.TenantInfo, support.asked[0], "the policy judges the tenant the call is for")
}
