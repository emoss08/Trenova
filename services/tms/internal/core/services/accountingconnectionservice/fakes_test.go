package accountingconnectionservice

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type storedTokens struct {
	access, refresh               string
	accessExpires, refreshExpires int64
}

type fakeConnections struct {
	mu     sync.Mutex
	rows   map[pulid.ID]*accountingsync.AccountingConnection
	tokens map[pulid.ID]storedTokens
}

func newFakeConnections() *fakeConnections {
	return &fakeConnections{
		rows:   map[pulid.ID]*accountingsync.AccountingConnection{},
		tokens: map[pulid.ID]storedTokens{},
	}
}

func sameTenant(conn *accountingsync.AccountingConnection, tenant pagination.TenantInfo) bool {
	return conn.OrganizationID == tenant.OrgID && conn.BusinessUnitID == tenant.BuID
}

func (f *fakeConnections) withoutTokens(conn *accountingsync.AccountingConnection) *accountingsync.AccountingConnection {
	out := *conn
	out.AccessTokenCiphertext = ""
	out.RefreshTokenCiphertext = ""
	return &out
}

func (f *fakeConnections) withTokens(conn *accountingsync.AccountingConnection) *accountingsync.AccountingConnection {
	out := *conn
	tokens := f.tokens[conn.ID]
	out.AccessTokenCiphertext = tokens.access
	out.RefreshTokenCiphertext = tokens.refresh
	out.AccessTokenExpiresAt = tokens.accessExpires
	out.RefreshTokenExpiresAt = tokens.refreshExpires
	return &out
}

func (f *fakeConnections) byType(req repositories.GetAccountingConnectionRequest) *accountingsync.AccountingConnection {
	for _, row := range f.rows {
		if sameTenant(row, req.TenantInfo) && row.IntegrationType == req.IntegrationType {
			return row
		}
	}
	return nil
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row := f.byType(req); row != nil {
		return f.withoutTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) GetByID(
	_ context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row, ok := f.rows[req.ID]; ok && sameTenant(row, req.TenantInfo) {
		return f.withoutTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) ListByTenant(
	_ context.Context,
	tenant pagination.TenantInfo,
) ([]*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingConnection{}
	for _, row := range f.rows {
		if sameTenant(row, tenant) {
			out = append(out, f.withoutTokens(row))
		}
	}
	return out, nil
}

func (f *fakeConnections) ListHoldingRealm(
	_ context.Context,
	req repositories.ListAccountingConnectionsByRealmRequest,
) ([]*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingConnection{}
	for _, row := range f.rows {
		if row.IntegrationType != req.IntegrationType ||
			row.Status == accountingsync.ConnectionStatusDisconnected {
			continue
		}
		for _, realm := range req.RealmIDs {
			if row.ExternalRealmID == realm {
				out = append(out, f.withoutTokens(row))
			}
		}
	}
	return out, nil
}

func (f *fakeConnections) ListDueForHealthCheck(
	_ context.Context,
	req repositories.ListDueAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingConnection{}
	for _, row := range f.rows {
		if !row.IsActive() {
			continue
		}
		if row.LastCheckedAt == nil || *row.LastCheckedAt < req.CheckedBefore {
			out = append(out, f.withoutTokens(row))
		}
	}
	return out, nil
}

func (f *fakeConnections) LockByTypeWithTokens(
	_ context.Context,
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row := f.byType(req); row != nil {
		return f.withTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) LockWithTokens(
	_ context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row, ok := f.rows[req.ID]; ok && sameTenant(row, req.TenantInfo) {
		return f.withTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) Create(
	_ context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[entity.ID] = f.withoutTokens(entity)
	f.tokens[entity.ID] = storedTokens{
		access:         entity.AccessTokenCiphertext,
		refresh:        entity.RefreshTokenCiphertext,
		accessExpires:  entity.AccessTokenExpiresAt,
		refreshExpires: entity.RefreshTokenExpiresAt,
	}
	return entity, nil
}

func (f *fakeConnections) Update(
	_ context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rows[entity.ID]; !ok {
		return nil, errortypes.NewNotFoundError("Accounting connection not found")
	}
	entity.Version++
	stored := f.withoutTokens(entity)
	tokens := f.tokens[entity.ID]
	stored.AccessTokenExpiresAt = tokens.accessExpires
	stored.RefreshTokenExpiresAt = tokens.refreshExpires
	f.rows[entity.ID] = stored
	return entity, nil
}

func (f *fakeConnections) StoreTokens(
	_ context.Context,
	req repositories.StoreAccountingTokensRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok {
		return errortypes.NewNotFoundError("Accounting connection not found")
	}
	f.tokens[req.ID] = storedTokens{
		access:         req.AccessTokenCiphertext,
		refresh:        req.RefreshTokenCiphertext,
		accessExpires:  req.AccessTokenExpiresAt,
		refreshExpires: req.RefreshTokenExpiresAt,
	}
	row.AccessTokenExpiresAt = req.AccessTokenExpiresAt
	row.RefreshTokenExpiresAt = req.RefreshTokenExpiresAt
	row.LastRefreshedAt = req.RefreshedAt
	return nil
}

func (f *fakeConnections) MarkWebhookReceived(
	_ context.Context,
	req repositories.MarkAccountingWebhookRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		for _, realm := range req.RealmIDs {
			if row.IntegrationType == req.IntegrationType && row.ExternalRealmID == realm &&
				row.Status != accountingsync.ConnectionStatusDisconnected {
				at := req.ReceivedAt
				row.LastWebhookAt = &at
				count++
			}
		}
	}
	return count, nil
}

func (f *fakeConnections) MarkReferenceRefresh(
	_ context.Context,
	req repositories.MarkAccountingReferenceRefreshRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok {
		return errortypes.NewNotFoundError("Accounting connection not found")
	}
	if req.StartedAt != nil {
		at := *req.StartedAt
		row.ReferenceRefreshStartedAt = &at
	}
	if req.RefreshedAt != nil {
		at := *req.RefreshedAt
		row.ReferenceRefreshedAt = &at
	}
	if req.RefreshedAt != nil || req.Error != "" {
		row.ReferenceRefreshError = req.Error
	}
	return nil
}

func (f *fakeConnections) ListActive(
	_ context.Context,
	req repositories.ListActiveAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingConnection, 0, len(f.rows))
	for _, row := range f.rows {
		if row.IsActive() && row.ID.String() > req.AfterID.String() {
			clone := *row
			out = append(out, &clone)
		}
	}
	slices.SortFunc(out, func(a, b *accountingsync.AccountingConnection) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	if req.Limit > 0 && len(out) > req.Limit {
		out = out[:req.Limit]
	}
	return out, nil
}

type fakeStates struct {
	mu     sync.Mutex
	states map[string]*repositories.AccountingOAuthState
	ttls   map[string]time.Duration
}

func newFakeStates() *fakeStates {
	return &fakeStates{
		states: map[string]*repositories.AccountingOAuthState{},
		ttls:   map[string]time.Duration{},
	}
}

func (f *fakeStates) Save(
	_ context.Context,
	state *repositories.AccountingOAuthState,
	ttl time.Duration,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[state.State] = state
	f.ttls[state.State] = ttl
	return nil
}

func (f *fakeStates) Take(_ context.Context, state string) (*repositories.AccountingOAuthState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	found, ok := f.states[state]
	if !ok {
		return nil, errortypes.NewNotFoundError("expired")
	}
	delete(f.states, state)
	return found, nil
}

type fakeIntegrations struct {
	mu      sync.Mutex
	records map[string]*integration.Integration
}

func newFakeIntegrations() *fakeIntegrations {
	return &fakeIntegrations{records: map[string]*integration.Integration{}}
}

func integrationKey(tenant pagination.TenantInfo, typ integration.Type) string {
	return tenant.OrgID.String() + tenant.BuID.String() + string(typ)
}

func (f *fakeIntegrations) ListByTenant(
	context.Context,
	pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (f *fakeIntegrations) ListEnabledByType(
	context.Context,
	integration.Type,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (f *fakeIntegrations) GetByType(
	_ context.Context,
	tenant pagination.TenantInfo,
	typ integration.Type,
) (*integration.Integration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if record, ok := f.records[integrationKey(tenant, typ)]; ok {
		copied := *record
		return &copied, nil
	}
	return nil, errortypes.NewNotFoundError("Integration not found")
}

func (f *fakeIntegrations) Upsert(
	_ context.Context,
	entity *integration.Integration,
) (*integration.Integration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	copied := *entity
	f.records[integrationKey(tenant, entity.Type)] = &copied
	return entity, nil
}

func (f *fakeIntegrations) enabled(tenant pagination.TenantInfo) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	record, ok := f.records[integrationKey(tenant, integration.TypeQuickBooksOnline)]
	return ok && record.Enabled
}

var errProvider = errors.New("provider failure")

type fakeConnector struct {
	mu             sync.Mutex
	available      bool
	exchangeGrant  *services.AccountingTokenGrant
	exchangeErr    error
	refreshGrant   *services.AccountingTokenGrant
	refreshErrs    []error
	refreshedWith  []string
	factsErrs      []error
	factsCalls     []string
	facts          accountingsync.CompanyFacts
	revoked        []string
	exchangedCodes []string
	webhookErr     error
	webhookRealms  []string
	errCategories  map[error]accountingsync.ErrorCategory
}

func newFakeConnector() *fakeConnector {
	return &fakeConnector{
		available: true,
		exchangeGrant: &services.AccountingTokenGrant{
			AccessToken:     "access-1",
			RefreshToken:    "refresh-1",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: 100 * 24 * time.Hour,
		},
		refreshGrant: &services.AccountingTokenGrant{
			AccessToken:     "access-2",
			RefreshToken:    "refresh-2",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: 100 * 24 * time.Hour,
		},
		facts: accountingsync.CompanyFacts{
			CompanyName:  "Acme Freight",
			LegalName:    "Acme Freight LLC",
			Country:      "US",
			HomeCurrency: "USD",
		},
		errCategories: map[error]accountingsync.ErrorCategory{},
	}
}

func (f *fakeConnector) IntegrationType() integration.Type { return integration.TypeQuickBooksOnline }

func (f *fakeConnector) Available() bool { return f.available }

func (f *fakeConnector) AuthorizeURL(state string) (string, error) {
	return "https://appcenter.intuit.com/connect/oauth2?state=" + state, nil
}

func (f *fakeConnector) ExchangeCode(
	_ context.Context,
	code string,
) (*services.AccountingTokenGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchangedCodes = append(f.exchangedCodes, code)
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return f.exchangeGrant, nil
}

func (f *fakeConnector) Refresh(
	_ context.Context,
	refreshToken string,
) (*services.AccountingTokenGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshedWith = append(f.refreshedWith, refreshToken)
	if len(f.refreshErrs) > 0 {
		err := f.refreshErrs[0]
		f.refreshErrs = f.refreshErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	return f.refreshGrant, nil
}

func (f *fakeConnector) Revoke(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked = append(f.revoked, token)
	return nil
}

func (f *fakeConnector) CompanyFacts(
	_ context.Context,
	_ string,
	accessToken string,
) (*accountingsync.CompanyFacts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.factsCalls = append(f.factsCalls, accessToken)
	if len(f.factsErrs) > 0 {
		err := f.factsErrs[0]
		f.factsErrs = f.factsErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	facts := f.facts
	return &facts, nil
}

func (f *fakeConnector) VerifyWebhook(string, []byte) error { return f.webhookErr }

func (f *fakeConnector) WebhookRealmIDs([]byte) ([]string, error) { return f.webhookRealms, nil }

func (f *fakeConnector) WebhookSignatureHeader() string { return "intuit-signature" }

func (f *fakeConnector) ClassifyError(err error) accountingsync.ErrorCategory {
	if err == nil {
		return ""
	}
	if category, ok := f.errCategories[err]; ok {
		return category
	}
	return accountingsync.ErrorCategoryUnknown
}

type fakeRegistry struct{ connector *fakeConnector }

func (r fakeRegistry) For(typ integration.Type) (services.AccountingConnector, bool) {
	if typ == integration.TypeQuickBooksOnline {
		return r.connector, true
	}
	return nil, false
}

type fakeAudit struct {
	services.AuditService
	mu      sync.Mutex
	entries []*services.LogActionParams
}

func (f *fakeAudit) LogAction(params *services.LogActionParams, _ ...services.LogOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)
	return nil
}

type projection struct {
	resolved bool
	item     services.WatchtowerItemInput
}

type fakeWatchtower struct {
	mu    sync.Mutex
	items map[string]projection
}

func newFakeWatchtower() *fakeWatchtower {
	return &fakeWatchtower{items: map[string]projection{}}
}

func (f *fakeWatchtower) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items[input.SourceID] = projection{item: input}
}

func (f *fakeWatchtower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if kind != watchtower.SourceAccountingSync {
		panic("resolved with the wrong kind: " + string(kind))
	}
	f.items[sourceID] = projection{resolved: true}
}

func (f *fakeWatchtower) get(sourceID string) (projection, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[sourceID]
	return item, ok
}
