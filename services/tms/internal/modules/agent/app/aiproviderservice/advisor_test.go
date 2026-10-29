package aiproviderservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listingTester struct {
	fakeTester
	models []aiproviderports.AIProviderModelOption
	err    error
	asked  int
}

func (f *listingTester) ListModels(
	_ context.Context,
	_ *aiproviderports.ProbeAIProviderModelsRequest,
) ([]aiproviderports.AIProviderModelOption, error) {
	f.asked++

	return f.models, f.err
}

func advisedDraft() *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:             pulid.MustNew("aiprv_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           aiprovider.KindAnthropicMessages,
		Model:          "claude-haiku-5-5",
		Tasks:          []aitask.Task{aitask.TaskAssistantChat},
		AdvisorEnabled: true,
	}
}

func confirm(
	t *testing.T,
	tester *listingTester,
	before, after *aiprovider.Provider,
) *errortypes.MultiError {
	t.Helper()

	svc := newTestService(&fakeProviderRepo{provider: before}, &fakeProber{})
	svc.tester = tester
	typed := "sk-ant-api03-typed"
	multiErr := errortypes.NewMultiError()
	svc.confirmAdvisor(t.Context(), &advisorCheck{
		before: before,
		after:  after,
		req: &aiproviderports.SaveAIProviderRequest{
			APIKey: &typed,
			TenantInfo: pagination.TenantInfo{
				OrgID: after.OrganizationID,
				BuID:  after.BusinessUnitID,
			},
		},
	}, multiErr)

	return multiErr
}

func TestConfirmAdvisor_TheModelListMustNameTheAdvisor(t *testing.T) {
	t.Parallel()

	listed := &listingTester{models: []aiproviderports.AIProviderModelOption{
		{ID: "claude-haiku-5-5"}, {ID: "claude-opus-5-5"},
	}}
	assert.False(t, confirm(t, listed, nil, advisedDraft()).HasErrors())
	assert.Equal(t, 1, listed.asked)

	missing := advisedDraft()
	missing.AdvisorModel = "claude-fable-5-1"
	multiErr := confirm(t, listed, nil, missing)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "does not include claude-fable-5-1")
}

func TestConfirmAdvisor_AnUnreadableListRefusesTheSave(t *testing.T) {
	t.Parallel()

	multiErr := confirm(t, &listingTester{err: errors.New("connection refused")}, nil,
		advisedDraft())
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "could not be read")
}

func TestConfirmAdvisor_AnUnchangedAdvisorIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	tester := &listingTester{}
	before := advisedDraft()
	after := *before
	after.Name = "Renamed"
	assert.False(t, confirm(t, tester, before, &after).HasErrors())
	assert.Zero(t, tester.asked, "a rename does not ask the endpoint for its models")

	off := advisedDraft()
	off.AdvisorEnabled = false
	assert.False(t, confirm(t, tester, nil, off).HasErrors())
	assert.Zero(t, tester.asked, "an advisor left off is never checked")
}
