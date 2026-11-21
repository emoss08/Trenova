package tenantlifecycle_test

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const now = int64(1_790_000_000)

type transactor struct{}

func (transactor) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type tenants struct {
	mu     sync.Mutex
	rows   map[string]cloudtenant.Tenant
	events []cloudtenant.Event
	locks  int
}

func (r *tenants) Lock(context.Context, string) error {
	r.locks++
	return nil
}

func (r *tenants) Get(_ context.Context, id string) (*cloudtenant.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.rows[id]
	if !ok {
		return nil, cloudtenantports.ErrNotFound
	}
	return &t, nil
}

func (r *tenants) Save(_ context.Context, t *cloudtenant.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[t.InstanceID] = *t
	return nil
}

func (r *tenants) List(
	_ context.Context,
	req *cloudtenantports.ListRequest,
) (*cloudtenantports.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.rows))
	for id := range r.rows {
		if id > req.After {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	page := &cloudtenantports.Page{}
	for i, id := range ids {
		if i == req.Limit {
			page.Next = ids[i-1]
			break
		}
		t := r.rows[id]
		page.Tenants = append(page.Tenants, &t)
	}
	return page, nil
}

func (r *tenants) ByInstances(_ context.Context, ids []string) ([]*cloudtenant.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*cloudtenant.Tenant, 0, len(ids))
	for _, id := range ids {
		if t, ok := r.rows[id]; ok {
			out = append(out, &t)
		}
	}
	return out, nil
}

func (r *tenants) InstanceOf(_ context.Context, org pulid.ID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, t := range r.rows {
		if t.OrganizationID == org {
			return id, nil
		}
	}
	return "", cloudtenantports.ErrNotHeld
}

func (r *tenants) Record(_ context.Context, e *cloudtenant.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, *e)
	return nil
}

func (r *tenants) kinds() []cloudtenant.EventKind {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]cloudtenant.EventKind, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, e.Kind)
	}
	return out
}

type administrators struct {
	taken map[string]bool
	setUp map[pulid.ID]bool
}

func (a *administrators) Facts(
	_ context.Context,
	reqs []cloudtenantports.AdministratorRequest,
) (map[pulid.ID]cloudtenantports.AdministratorFacts, error) {
	out := make(map[pulid.ID]cloudtenantports.AdministratorFacts, len(reqs))
	for _, req := range reqs {
		facts := cloudtenantports.AdministratorFacts{UserID: req.UserID, SetUp: a.setUp[req.UserID]}
		if !facts.SetUp && req.InviteToken != "" {
			facts.InviteLive, facts.InviteToken = true, req.InviteToken
		}
		out[req.UserID] = facts
	}
	return out, nil
}

func (a *administrators) EmailTaken(_ context.Context, email string) (bool, error) {
	return a.taken[email], nil
}

type members struct {
	ids []pulid.ID
}

func (m *members) Page(
	context.Context,
	dbscope.Tenant,
	pulid.ID,
	int,
) (*cloudtenantports.MembersPage, error) {
	page := &cloudtenantports.MembersPage{}
	for _, id := range m.ids {
		page.Members = append(page.Members, cloudtenantports.Member{UserID: id})
	}
	return page, nil
}

func (m *members) Reassign(context.Context, dbscope.Tenant, pulid.ID) (bool, error) {
	return false, nil
}

type bootstrapper struct {
	calls int
}

func (b *bootstrapper) LockProvisioning(context.Context) error { return nil }

func (b *bootstrapper) Bootstrap(
	_ context.Context,
	req *tenantports.BootstrapTenantRequest,
) (*tenantports.BootstrapTenantResult, error) {
	b.calls++
	bu := &tenant.BusinessUnit{ID: pulid.MustNew("bu_"), Name: req.BusinessUnitName}
	req.Organization.ID = pulid.MustNew("org_")
	req.Organization.BusinessUnitID = bu.ID
	req.Owner.ID = pulid.MustNew("usr_")
	req.Owner.Username = req.UsernameBase
	return &tenantports.BootstrapTenantResult{
		BusinessUnit: bu,
		Organization: req.Organization,
		Owner:        req.Owner,
		AdminRoleID:  pulid.MustNew("rol_"),
		SystemActor:  pulid.MustNew("usr_"),
	}, nil
}

type features struct {
	lists      map[string]*managedfeatures.Received
	lifecycles map[string]managedfeatures.Lifecycle
}

func (f *features) State(context.Context, managedscope.Scope) (*managedfeatures.State, error) {
	return nil, managedfeatures.ErrNoState
}

func (f *features) SetFeatures(
	_ context.Context,
	s managedscope.Scope,
	r *managedfeatures.Received,
) error {
	f.lists[s.Key()] = r
	return nil
}

func (f *features) SetLifecycle(
	_ context.Context,
	s managedscope.Scope,
	l managedfeatures.Lifecycle,
) error {
	f.lifecycles[s.Key()] = l
	return nil
}

type usage struct {
	states   map[string]*managedusage.State
	readings map[pulid.ID][]managedusageports.MeterReading
	asked    [][]pulid.ID
}

func (u *usage) SetState(_ context.Context, s managedscope.Scope, st *managedusage.State) error {
	u.states[s.Key()] = st
	return nil
}

func (u *usage) ReadingsForOrganizations(
	_ context.Context,
	ids []pulid.ID,
	_ int64,
) (map[pulid.ID][]managedusageports.MeterReading, error) {
	u.asked = append(u.asked, ids)
	out := make(map[pulid.ID][]managedusageports.MeterReading, len(ids))
	for _, id := range ids {
		out[id] = u.readings[id]
	}
	return out, nil
}

type invites struct {
	installed []string
}

func (i *invites) Install(
	context.Context,
	*instancebootstrapports.FirstAdministratorInvite,
) (*instancebootstrapports.FirstAdministratorInviteResult, error) {
	return nil, instancebootstrapports.ErrNotBootstrapped
}

func (i *invites) InstallFor(
	_ context.Context,
	_ pulid.ID,
	inv *instancebootstrapports.FirstAdministratorInvite,
) (*instancebootstrapports.FirstAdministratorInviteResult, error) {
	i.installed = append(i.installed, inv.TokenSHA256)
	return &instancebootstrapports.FirstAdministratorInviteResult{
		InviteTokenSHA256: inv.TokenSHA256,
	}, nil
}

type sessions struct {
	sessionports.SessionRepository
	ended []pulid.ID
}

func (s *sessions) ListForUser(_ context.Context, user pulid.ID) ([]*session.Session, error) {
	return []*session.Session{{ID: pulid.MustNew("ses_"), UserID: user}}, nil
}

func (s *sessions) DeleteAllForUser(_ context.Context, user pulid.ID) error {
	s.ended = append(s.ended, user)
	return nil
}

type workflows struct {
	kernelports.WorkflowStarter
	started []string
}

func (w *workflows) StartWorkflow(
	_ context.Context,
	opts client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	w.started = append(w.started, opts.ID)
	return nil, nil
}

type commands struct {
	mu    sync.Mutex
	rows  map[string]tenantcommand.Command
	grant *tenantcommand.Grant
}

func (c *commands) Get(_ context.Context, id string) (*tenantcommand.Command, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cmd, ok := c.rows[id]
	if !ok {
		return nil, cloudtenantports.ErrCommandNotFound
	}
	return &cmd, nil
}

func (c *commands) Create(
	_ context.Context,
	cmd *tenantcommand.Command,
) (*tenantcommand.Command, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if held, ok := c.rows[cmd.CommandID]; ok {
		return &held, false, nil
	}
	c.rows[cmd.CommandID] = *cmd
	return cmd, true, nil
}

func (c *commands) Save(_ context.Context, cmd *tenantcommand.Command) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows[cmd.CommandID] = *cmd
	return nil
}

func (c *commands) Open(_ context.Context, instanceID string) ([]*tenantcommand.Command, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*tenantcommand.Command, 0)
	for _, cmd := range c.rows {
		if cmd.InstanceID == instanceID && !cmd.State.Settled() {
			out = append(out, &cmd)
		}
	}
	return out, nil
}

func (c *commands) Failures(_ context.Context, ids []string) ([]*tenantcommand.Command, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*tenantcommand.Command, 0)
	for _, cmd := range c.rows {
		if slices.Contains(ids, cmd.InstanceID) && cmd.State == tenantcommand.StateFailed &&
			cmd.Kind != tenantcommand.KindRestore {
			out = append(out, &cmd)
		}
	}
	return out, nil
}

func (c *commands) Progress(_ context.Context, id string, p *tenantcommand.Progress) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cmd := c.rows[id]
	cmd.Progress = *p
	c.rows[id] = cmd
	return nil
}

func (c *commands) SetGrant(_ context.Context, g *tenantcommand.Grant) error {
	c.grant = g
	return nil
}

func (c *commands) Grant(context.Context, string) (*tenantcommand.Grant, error) {
	if c.grant == nil {
		return nil, cloudtenantports.ErrGrantMissing
	}
	return c.grant, nil
}

type cipher struct{}

func (cipher) Seal(
	_ context.Context,
	plain string,
	scope *kernelports.SecretScope,
) (string, error) {
	return "sealed:" + scope.ResourceID + ":" + base64.StdEncoding.EncodeToString(
		[]byte(plain),
	), nil
}

func (cipher) Open(
	_ context.Context,
	sealed string,
	scope *kernelports.SecretScope,
) (string, error) {
	body, ok := strings.CutPrefix(sealed, "sealed:"+scope.ResourceID+":")
	if !ok {
		return "", errors.New("sealed for another resource")
	}
	plain, err := base64.StdEncoding.DecodeString(body)
	return string(plain), err
}

type rig struct {
	commands  *commands
	svc       *tenantlifecycle.Service
	tenants   *tenants
	admins    *administrators
	bootstrap *bootstrapper
	features  *features
	usage     *usage
	invites   *invites
	sessions  *sessions
	workflows *workflows
	audit     *securityaudittest.Recorder
	members   *members
	grants    *liveGrants
}

type liveGrants struct {
	mu    sync.Mutex
	live  []*supportaccessports.LiveGrant
	asked []supportaccessports.ListLiveRequest
}

func (g *liveGrants) ListLive(
	_ context.Context,
	req *supportaccessports.ListLiveRequest,
) ([]*supportaccessports.LiveGrant, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.asked = append(g.asked, *req)
	out := make([]*supportaccessports.LiveGrant, 0)
	for _, l := range g.live {
		if slices.Contains(req.OrganizationIDs, l.Grant.OrganizationID) {
			out = append(out, l)
		}
	}
	return out, nil
}

func newRig(t *testing.T, shared bool) *rig {
	t.Helper()
	tenancy := managedconfig.TenancyDedicated
	if shared {
		tenancy = managedconfig.TenancyShared
	}
	r := &rig{
		tenants:   &tenants{rows: map[string]cloudtenant.Tenant{}},
		admins:    &administrators{taken: map[string]bool{}, setUp: map[pulid.ID]bool{}},
		bootstrap: &bootstrapper{},
		features: &features{
			lists:      map[string]*managedfeatures.Received{},
			lifecycles: map[string]managedfeatures.Lifecycle{},
		},
		usage: &usage{
			states:   map[string]*managedusage.State{},
			readings: map[pulid.ID][]managedusageports.MeterReading{},
		},
		invites:   &invites{},
		sessions:  &sessions{},
		workflows: &workflows{},
		audit:     &securityaudittest.Recorder{},
		members:   &members{},
		commands:  &commands{rows: map[string]tenantcommand.Command{}},
		grants:    &liveGrants{},
	}
	r.svc = tenantlifecycle.New(tenantlifecycle.Params{
		Config:         &managedconfig.Config{Enabled: true, Tenancy: tenancy},
		DB:             transactor{},
		Tenants:        r.tenants,
		Administrators: r.admins,
		Members:        r.members,
		Bootstrap:      r.bootstrap,
		Features:       r.features,
		Usage:          r.usage,
		Readings:       r.usage,
		Invites:        r.invites,
		Sessions:       r.sessions,
		Auditor:        r.audit,
		Workflows:      r.workflows,
		Commands:       r.commands,
		Grants:         r.grants,
		Secrets:        cipher{},
		Scopes:         managedscope.NewResolver(shared),
		Clock:          func() int64 { return now },
		Logger:         zap.NewNop(),
	})
	return r
}
