package supportaccessservice

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/internal/testutil/sessiontest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testInstance = "inst_01JTESTINSTANCE"
	testDomain   = "trenova.app"
	testKeyID    = "cp-2026-11"
	staffEmail   = "jane.doe@trenova.app"
)

type grantStore struct {
	mu     sync.Mutex
	calls  int
	grants map[pulid.ID]*supportaccess.Grant
	users  map[pulid.ID]string
	orgs   map[pulid.ID]pulid.ID
}

func newGrantStore() *grantStore {
	return &grantStore{
		grants: map[pulid.ID]*supportaccess.Grant{},
		users:  map[pulid.ID]string{},
		orgs:   map[pulid.ID]pulid.ID{},
	}
}

func (s *grantStore) touch() {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
}

func (s *grantStore) Create(_ context.Context, req *supportaccessports.CreateGrantRequest) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	live := 0
	for _, g := range s.grants {
		if g.OrganizationID != req.Grant.OrganizationID || !g.LiveAt(req.Now) {
			continue
		}
		if g.StaffEmail == req.Grant.StaffEmail {
			return nil, errortypes.NewValidationError("staffEmail", errortypes.ErrDuplicate, "already has access")
		}
		live++
	}
	if live >= supportaccess.MaxLiveGrants {
		return nil, errortypes.NewValidationError("staffEmail", errortypes.ErrInvalid, "too many")
	}
	grant := *req.Grant
	grant.ID = pulid.MustNew(supportaccess.GrantIDPrefix)
	s.grants[grant.ID] = &grant
	s.users[req.Principal.ID] = req.Principal.Name
	out := grant
	return &out, nil
}

func (s *grantStore) Get(_ context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[id]
	if !ok || g.OrganizationID != tenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("Support access grant not found within your organization")
	}
	out := *g
	return &out, nil
}

func (s *grantStore) List(context.Context, *supportaccessports.ListGrantsRequest) (*pagination.CursorListResult[*supportaccess.Grant], error) {
	s.touch()
	return &pagination.CursorListResult[*supportaccess.Grant]{}, nil
}

func (s *grantStore) Revoke(_ context.Context, grant *supportaccess.Grant, now int64) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grant.ID]
	if !ok || !g.LiveAt(now) {
		return nil, supportaccess.ErrGrantNotFound
	}
	g.RevokedAt = &now
	g.RevokedByID = grant.RevokedByID
	g.RevokeReason = grant.RevokeReason
	out := *g
	return &out, nil
}

func (s *grantStore) Extend(_ context.Context, grant *supportaccess.Grant, now int64) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grant.ID]
	if !ok || !g.LiveAt(now) {
		return nil, supportaccess.ErrGrantNotFound
	}
	g.ExpiresAt = grant.ExpiresAt
	g.DurationHours = grant.DurationHours
	g.ExtendedAt = &now
	g.ExtendedByID = grant.ExtendedByID
	out := *g
	return &out, nil
}

func (s *grantStore) ByPrincipal(_ context.Context, tenantInfo pagination.TenantInfo, userID pulid.ID) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.grants {
		if g.PrincipalUserID == userID && g.OrganizationID == tenantInfo.OrgID {
			out := *g
			return &out, nil
		}
	}
	return nil, supportaccess.ErrGrantNotFound
}

func (s *grantStore) FindLiveForEntry(_ context.Context, req *supportaccessports.FindLiveGrantRequest) (*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.grants {
		if g.OrganizationID == req.OrganizationID && g.StaffEmail == req.StaffEmail && g.LiveAt(req.Now) {
			out := *g
			return &out, nil
		}
	}
	return nil, supportaccess.ErrGrantNotFound
}

func (s *grantStore) ListLiveForStaff(
	_ context.Context,
	req *supportaccessports.ListLiveForStaffRequest,
) ([]*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*supportaccess.Grant
	for _, g := range s.grants {
		if g.StaffEmail == req.StaffEmail && g.LiveAt(req.Now) && len(out) < req.Limit {
			copied := *g
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (s *grantStore) ListLive(
	_ context.Context,
	req *supportaccessports.ListLiveRequest,
) ([]*supportaccessports.LiveGrant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*supportaccessports.LiveGrant{}
	for _, g := range s.grants {
		if g.LiveAt(req.Now) {
			copied := *g
			out = append(out, &supportaccessports.LiveGrant{Grant: &copied})
		}
	}
	slices.SortFunc(out, func(a, b *supportaccessports.LiveGrant) int {
		return int(a.Grant.ExpiresAt - b.Grant.ExpiresAt)
	})
	if len(out) > req.Limit {
		out = out[:req.Limit]
	}
	return out, nil
}

func (s *grantStore) OrganizationBusinessUnit(_ context.Context, orgID pulid.ID) (pulid.ID, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	bu, ok := s.orgs[orgID]
	if !ok {
		return pulid.Nil, supportaccess.ErrGrantNotFound
	}
	return bu, nil
}

func (s *grantStore) MarkEntered(_ context.Context, req *supportaccessports.MarkEnteredRequest) error {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	if g, ok := s.grants[req.Grant.ID]; ok {
		at := req.At
		g.LastEnteredAt = &at
		g.LastStaffName = req.StaffName
	}
	return nil
}

func (s *grantStore) ListUnrecordedExpiries(_ context.Context, now int64, limit int) ([]*supportaccess.Grant, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*supportaccess.Grant
	for _, g := range s.grants {
		if g.RevokedAt == nil && g.ExpiryRecordedAt == nil && g.ExpiresAt <= now && len(out) < limit {
			copied := *g
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (s *grantStore) MarkExpiryRecorded(_ context.Context, grant *supportaccess.Grant, at int64) (bool, error) {
	s.touch()
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grant.ID]
	if !ok || g.ExpiryRecordedAt != nil || g.RevokedAt != nil {
		return false, nil
	}
	g.ExpiryRecordedAt = &at
	return true, nil
}

type keyStore struct {
	mu   sync.Mutex
	held *supportaccess.KeySet
	gets int
}

func (k *keyStore) Get(context.Context) (*supportaccess.KeySet, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.gets++
	if k.held == nil {
		return nil, supportaccess.ErrNoKeySet
	}
	out := *k.held
	return &out, nil
}

func (k *keyStore) Save(_ context.Context, keys *supportaccess.KeySet) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := *keys
	k.held = &out
	return nil
}

type ledger struct {
	mu   sync.Mutex
	used map[string]bool
}

func (l *ledger) Use(_ context.Context, use *supportaccess.AssertionUse) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used == nil {
		l.used = map[string]bool{}
	}
	if l.used[use.AssertionID] {
		return false, nil
	}
	l.used[use.AssertionID] = true
	return true, nil
}

type sessionStore struct {
	mu           sync.Mutex
	sessions     map[pulid.ID]*session.Session
	endedForUser []pulid.ID
}

var _ sessionports.SessionRepository = (*sessionStore)(nil)

func (s *sessionStore) Get(_ context.Context, id pulid.ID) (*session.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, sessionports.ErrSessionNotFound
	}
	return sess, nil
}

func (s *sessionStore) Create(_ context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions == nil {
		s.sessions = map[pulid.ID]*session.Session{}
	}
	s.sessions[sess.ID] = sess
	return nil
}

func (s *sessionStore) Update(context.Context, pulid.ID, func(*session.Session) error) (*session.Session, error) {
	return nil, sessionports.ErrSessionNotFound
}

func (s *sessionStore) Delete(_ context.Context, id pulid.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *sessionStore) DeleteAllForUser(_ context.Context, userID pulid.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endedForUser = append(s.endedForUser, userID)
	for id, sess := range s.sessions {
		if sess.UserID == userID {
			delete(s.sessions, id)
		}
	}
	return nil
}

func (s *sessionStore) ListForUser(context.Context, pulid.ID) ([]*session.Session, error) {
	return nil, nil
}

func (s *sessionStore) Touch(context.Context, pulid.ID, int64) error { return nil }

func (s *sessionStore) ConfirmAccess(context.Context, pulid.ID, int64) error { return nil }

func (s *sessionStore) live() []*session.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*session.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		out = append(out, sess)
	}
	return out
}

type authEvents struct {
	mu      sync.Mutex
	records []iamports.AuthEventRecord
}

func (a *authEvents) Record(_ context.Context, rec *iamports.AuthEventRecord) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, *rec)
}

type notices struct {
	mu   sync.Mutex
	sent []supportaccessports.Notice
}

func (n *notices) SupportAccessChanged(_ context.Context, notice *supportaccessports.Notice) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, *notice)
	return nil
}

func (n *notices) kinds() []supportaccessports.NoticeKind {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]supportaccessports.NoticeKind, 0, len(n.sent))
	for _, notice := range n.sent {
		out = append(out, notice.Kind)
	}
	return out
}

type clock struct {
	mu  sync.Mutex
	now int64
}

func (c *clock) Now() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now += int64(d / time.Second)
}

type rig struct {
	svc        *Service
	principals *Principals
	grants     *grantStore
	keys       *keyStore
	sessions   *sessionStore
	audit      *securityaudittest.Recorder
	events     *authEvents
	notices    *notices
	clock      *clock
	signer     ed25519.PrivateKey
	public     ed25519.PublicKey
}

func newRig(t *testing.T, managed bool) *rig {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	r := &rig{
		grants:   newGrantStore(),
		keys:     &keyStore{},
		sessions: &sessionStore{},
		audit:    &securityaudittest.Recorder{},
		events:   &authEvents{},
		notices:  &notices{},
		clock:    &clock{now: 1_800_000_000},
		signer:   priv,
		public:   pub,
	}
	managedCfg := &managedconfig.Config{Enabled: managed}
	r.principals = NewPrincipals(PrincipalsParams{
		Grants:   r.grants,
		Registry: permissioncatalog.NewRegistry(),
		Managed:  managedCfg,
	})
	r.svc = New(Params{
		Grants:     r.grants,
		Keys:       r.keys,
		Ledger:     &ledger{},
		Sessions:   r.sessions,
		Tokens:     sessiontest.Tokens(),
		Principals: r.principals,
		Auditor:    r.audit,
		AuthEvents: r.events,
		Notifier:   r.notices,
		Managed:    managedCfg,
		Security:   &securityconfig.Config{Session: securityconfig.SessionConfig{MaxAge: 12 * time.Hour}},
		Clock:      r.clock.Now,
		Logger:     zap.NewNop(),
	})
	return r
}

func (r *rig) deliverKeys(t *testing.T) {
	t.Helper()
	require.NoError(t, r.svc.SetKeys(t.Context(), &supportaccess.KeySet{
		InstanceID:  testInstance,
		StaffDomain: testDomain,
		Keys: []supportaccess.SigningKey{{
			KeyID:     testKeyID,
			KeyType:   "OKP",
			Curve:     "Ed25519",
			Algorithm: "EdDSA",
			Use:       "sig",
			X:         base64.RawURLEncoding.EncodeToString(r.public),
		}},
	}))
}

func (r *rig) organization(orgID, buID pulid.ID) pagination.TenantInfo {
	r.grants.mu.Lock()
	r.grants.orgs[orgID] = buID
	r.grants.mu.Unlock()
	return pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: pulid.MustNew("usr_")}
}

type assertionOptions struct {
	email    string
	org      pulid.ID
	audience string
	id       string
}

func (r *rig) assertion(t *testing.T, opts assertionOptions) string {
	t.Helper()
	now := r.clock.Now()
	if opts.email == "" {
		opts.email = staffEmail
	}
	if opts.audience == "" {
		opts.audience = testInstance
	}
	if opts.id == "" {
		opts.id = "as_" + pulid.MustNew("x").String()[2:]
	}
	payload, err := sonic.Marshal(map[string]any{
		"iss":  supportaccess.AssertionIssuer,
		"sub":  opts.email,
		"aud":  opts.audience,
		"jti":  opts.id,
		"iat":  now,
		"exp":  now + 60,
		"name": "Jane Doe",
		"org":  opts.org.String(),
	})
	require.NoError(t, err)
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{Key: r.signer, KeyID: testKeyID}},
		(&jose.SignerOptions{}).WithType(supportaccess.AssertionType),
	)
	require.NoError(t, err)
	obj, err := signer.Sign(payload)
	require.NoError(t, err)
	token, err := obj.CompactSerialize()
	require.NoError(t, err)
	return token
}

func (r *rig) actions() []string {
	var out []string
	for _, change := range r.audit.Changes() {
		out = append(out, change.Action)
	}
	return out
}

func newOrgID() pulid.ID {
	return pulid.MustNew("org_")
}

func containsUser(ids []pulid.ID, id pulid.ID) bool {
	return slices.Contains(ids, id)
}
