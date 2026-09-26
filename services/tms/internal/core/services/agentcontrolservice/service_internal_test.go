package agentcontrolservice

import (
	"context"
	"testing"

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
	f.updated = entity
	return entity, nil
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
		repo:        controls,
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
