package agentcontrolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/dberror"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const consentChangedAt = int64(1_790_000_000)

type fakeControls struct {
	control *tenant.AgentControl
	updated *tenant.AgentControl
}

func (f *fakeControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	current := *f.control
	return &current, nil
}

func (f *fakeControls) Update(
	_ context.Context,
	entity *tenant.AgentControl,
) (*tenant.AgentControl, error) {
	if entity.Version != f.control.Version {
		return nil, dberror.CreateVersionMismatchError("AgentControl", entity.ID.String())
	}
	entity.Version++
	f.updated = entity
	return entity, nil
}

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

type absentDefinitions struct {
	repositories.AgentDefinitionRepository
}

func (absentDefinitions) GetBySystemKey(
	context.Context,
	repositories.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	return nil, errortypes.NewNotFoundError("agent definition not found")
}

func newTestService(control *tenant.AgentControl) (*Service, *fakeControls) {
	controls := &fakeControls{control: control}
	return &Service{
		l:           zap.NewNop(),
		db:          dbtest.NopConnection{},
		repo:        controls,
		versions:    &settingVersions{},
		definitions: absentDefinitions{},
		audit:       &mocks.NoopAuditService{},
		now:         func() int64 { return consentChangedAt },
	}, controls
}

func boolPtr(v bool) *bool { return &v }

func defaultControl() *tenant.AgentControl {
	return &tenant.AgentControl{
		ID:                 pulid.ID("agc_test"),
		OrganizationID:     pulid.ID("org_test"),
		BusinessUnitID:     pulid.ID("bu_test"),
		PromotionThreshold: tenant.DefaultPromotionThreshold,
		BriefingHourLocal:  tenant.DefaultBriefingHourLocal,
	}
}

func userActor(id pulid.ID) *services.RequestActor {
	return &services.RequestActor{PrincipalType: services.PrincipalTypeUser, PrincipalID: id, UserID: id}
}

func TestUpdate_GrantingTrainingConsentRecordsWhoAndWhen(t *testing.T) {
	t.Parallel()

	svc, controls := newTestService(defaultControl())
	userID := pulid.ID("usr_admin")

	updated, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		AITrainingConsent: boolPtr(true),
	}, userActor(userID))

	require.NoError(t, err)
	require.Same(t, updated, controls.updated)
	assert.True(t, updated.AITrainingConsent)
	require.NotNil(t, updated.AITrainingConsentChangedAt)
	assert.Equal(t, consentChangedAt, *updated.AITrainingConsentChangedAt)
	require.NotNil(t, updated.AITrainingConsentChangedByID)
	assert.Equal(t, userID, *updated.AITrainingConsentChangedByID)
}

func TestUpdate_LeavesTrainingConsentAloneWhenNotSent(t *testing.T) {
	t.Parallel()

	changedAt := int64(1_700_000_000)
	changedBy := pulid.ID("usr_first")
	control := defaultControl()
	control.AITrainingConsent = true
	control.AITrainingConsentChangedAt = &changedAt
	control.AITrainingConsentChangedByID = &changedBy
	svc, _ := newTestService(control)

	updated, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		ShadowMode: true,
	}, userActor(pulid.ID("usr_other")))

	require.NoError(t, err)
	assert.True(t, updated.AITrainingConsent)
	assert.Equal(t, changedAt, *updated.AITrainingConsentChangedAt)
	assert.Equal(t, changedBy, *updated.AITrainingConsentChangedByID)
}

func TestUpdate_SendingTheSameConsentDoesNotRestampIt(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(defaultControl())

	updated, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		AITrainingConsent: boolPtr(false),
	}, &services.RequestActor{PrincipalType: services.PrincipalTypeAPIKey, APIKeyID: "key_1"})

	require.NoError(t, err)
	assert.False(t, updated.AITrainingConsent)
	assert.Nil(t, updated.AITrainingConsentChangedAt)
	assert.Nil(t, updated.AITrainingConsentChangedByID)
}

func TestUpdate_RefusesAConsentChangeWithoutAPerson(t *testing.T) {
	t.Parallel()

	svc, controls := newTestService(defaultControl())

	_, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		AITrainingConsent: boolPtr(true),
	}, &services.RequestActor{PrincipalType: services.PrincipalTypeAPIKey, APIKeyID: "key_1"})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiTrainingConsent", multiErr.Errors[0].Field)
	assert.Nil(t, controls.updated)
}

func TestUpdate_RefusesAConsentChangeFromAnAgentCarryingTheSystemUser(t *testing.T) {
	t.Parallel()

	svc, controls := newTestService(defaultControl())

	_, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		AITrainingConsent: boolPtr(true),
	}, &services.RequestActor{
		PrincipalType: services.PrincipalTypeAgent,
		PrincipalID:   pulid.ID("agdef_dispatch"),
		UserID:        pulid.ID("usr_system"),
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiTrainingConsent", multiErr.Errors[0].Field)
	assert.Nil(t, controls.updated, "the system account an agent carries is not a person")
}

func TestUpdate_WithdrawingConsentIsRecorded(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.AITrainingConsent = true
	svc, _ := newTestService(control)
	userID := pulid.ID("usr_admin")

	updated, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		AITrainingConsent: boolPtr(false),
	}, userActor(userID))

	require.NoError(t, err)
	assert.False(t, updated.AITrainingConsent)
	assert.Equal(t, userID, *updated.AITrainingConsentChangedByID)
}

func TestAgentControlAuditComment(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "AI training consent granted",
		agentControlAuditComment(true, &tenant.AgentControl{AITrainingConsent: true}))
	assert.Equal(t, "AI training consent withdrawn",
		agentControlAuditComment(true, &tenant.AgentControl{}))
	assert.Equal(t, "Agent control updated",
		agentControlAuditComment(false, &tenant.AgentControl{AITrainingConsent: true}))
}

func TestUpdateRecordsTheVersionTheControlsBecome(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(defaultControl())
	author := pulid.MustNew("usr_")

	updated, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		ShadowMode: true,
		TenantInfo: pagination.TenantInfo{OrgID: "org_test", BuID: "bu_test"},
	}, userActor(author))

	require.NoError(t, err)
	versions := svc.versions.(*settingVersions).created
	require.Len(t, versions, 1)
	assert.Equal(t, settingversion.KindAgentControl, versions[0].Kind)
	assert.Equal(t, updated.Version, versions[0].Version)
	assert.Equal(t, author, *versions[0].AuthorID)
	assert.Equal(t, true, versions[0].Snapshot["shadowMode"])
}

func TestUpdateOnAStaleVersionSaysWhoChangedWhat(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(defaultControl())
	tenantInfo := pagination.TenantInfo{OrgID: "org_test", BuID: "bu_test"}
	_, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		ShadowMode: false,
		Version:    int64Ptr(0),
		TenantInfo: tenantInfo,
	}, userActor(pulid.MustNew("usr_")))
	require.NoError(t, err)

	svc.repo.(*fakeControls).control.Version = 1
	svc.repo.(*fakeControls).control.LearningOff = true
	versions := svc.versions.(*settingVersions)
	versions.created = append(versions.created, &settingversion.SettingVersion{
		Kind:     settingversion.KindAgentControl,
		Version:  1,
		Snapshot: map[string]any{"learningOff": true},
		Author:   &tenant.User{Name: "Sarah Alvarez"},
	})
	versions.created[0].Version = 0

	_, err = svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		ShadowMode: true,
		Version:    int64Ptr(0),
		TenantInfo: tenantInfo,
	}, userActor(pulid.MustNew("usr_")))

	edit, ok := errortypes.EditConflictOf(err)
	require.True(t, ok)
	assert.Equal(t, int64(1), edit.Version)
	assert.Equal(t, "Sarah Alvarez", edit.UpdatedByName)
	assert.Equal(t, []errortypes.EditConflictChange{
		{Field: "learningOff", Label: "Learn from their work"},
	}, edit.Changes)
}

func int64Ptr(v int64) *int64 { return &v }
