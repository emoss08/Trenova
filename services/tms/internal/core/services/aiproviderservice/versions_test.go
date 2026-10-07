package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type settingVersions struct {
	created []*settingversion.SettingVersion
}

func (v *settingVersions) Create(_ context.Context, version *settingversion.SettingVersion) error {
	v.created = append(v.created, version)
	return nil
}

func (v *settingVersions) LatestAt(
	_ context.Context,
	req *repositories.GetSettingVersionAtRequest,
) (*settingversion.SettingVersion, error) {
	var best *settingversion.SettingVersion
	for _, version := range v.created {
		if version.Version <= req.Version && (best == nil || version.Version > best.Version) {
			best = version
		}
	}
	return best, nil
}

func TestSaveRecordsTheProviderWithoutItsKey(t *testing.T) {
	t.Parallel()

	versions := &settingVersions{}
	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	svc.db = dbtest.NopConnection{}
	svc.versions = versions
	provider := testProvider(t, svc)
	provider.Version = 3
	author := pulid.MustNew("usr_")

	saved, err := svc.save(t.Context(), func(context.Context) (*aiprovider.Provider, error) {
		return provider, nil
	}, &services.RequestActor{UserID: author})

	require.NoError(t, err)
	assert.Same(t, provider, saved)
	require.Len(t, versions.created, 1)
	version := versions.created[0]
	assert.Equal(t, settingversion.KindAIProvider, version.Kind)
	assert.Equal(t, provider.ID, version.SubjectID)
	assert.Equal(t, int64(3), version.Version)
	assert.Equal(t, author, *version.AuthorID)
	assert.NotContains(t, jsonutils.MustToJSON(version.Snapshot), provider.APIKey)
}

func TestExplainConflictNamesWhatTheOtherSaveChanged(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	loaded := testProvider(t, svc)
	loaded.Version = 2
	current := *loaded
	current.Version = 3
	current.Model = "qwen3:72b"
	current.Priority = 4
	current.UpdatedAt = 1_800_000_900
	other := &tenant.User{ID: pulid.MustNew("usr_"), Name: "Sarah Alvarez"}

	versions := &settingVersions{}
	tenantInfo := pagination.TenantInfo{OrgID: loaded.OrganizationID, BuID: loaded.BusinessUnitID}
	for _, saved := range []*aiprovider.Provider{loaded, &current} {
		snapshot := make(map[string]any)
		require.NoError(t, jsonutils.Convert(saved.Redacted(), &snapshot))
		versions.created = append(versions.created, &settingversion.SettingVersion{
			Kind: settingversion.KindAIProvider, SubjectID: saved.ID,
			Version: saved.Version, Snapshot: snapshot,
		})
	}
	versions.created[1].Author = other
	svc.repo = &fakeProviderRepo{provider: &current}
	svc.versions = versions

	err := svc.explainConflict(t.Context(), &conflictRequest{
		tenantInfo: tenantInfo,
		providerID: loaded.ID,
		loaded:     2,
		cause:      dberror.CreateVersionMismatchError("AIProvider", loaded.ID.String()),
	})

	edit, ok := errortypes.EditConflictOf(err)
	require.True(t, ok)
	assert.Equal(t, int64(3), edit.Version)
	assert.Equal(t, "Sarah Alvarez", edit.UpdatedByName)
	assert.Equal(t, []errortypes.EditConflictChange{
		{Field: "model", Label: "Model"},
		{Field: "priority", Label: "Order"},
	}, edit.Changes)
}

func TestExplainConflictPassesOtherFailuresThrough(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	refused := errortypes.NewBusinessError("refused")

	assert.Same(t, refused, svc.explainConflict(t.Context(), &conflictRequest{cause: refused}))
}
