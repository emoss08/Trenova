package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProfiles struct {
	profiles []*email.Profile
	request  *repositories.EmailProfileSelectOptionsRequest
}

func (f *fakeProfiles) SelectProfileOptions(
	_ context.Context,
	req *repositories.EmailProfileSelectOptionsRequest,
) (*pagination.ListResult[*email.Profile], error) {
	f.request = req

	return &pagination.ListResult[*email.Profile]{Items: f.profiles}, nil
}

// The email tools took a profileId with no way to find one; a model that
// wants to write to a customer needs to know what it may send from.
func TestListEmailProfiles_ListsActiveSendingProfiles(t *testing.T) {
	t.Parallel()

	profiles := &fakeProfiles{profiles: []*email.Profile{
		{ID: pulid.MustNew("emp_"), Name: "Operations", SenderName: "Acme Dispatch", SenderEmail: "dispatch@acme.example", Status: email.ProfileStatusActive},
		{ID: pulid.MustNew("emp_"), Name: "Retired", SenderEmail: "old@acme.example", Status: email.ProfileStatusInactive},
	}}
	tool := newListEmailProfilesTool(profiles)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "ops"}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]emailProfileRow)
	require.True(t, ok)
	require.Len(t, rows, 1, "an inactive profile is not offered")
	assert.Equal(t, "Operations", rows[0].Name)
	assert.Equal(t, "dispatch@acme.example", rows[0].SenderEmail)
	assert.Equal(t, "ops", profiles.request.SelectQueryRequest.Query)
	assert.Equal(t, permission.ResourceEmailProfile, tool.PermissionResource())
}
