package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type versionStore struct {
	created  []*agentdefinition.DefinitionVersion
	byNumber map[int64]*agentdefinition.DefinitionVersion
}

func (v *versionStore) Create(_ context.Context, version *agentdefinition.DefinitionVersion) error {
	v.created = append(v.created, version)
	return nil
}

func (v *versionStore) List(
	context.Context,
	*repositories.ListAgentDefinitionVersionsRequest,
) ([]*agentdefinition.DefinitionVersion, error) {
	return v.created, nil
}

func (v *versionStore) Get(
	_ context.Context,
	req *repositories.GetAgentDefinitionVersionRequest,
) (*agentdefinition.DefinitionVersion, error) {
	version, ok := v.byNumber[req.Version]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent version")
	}
	return version, nil
}

func (v *versionStore) LatestAt(
	_ context.Context,
	req *repositories.GetAgentDefinitionVersionAtRequest,
) (*agentdefinition.DefinitionVersion, error) {
	var best *agentdefinition.DefinitionVersion
	for number, version := range v.byNumber {
		if number <= req.Version && (best == nil || number > best.Version) {
			best = version
		}
	}
	return best, nil
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestVersionedRecordsTheSaveItWraps(t *testing.T) {
	t.Parallel()

	versions := &versionStore{}
	svc := &Service{l: zap.NewNop(), db: dbtest.NopConnection{}, versions: versions}
	before := agentFixture()
	before.Version = 2
	after := agentFixture()
	after.Instructions = "Only pay rates."
	after.Version = 3
	author := pulid.MustNew("usr_")

	saved, err := svc.inTransaction(t.Context(), svc.versioned(
		func(context.Context) (*agentdefinition.Definition, error) { return after, nil },
		before,
		&serviceports.RequestActor{UserID: author},
	))

	require.NoError(t, err)
	assert.Same(t, after, saved)
	require.Len(t, versions.created, 1)
	assert.Equal(t, int64(3), versions.created[0].Version)
	assert.Equal(t, "Instructions", versions.created[0].Summary)
	assert.Equal(t, author, *versions.created[0].AuthorID)
}

func TestVersionedRecordsNothingWhenTheSaveFails(t *testing.T) {
	t.Parallel()

	versions := &versionStore{}
	svc := &Service{l: zap.NewNop(), db: dbtest.NopConnection{}, versions: versions}
	refused := errortypes.NewBusinessError("refused")

	_, err := svc.inTransaction(t.Context(), svc.versioned(
		func(context.Context) (*agentdefinition.Definition, error) { return nil, refused },
		nil,
		nil,
	))

	require.ErrorIs(t, err, refused)
	assert.Empty(t, versions.created)
}

func TestEditConflictSaysWhoSavedWhatSinceTheLoadedVersion(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenant()
	loaded := agentFixture()
	loaded.Version = 4
	current := agentFixture()
	current.ID = loaded.ID
	current.Version = 5
	current.Instructions = "Answer only about pay."
	current.UpdatedAt = 1_800_000_500
	other := &tenant.User{ID: pulid.MustNew("usr_"), Name: "Sarah Alvarez"}

	svc := &Service{
		l:    zap.NewNop(),
		repo: &stubDefinitionRepo{existing: current},
		versions: &versionStore{byNumber: map[int64]*agentdefinition.DefinitionVersion{
			4: {Version: 4, Snapshot: loaded},
			5: {Version: 5, Snapshot: current, Author: other},
		}},
	}

	err := svc.editConflict(t.Context(), &conflictRequest{
		tenantInfo: tenantInfo,
		agentID:    loaded.ID,
		loaded:     4,
		cause:      dberror.CreateVersionMismatchError("AgentDefinition", loaded.ID.String()),
	})

	edit, ok := errortypes.EditConflictOf(err)
	require.True(t, ok)
	assert.Equal(t, int64(5), edit.Version)
	assert.Equal(t, "Sarah Alvarez", edit.UpdatedByName)
	assert.Equal(t, int64(1_800_000_500), edit.UpdatedAt)
	assert.Equal(t, []errortypes.EditConflictChange{
		{Field: "instructions", Label: "Instructions"},
	}, edit.Changes)
	assert.True(t, errortypes.IsEditConflictError(err))
}

func TestEditConflictPassesOtherFailuresThrough(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}
	refused := errortypes.NewBusinessError("refused")

	err := svc.editConflict(t.Context(), &conflictRequest{cause: refused})

	assert.Same(t, refused, err)
}

func TestRestoreVersionIsADraftOnTheCurrentVersion(t *testing.T) {
	t.Parallel()

	current := agentFixture()
	current.Version = 9
	current.Instructions = "Current instructions."
	earlier := agentFixture()
	earlier.ID = current.ID
	earlier.Version = 2
	earlier.Instructions = "Earlier instructions."

	svc := &Service{
		l:    zap.NewNop(),
		repo: &stubDefinitionRepo{existing: current},
		versions: &versionStore{byNumber: map[int64]*agentdefinition.DefinitionVersion{
			2: {Version: 2, Snapshot: earlier},
		}},
	}

	draft, err := svc.RestoreVersion(t.Context(), &repositories.GetAgentDefinitionVersionRequest{
		TenantInfo:        testTenant(),
		AgentDefinitionID: current.ID,
		Version:           2,
	})

	require.NoError(t, err)
	assert.Equal(t, "Earlier instructions.", draft.Instructions)
	assert.Equal(t, int64(9), draft.Version, "saving the draft replaces the agent as it is now")
	assert.Equal(t, "Current instructions.", current.Instructions, "nothing was saved")
}

func agentFixture() *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:           pulid.MustNew("agdef_"),
		Name:         "Payroll desk",
		Instructions: "Answer payroll questions.",
	}
}
