package userservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *eventLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

type recordingTx struct {
	dbtest.NopConnection
	log *eventLog
}

func (c *recordingTx) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	txCtx, hooks := kernelports.WithAfterCommitHooks(ctx)
	c.log.add("begin")
	if err := fn(txCtx, bun.Tx{}); err != nil {
		c.log.add("rollback")
		return err
	}
	c.log.add("commit")
	hooks.Run(ctx)
	return nil
}

type fakeRoles struct {
	log      *eventLog
	requests []roleservice.AssignRoleRequest
	refuse   map[pulid.ID]error
}

func (f *fakeRoles) AssignRole(_ context.Context, req roleservice.AssignRoleRequest) error {
	f.requests = append(f.requests, req)
	if err, ok := f.refuse[req.Assignment.RoleID]; ok {
		return err
	}
	f.log.add("assign:" + req.Assignment.RoleID.String())
	return nil
}

type fakeInviter struct {
	log     *eventLog
	invited []passwordresetservice.InviteRequest
	err     error
}

func (f *fakeInviter) SendInvite(_ context.Context, req passwordresetservice.InviteRequest) error {
	f.invited = append(f.invited, req)
	f.log.add("invite")
	return f.err
}

type createDeps struct {
	*testDeps
	log      *eventLog
	roles    *fakeRoles
	invites  *fakeInviter
	security *securityaudittest.Recorder
	actor    pagination.TenantInfo
}

func setupCreateTest(t *testing.T) *createDeps {
	t.Helper()

	deps := setupTest(t)
	log := &eventLog{}
	d := &createDeps{
		testDeps: deps,
		log:      log,
		roles:    &fakeRoles{log: log, refuse: map[pulid.ID]error{}},
		invites:  &fakeInviter{log: log},
		security: &securityaudittest.Recorder{},
		actor: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	deps.svc.db = &recordingTx{log: log}
	deps.svc.roles = d.roles
	deps.svc.invites = d.invites
	deps.svc.auditor = d.security
	deps.svc.validator = NewTestValidatorWithRepository(deps.repo)

	return d
}

func newUserInput() *tenant.User {
	return &tenant.User{
		Name:         "  Dana Whitfield ",
		Username:     "dwhitfield",
		EmailAddress: " Dana.Whitfield@Example.com ",
		Timezone:     "America/Chicago",
		Status:       domaintypes.StatusActive,
	}
}

func (d *createDeps) expectNoConflicts() {
	d.repo.On("FindIdentityConflicts", mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{}, nil).Once()
}

func (d *createDeps) expectCreate() *tenantports.CreateUserRequest {
	captured := &tenantports.CreateUserRequest{}
	d.repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req tenantports.CreateUserRequest) *tenant.User {
			*captured = req
			req.User.ID = pulid.MustNew("usr_")
			req.Membership.UserID = req.User.ID
			d.log.add("create")
			return req.User
		}, func(context.Context, tenantports.CreateUserRequest) error {
			return nil
		}).Once()
	return captured
}

func TestCreate_CreatesTheUserMembershipAndRolesInOneTransaction(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	captured := d.expectCreate()
	var logged *auditports.LogActionParams
	d.audit.On("LogAction", mock.MatchedBy(func(p *auditports.LogActionParams) bool {
		logged = p
		return true
	}), mock.Anything).Return(nil).Once()

	dispatcher := pulid.MustNew("rol_")
	billing := pulid.MustNew("rol_")

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
		RoleIDs:    []pulid.ID{dispatcher, billing, dispatcher},
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.ID.IsNotNil())
	assert.Equal(t, "Dana Whitfield", created.Name)
	assert.Equal(t, "dana.whitfield@example.com", created.EmailAddress)
	assert.Equal(t, d.actor.OrgID, created.CurrentOrganizationID)
	assert.Equal(t, d.actor.BuID, created.BusinessUnitID)
	assert.Equal(t, domaintypes.TimeFormat12Hour, created.TimeFormat)
	assert.Equal(t, "en", created.Locale)
	assert.Equal(t, tenant.AvatarStyleAuto, created.AvatarStyle)
	assert.Equal(t, tenant.AvailabilityAvailable, created.AvailabilityStatus)
	assert.True(t, created.AwaitingAccountSetup())

	require.NotNil(t, captured.Membership)
	assert.Equal(t, d.actor.OrgID, captured.Membership.OrganizationID)
	assert.Equal(t, d.actor.BuID, captured.Membership.BusinessUnitID)
	assert.Equal(t, d.actor.UserID, captured.Membership.GrantedByID)
	assert.True(t, captured.Membership.IsDefault)

	require.Len(t, d.roles.requests, 2)
	for i, roleID := range []pulid.ID{dispatcher, billing} {
		req := d.roles.requests[i]
		assert.Equal(t, roleID, req.Assignment.RoleID)
		assert.Equal(t, created.ID, req.Assignment.UserID)
		assert.Equal(t, d.actor.UserID, req.ActorID)
		assert.Equal(t, d.actor.OrgID, req.OrganizationID)
	}

	assert.Equal(t, []string{
		"begin",
		"create",
		"assign:" + dispatcher.String(),
		"assign:" + billing.String(),
		"commit",
		"invite",
	}, d.log.all())

	require.NotNil(t, logged)
	assert.Equal(t, permission.OpCreate, logged.Operation)
	assert.Equal(t, created.ID.String(), logged.ResourceID)
	assert.Equal(t, d.actor.UserID, logged.UserID)
}

func TestCreate_StoresAPasswordNobodyKnows(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	captured := d.expectCreate()
	d.expectAuditLog()

	input := newUserInput()
	input.Password = "chosen-by-the-admin"

	_, err := d.svc.Create(t.Context(), &CreateUserRequest{TenantInfo: d.actor, User: input})

	require.NoError(t, err)
	stored := captured.User.Password
	require.NotEmpty(t, stored)
	assert.NotEqual(t, "chosen-by-the-admin", stored)
	_, costErr := bcrypt.Cost([]byte(stored))
	require.NoError(t, costErr, "the stored value is a bcrypt hash")
	assert.Error(t, bcrypt.CompareHashAndPassword([]byte(stored), []byte("chosen-by-the-admin")))
	assert.Nil(t, captured.User.PasswordChangedAt)
}

func TestCreate_RefusesADuplicateEmailAddress(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.repo.On("FindIdentityConflicts", mock.Anything, tenantports.UserIdentityConflictsRequest{
		EmailAddress: "dana.whitfield@example.com",
		Username:     "dwhitfield",
	}).Return(tenantports.UserIdentityConflicts{EmailAddress: true}, nil).Once()

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
	})

	require.Nil(t, created)
	assertFieldError(t, err, "emailAddress", errortypes.ErrDuplicate)
	d.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	assert.Empty(t, d.log.all())
	assert.Empty(t, d.invites.invited)
}

func TestCreate_RefusesADuplicateUsername(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.repo.On("FindIdentityConflicts", mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{Username: true}, nil).Once()

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
	})

	require.Nil(t, created)
	assertFieldError(t, err, "username", errortypes.ErrDuplicate)
	d.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreate_RollsBackWhenARoleCannotBeGranted(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()

	allowed := pulid.MustNew("rol_")
	forbidden := pulid.MustNew("rol_")
	refusal := errortypes.NewMultiError()
	refusal.Add(
		"permissions",
		errortypes.ErrForbidden,
		"cannot grant permissions for resource '{0}' that you don't have access to",
		"invoice",
	)
	d.roles.refuse[forbidden] = refusal

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
		RoleIDs:    []pulid.ID{allowed, forbidden},
	})

	require.Nil(t, created)
	assertFieldError(t, err, "roleIds", errortypes.ErrForbidden)
	assert.Equal(t, []string{"begin", "create", "assign:" + allowed.String(), "rollback"}, d.log.all())
	assert.Empty(t, d.invites.invited, "no invitation for a user that was never committed")
	assert.Empty(t, d.security.Changes())
	d.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestCreate_ReportsAMissingRoleOnTheRolesField(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()

	missing := pulid.MustNew("rol_")
	d.roles.refuse[missing] = errortypes.NewNotFoundError("Role not found")

	_, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
		RoleIDs:    []pulid.ID{missing},
	})

	assertFieldError(t, err, "roleIds", errortypes.ErrInvalidReference)
}

func TestCreate_RecordsASecurityAuditEntryAfterCommit(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()
	d.expectAuditLog()
	role := pulid.MustNew("rol_")

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
		RoleIDs:    []pulid.ID{role},
	})

	require.NoError(t, err)
	change := d.security.Only(t)
	assert.Equal(t, permission.ResourceUser, change.Resource)
	assert.Equal(t, permission.OpCreate, change.Operation)
	assert.Equal(t, created.ID.String(), change.ResourceID)
	assert.Equal(t, d.actor.UserID, change.Actor.UserID)
	assert.Equal(t, d.actor.OrgID, change.OrganizationID)
	assert.Equal(t, "User created", change.Comment)
	assert.Equal(t, true, change.Metadata["inviteSent"])
	after, ok := change.After.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{role.String()}, after["roleIds"])
}

func TestCreate_InvitesTheNewUserAfterCommit(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()
	d.expectAuditLog()

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
	})

	require.NoError(t, err)
	require.Len(t, d.invites.invited, 1)
	assert.Equal(t, created.ID, d.invites.invited[0].User.ID)
	assert.Equal(t, d.actor, d.invites.invited[0].Actor)
	events := d.log.all()
	assert.Equal(t, "invite", events[len(events)-1])
	assert.Contains(t, events, "commit")
}

func TestCreate_KeepsTheUserWhenTheInvitationCannotBeSent(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()
	d.expectAuditLog()
	d.invites.err = errors.New("mail is down")

	created, err := d.svc.Create(t.Context(), &CreateUserRequest{
		TenantInfo: d.actor,
		User:       newUserInput(),
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, false, d.security.Only(t).Metadata["inviteSent"])
}

func TestCreate_DoesNotInviteAnInactiveUser(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	d.expectNoConflicts()
	d.expectCreate()
	d.expectAuditLog()
	input := newUserInput()
	input.Status = domaintypes.StatusInactive

	_, err := d.svc.Create(t.Context(), &CreateUserRequest{TenantInfo: d.actor, User: input})

	require.NoError(t, err)
	assert.Empty(t, d.invites.invited)
}

func TestCreate_RefusesAnInputThatAlreadyHasAnID(t *testing.T) {
	t.Parallel()

	d := setupCreateTest(t)
	input := newUserInput()
	input.ID = pulid.MustNew("usr_")

	_, err := d.svc.Create(t.Context(), &CreateUserRequest{TenantInfo: d.actor, User: input})

	require.Error(t, err)
	d.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func assertFieldError(t *testing.T, err error, field string, code errortypes.ErrorCode) {
	t.Helper()

	multiErr, ok := errors.AsType[*errortypes.MultiError](err)
	require.True(t, ok, "expected a field error, got %v", err)
	for _, item := range multiErr.Errors {
		if item.Field == field && item.Code == code {
			return
		}
	}
	t.Fatalf("no %s error on %q in %v", code, field, multiErr.Errors)
}
