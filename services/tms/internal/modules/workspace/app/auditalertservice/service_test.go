package auditalertservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRecipients struct {
	users []permissionports.PermittedUser
	req   permissionports.ListUsersWithPermissionRequest
}

func (f *fakeRecipients) ListUsersWithPermission(
	_ context.Context,
	req permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	f.req = req
	return f.users, nil
}

type fakeNotifications struct {
	notified []pulid.ID
	created  []*notification.Notification
	failOn   int
	query    notificationports.ExistsRecentNotificationRequest
}

func (f *fakeNotifications) NotifiedUsers(
	_ context.Context,
	req notificationports.ExistsRecentNotificationRequest,
) ([]pulid.ID, error) {
	f.query = req
	return f.notified, nil
}

func (f *fakeNotifications) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	if f.failOn > 0 && len(f.created)+1 == f.failOn {
		return nil, errors.New("notification store unavailable")
	}
	f.created = append(f.created, entity)
	f.notified = append(f.notified, *entity.TargetUserID)
	return entity, nil
}

type recordingAuditor struct {
	changes []*permissionports.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	a.changes = append(a.changes, change)
}

type harness struct {
	svc           *Service
	recipients    *fakeRecipients
	notifications *fakeNotifications
	auditor       *recordingAuditor
	policies      *auditportsmock.MockAlertPolicyRepository
}

func newHarness(t *testing.T, recipients int) *harness {
	t.Helper()

	users := make([]permissionports.PermittedUser, 0, recipients)
	for range recipients {
		users = append(users, permissionports.PermittedUser{UserID: pulid.MustNew("usr_")})
	}
	h := &harness{
		recipients:    &fakeRecipients{users: users},
		notifications: &fakeNotifications{},
		auditor:       &recordingAuditor{},
		policies:      auditportsmock.NewMockAlertPolicyRepository(t),
	}
	h.svc = &Service{
		policies:      h.policies,
		recipients:    h.recipients,
		notifications: h.notifications,
		auditor:       h.auditor,
		l:             zap.NewNop(),
	}
	return h
}

func roleChangedPayload() *auditports.AlertPayload {
	return &auditports.AlertPayload{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Kind:          audit.AlertRoleChanged,
		EntryID:       pulid.MustNew("ae_"),
		Action:        "role.assigned",
		Resource:      permission.ResourceRole,
		ResourceID:    "rol_1",
		PrincipalID:   pulid.MustNew("usr_"),
		PrincipalType: "session_user",
		ActorName:     "Dana Ruiz",
		EntityRef:     "Dispatcher",
		OccurredAt:    1_800_000_000,
	}
}

func TestNotify_TellsEverySecurityReviewerAndRecordsTheAlertOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3)
	payload := roleChangedPayload()

	sent, err := h.svc.Notify(t.Context(), payload)

	require.NoError(t, err)
	assert.Equal(t, 3, sent)
	assert.Equal(t, permission.ResourceSecurityAuditLog, h.recipients.req.Resource)
	assert.Equal(t, permission.OpRead, h.recipients.req.Operation)
	require.Len(t, h.notifications.created, 3)
	first := h.notifications.created[0]
	assert.Equal(t, "audit.alert.RoleChanged", first.EventType)
	assert.Equal(t, payload.EntryID.String(), *first.CorrelationID)
	assert.Equal(t, notification.PriorityHigh, first.Priority)
	assert.Equal(t, "/admin/audit-logs?entry="+payload.EntryID.String(), first.Data["link"])
	assert.Contains(t, first.Message, "(Dispatcher, by Dana Ruiz).")

	require.Len(t, h.auditor.changes, 1)
	raised := h.auditor.changes[0]
	assert.Equal(t, audit.ActionAlertRaised, raised.Action)
	assert.Equal(t, string(audit.AlertRoleChanged), raised.ResourceID)
	assert.Equal(t, payload.TenantInfo.OrgID, raised.OrganizationID)
}

func TestNotify_ARetryTellsOnlyTheReviewersNotYetToldAndRecordsNothingAgain(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3)
	h.notifications.failOn = 2
	payload := roleChangedPayload()

	_, err := h.svc.Notify(t.Context(), payload)
	require.Error(t, err)
	require.Len(t, h.notifications.created, 1)
	assert.Empty(t, h.auditor.changes, "a run that failed part way records no alert")

	h.notifications.failOn = 0
	sent, err := h.svc.Notify(t.Context(), payload)

	require.NoError(t, err)
	assert.Equal(t, 2, sent)
	require.Len(t, h.notifications.created, 3)
	seen := map[pulid.ID]int{}
	for _, created := range h.notifications.created {
		seen[*created.TargetUserID]++
	}
	for _, user := range h.recipients.users {
		assert.Equal(t, 1, seen[user.UserID], "each reviewer is told once")
	}
	assert.Equal(t, payload.OccurredAt, h.notifications.query.Since)
	assert.Equal(t, payload.EntryID.String(), h.notifications.query.CorrelationID)
	assert.Empty(t, h.auditor.changes, "the retry finds reviewers already told and does not record the alert twice")
}

func TestNotify_WithNoReviewersStillRecordsTheAlert(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 0)

	sent, err := h.svc.Notify(t.Context(), roleChangedPayload())

	require.NoError(t, err)
	assert.Zero(t, sent)
	require.Len(t, h.auditor.changes, 1)
	assert.Equal(t, 0, h.auditor.changes[0].After.(map[string]any)["recipients"])
}

func TestNotify_RefusesAnUnknownKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	payload := roleChangedPayload()
	payload.Kind = "Unknown"

	_, err := h.svc.Notify(t.Context(), payload)

	require.Error(t, err)
	assert.Empty(t, h.notifications.created)
}

func TestNotify_ACountedAlertSaysHowManyTimes(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	payload := roleChangedPayload()
	payload.Kind = audit.AlertBulkDelete
	payload.Count = 51

	_, err := h.svc.Notify(t.Context(), payload)

	require.NoError(t, err)
	require.Len(t, h.notifications.created, 1)
	assert.Contains(t, h.notifications.created[0].Message, "Dana Ruiz did this 51 times")
}

func TestUpdatePolicy_SavesTheChoiceAndRecordsItAsASecurityChange(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 0)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	h.policies.EXPECT().ListForTenant(mock.Anything, tenant).Return(nil, nil).Once()
	h.policies.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*audit.AlertPolicy")).
		RunAndReturn(func(_ context.Context, policy *audit.AlertPolicy) (*audit.AlertPolicy, error) {
			return policy, nil
		}).
		Once()

	view, err := h.svc.UpdatePolicy(t.Context(), &UpdatePolicyRequest{
		TenantInfo:    tenant,
		Kind:          audit.AlertBulkDelete,
		Enabled:       true,
		Threshold:     200,
		WindowMinutes: 30,
	})

	require.NoError(t, err)
	assert.Equal(t, 200, view.Policy.Threshold)
	assert.Equal(t, 30, view.Policy.WindowMinutes)
	require.Len(t, h.auditor.changes, 1)
	change := h.auditor.changes[0]
	assert.Equal(t, audit.ActionAlertPolicyChanged, change.Action)
	before := change.Before.(*audit.AlertPolicy)
	assert.Equal(t, 50, before.Threshold, "the change records the default it replaced")
}

func TestUpdatePolicy_RefusesAThresholdOfOneForACountedAlert(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 0)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	h.policies.EXPECT().ListForTenant(mock.Anything, tenant).Return(nil, nil).Once()

	_, err := h.svc.UpdatePolicy(t.Context(), &UpdatePolicyRequest{
		TenantInfo:    tenant,
		Kind:          audit.AlertBulkDelete,
		Enabled:       true,
		Threshold:     1,
		WindowMinutes: 10,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, h.auditor.changes)
}
