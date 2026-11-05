package scimservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/directoryaccounts"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeRepo struct {
	scimports.Repository

	credential *scimports.Credential
	prefix     string
	touched    []pulid.ID
	records    map[pulid.ID]*scimports.UserRecord
	member     *tenant.User
	links      []*scim.User
	saved      []*scim.User
	deleted    []pulid.ID
	pending    []pulid.ID
	sources    []*scim.RoleSource
	grants     []*scim.RoleGrant
	created    []*scim.RoleGrant
	completed  []*scim.User
	recorded   []*iam.ProvisioningAuditRecord
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{records: make(map[pulid.ID]*scimports.UserRecord)}
}

func (f *fakeRepo) FindCredential(_ context.Context, prefix string) (*scimports.Credential, error) {
	if f.credential == nil || prefix != f.prefix {
		return nil, errortypes.NewNotFoundError("SCIM token not found")
	}
	return f.credential, nil
}

func (f *fakeRepo) TouchCredential(_ context.Context, tokenID pulid.ID, _ int64) error {
	f.touched = append(f.touched, tokenID)
	return nil
}

func (f *fakeRepo) RecordProvisioning(
	_ context.Context,
	record *iam.ProvisioningAuditRecord,
) error {
	f.recorded = append(f.recorded, record)
	return nil
}

func (f *fakeRepo) GetUser(
	_ context.Context,
	directoryID, userID pulid.ID,
) (*scimports.UserRecord, error) {
	record, ok := f.records[userID]
	if !ok || record.Link.DirectoryID != directoryID {
		return nil, errortypes.NewNotFoundError("SCIM user not found")
	}
	copied := *record.Link
	return &scimports.UserRecord{Link: &copied, User: record.User}, nil
}

func (f *fakeRepo) FindMember(_ context.Context, _ string) (*tenant.User, error) {
	if f.member == nil {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	return f.member, nil
}

func (f *fakeRepo) CreateUser(_ context.Context, link *scim.User) error {
	f.links = append(f.links, link)
	f.records[link.UserID] = &scimports.UserRecord{Link: link}
	return nil
}

func (f *fakeRepo) SaveUser(_ context.Context, link *scim.User) error {
	f.saved = append(f.saved, link)
	if record, ok := f.records[link.UserID]; ok {
		copied := *link
		record.Link = &copied
	}
	return nil
}

func (f *fakeRepo) DeleteUser(_ context.Context, _ pulid.ID, userID pulid.ID) error {
	f.deleted = append(f.deleted, userID)
	delete(f.records, userID)
	return nil
}

func (f *fakeRepo) MarkPending(_ context.Context, _ pulid.ID, userIDs []pulid.ID) error {
	f.pending = append(f.pending, userIDs...)
	return nil
}

func (f *fakeRepo) RoleSources(_ context.Context, _, _ pulid.ID) ([]*scim.RoleSource, error) {
	return f.sources, nil
}

func (f *fakeRepo) ListGrants(_ context.Context, _, userID pulid.ID) ([]*scim.RoleGrant, error) {
	held := make([]*scim.RoleGrant, 0, len(f.grants))
	for _, grant := range f.grants {
		if grant.UserID == userID {
			held = append(held, grant)
		}
	}
	return held, nil
}

func (f *fakeRepo) CreateGrant(_ context.Context, grant *scim.RoleGrant) error {
	f.created = append(f.created, grant)
	return nil
}

func (f *fakeRepo) CompleteReconcile(_ context.Context, link *scim.User) error {
	f.completed = append(f.completed, link)
	return nil
}

func (f *fakeRepo) statuses() []string {
	statuses := make([]string, 0, len(f.recorded))
	for _, record := range f.recorded {
		statuses = append(statuses, record.Status)
	}
	return statuses
}

type fakeMembers struct {
	tenantports.UserRepository

	members map[pulid.ID]*tenant.User
}

func (f *fakeMembers) GetTenantMember(
	_ context.Context,
	req tenantports.GetTenantMemberRequest,
) (*tenant.User, error) {
	member, ok := f.members[req.UserID]
	if !ok {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	return member, nil
}

type fakeAccounts struct {
	created  []*directoryaccounts.CreateRequest
	updated  []*directoryaccounts.UpdateRequest
	revoked  []*directoryaccounts.AccessRequest
	restored []*directoryaccounts.AccessRequest
}

func (f *fakeAccounts) Create(
	_ context.Context,
	req *directoryaccounts.CreateRequest,
) (*tenant.User, error) {
	f.created = append(f.created, req)
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        req.Actor.BusinessUnitID,
		CurrentOrganizationID: req.Actor.OrganizationID,
		Name:                  req.Profile.Name,
		EmailAddress:          req.Profile.EmailAddress,
		Status:                domaintypes.StatusActive,
	}, nil
}

func (f *fakeAccounts) Update(
	_ context.Context,
	req *directoryaccounts.UpdateRequest,
) (*directoryaccounts.UpdateResult, error) {
	f.updated = append(f.updated, req)
	return &directoryaccounts.UpdateResult{User: &tenant.User{ID: req.UserID}}, nil
}

func (f *fakeAccounts) Revoke(_ context.Context, req *directoryaccounts.AccessRequest) error {
	f.revoked = append(f.revoked, req)
	return nil
}

func (f *fakeAccounts) Restore(_ context.Context, req *directoryaccounts.AccessRequest) error {
	f.restored = append(f.restored, req)
	return nil
}

type fakeRoles struct {
	refuse     map[[2]pulid.ID]error
	assignErr  error
	assigned   []roleservice.AssignRoleRequest
	unassigned []roleservice.UnassignRoleRequest
	checked    [][2]pulid.ID
}

func (f *fakeRoles) AssignRole(_ context.Context, req roleservice.AssignRoleRequest) error {
	if f.assignErr != nil {
		return f.assignErr
	}
	req.Assignment.ID = pulid.MustNew("ura_")
	f.assigned = append(f.assigned, req)
	return nil
}

func (f *fakeRoles) UnassignRole(_ context.Context, req roleservice.UnassignRoleRequest) error {
	f.unassigned = append(f.unassigned, req)
	return nil
}

func (f *fakeRoles) RequireCanGrantRole(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	roleID pulid.ID,
) error {
	key := [2]pulid.ID{tenantInfo.UserID, roleID}
	f.checked = append(f.checked, key)
	return f.refuse[key]
}

type fakeRequester struct {
	requested []pulid.ID
}

func (f *fakeRequester) RequestReconcile(
	_ context.Context,
	_ pagination.TenantInfo,
	directoryID pulid.ID,
) {
	f.requested = append(f.requested, directoryID)
}

type harness struct {
	svc       *Service
	repo      *fakeRepo
	members   *fakeMembers
	accounts  *fakeAccounts
	roles     *fakeRoles
	requester *fakeRequester
	auditor   *securityaudittest.Recorder
	dir       *scimports.Directory
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		repo:      newFakeRepo(),
		members:   &fakeMembers{members: make(map[pulid.ID]*tenant.User)},
		accounts:  &fakeAccounts{},
		roles:     &fakeRoles{refuse: make(map[[2]pulid.ID]error)},
		requester: &fakeRequester{},
		auditor:   &securityaudittest.Recorder{},
		dir: &scimports.Directory{
			ID:             pulid.MustNew("scd_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			TokenID:        pulid.MustNew("sct_"),
			TenantSlug:     "acme",
		},
	}
	h.svc = &Service{
		l:         zap.NewNop(),
		repo:      h.repo,
		users:     h.members,
		accounts:  h.accounts,
		roles:     h.roles,
		requester: h.requester,
		auditor:   h.auditor,
		db:        dbtest.NopConnection{},
	}

	return h
}

func (h *harness) activeMember() *tenant.User {
	user := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        h.dir.BusinessUnitID,
		CurrentOrganizationID: h.dir.OrganizationID,
		Name:                  "Ada Lovelace",
		EmailAddress:          "ada@example.com",
		Status:                domaintypes.StatusActive,
	}
	h.members.members[user.ID] = user
	return user
}

func (h *harness) link(user *tenant.User, active bool) *scim.User {
	link := &scim.User{
		ID:             pulid.MustNew("scu_"),
		OrganizationID: h.dir.OrganizationID,
		BusinessUnitID: h.dir.BusinessUnitID,
		DirectoryID:    h.dir.ID,
		UserID:         user.ID,
		UserName:       user.EmailAddress,
		Active:         active,
		RolesVersion:   1,
	}
	h.repo.records[user.ID] = &scimports.UserRecord{Link: link, User: user}
	return link
}
