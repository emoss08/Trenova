package edilading

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func carrierStarter(t *testing.T) *lading.Starter {
	t.Helper()
	built, err := lading.Default().Starter(t.Context(), &lading.StarterRequest{
		Name:  "acme",
		Label: "Acme Retail Group",
		Role:  lading.RoleCarrier,
		Them:  &[2]string{"ZZ", "ACMERETAIL"},
		Us:    &[2]string{"ZZ", "TRENOVA"},
		Zone:  "America/Chicago",
	})
	require.NoError(t, err)
	require.NotEmpty(t, built.Samples, "the carrier starter's flow tests read its samples")
	return built
}

func projectsWith(t *testing.T, repo ladingports.LadingProjectRepository) *Projects {
	t.Helper()
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{
			Name:     "Trenova Logistics",
			ScacCode: "TRNV",
			Timezone: "America/Chicago",
		}, nil)
	return &Projects{
		l:        zap.NewNop(),
		runtime:  NewRuntime(RuntimeParams{Logger: zap.NewNop(), Orgs: orgs}),
		projects: repo,
	}
}

func sampleRefs(samples []lading.Sample) ([]edi.LadingSampleRef, []*edi.LadingSample) {
	refs := make([]edi.LadingSampleRef, 0, len(samples))
	stored := make([]*edi.LadingSample, 0, len(samples))
	for _, sample := range samples {
		digest := lading.Digest(sample.Content)
		refs = append(refs, edi.LadingSampleRef{
			Path:   sample.Path,
			Digest: digest,
			Size:   len(sample.Content),
		})
		stored = append(stored, &edi.LadingSample{
			Digest:  digest,
			Size:    len(sample.Content),
			Content: sample.Content,
		})
	}
	return refs, stored
}

func TestSaveDraft_ChecksTheFilesWithTheSamplesTheirTestsRead(t *testing.T) {
	t.Parallel()

	starter := carrierStarter(t)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	draft := &edi.LadingVersion{
		ID:        pulid.MustNew("ldv_"),
		ProjectID: pulid.MustNew("ldp_"),
		Status:    edi.LadingVersionStatusDraft,
		Version:   3,
	}
	repo := ladingportsmock.NewMockLadingProjectRepository(t)
	repo.EXPECT().GetVersion(mock.Anything, mock.Anything).Return(draft, nil)
	var saved *edi.LadingVersion
	repo.EXPECT().SaveDraft(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			version *edi.LadingVersion,
			_ []*edi.LadingSample,
		) (*edi.LadingVersion, error) {
			saved = version
			return version, nil
		})

	_, err := projectsWith(t, repo).SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo: tenantInfo,
		VersionID:  draft.ID,
		Files:      starter.Files,
		Samples:    starter.Samples,
		Version:    3,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Zero(t, saved.ErrorCount, "%+v", saved.Manifest.Files)
	assert.Zero(t, saved.Manifest.Errors)
	assert.Len(t, saved.Samples, len(starter.Samples))
}

func TestNewDraft_ChecksTheVersionWithItsStoredSamples(t *testing.T) {
	t.Parallel()

	starter := carrierStarter(t)
	refs, stored := sampleRefs(starter.Samples)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	active := &edi.LadingVersion{
		ID:        pulid.MustNew("ldv_"),
		ProjectID: pulid.MustNew("ldp_"),
		Number:    2,
		Status:    edi.LadingVersionStatusActive,
		Files:     starter.Files,
		Samples:   refs,
	}
	repo := ladingportsmock.NewMockLadingProjectRepository(t)
	repo.EXPECT().GetVersion(mock.Anything, mock.Anything).Return(active, nil)
	sameProject := mock.MatchedBy(func(req ladingports.GetLadingSamplesRequest) bool {
		return req.ProjectID == active.ProjectID && len(req.Digests) == len(stored)
	})
	repo.EXPECT().GetSamples(mock.Anything, sameProject).Return(stored, nil)
	var created *edi.LadingVersion
	repo.EXPECT().CreateVersion(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			version *edi.LadingVersion,
			_ []*edi.LadingSample,
		) (*edi.LadingVersion, error) {
			created = version
			return version, nil
		})

	_, err := projectsWith(t, repo).NewDraft(t.Context(), &NewDraftRequest{
		TenantInfo:    tenantInfo,
		ProjectID:     active.ProjectID,
		FromVersionID: active.ID,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, edi.LadingVersionStatusDraft, created.Status)
	assert.Zero(t, created.ErrorCount, "%+v", created.Manifest.Files)
	assert.Equal(t, refs, created.Samples)
}
