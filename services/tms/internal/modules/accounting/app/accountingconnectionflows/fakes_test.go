package accountingconnectionflows

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type storedTokens struct {
	access, refresh, clientKey    string
	accessExpires, refreshExpires int64
}

type fakeConnections struct {
	mu     sync.Mutex
	rows   map[pulid.ID]*accountingconnection.AccountingConnection
	tokens map[pulid.ID]storedTokens
}

func newFakeConnections() *fakeConnections {
	return &fakeConnections{
		rows:   map[pulid.ID]*accountingconnection.AccountingConnection{},
		tokens: map[pulid.ID]storedTokens{},
	}
}

func sameTenant(
	conn *accountingconnection.AccountingConnection,
	tenant pagination.TenantInfo,
) bool {
	return conn.OrganizationID == tenant.OrgID && conn.BusinessUnitID == tenant.BuID
}

func (f *fakeConnections) withoutTokens(
	conn *accountingconnection.AccountingConnection,
) *accountingconnection.AccountingConnection {
	out := *conn
	out.AccessTokenCiphertext = ""
	out.RefreshTokenCiphertext = ""
	out.ClientKeyCiphertext = ""
	return &out
}

func (f *fakeConnections) withTokens(
	conn *accountingconnection.AccountingConnection,
) *accountingconnection.AccountingConnection {
	out := *conn
	tokens := f.tokens[conn.ID]
	out.AccessTokenCiphertext = tokens.access
	out.RefreshTokenCiphertext = tokens.refresh
	out.ClientKeyCiphertext = tokens.clientKey
	out.AccessTokenExpiresAt = tokens.accessExpires
	out.RefreshTokenExpiresAt = tokens.refreshExpires
	return &out
}

func (f *fakeConnections) byType(
	req accountingconnectionports.GetAccountingConnectionRequest,
) *accountingconnection.AccountingConnection {
	for _, row := range f.rows {
		if sameTenant(row, req.TenantInfo) && row.IntegrationType == req.IntegrationType {
			return row
		}
	}
	return nil
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row := f.byType(req); row != nil {
		return f.withoutTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) GetByID(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionByIDRequest,
) (*accountingconnection.AccountingConnection, error) {
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
) ([]*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingconnection.AccountingConnection{}
	for _, row := range f.rows {
		if sameTenant(row, tenant) {
			out = append(out, f.withoutTokens(row))
		}
	}
	return out, nil
}

func (f *fakeConnections) ListHoldingRealm(
	_ context.Context,
	req accountingconnectionports.ListAccountingConnectionsByRealmRequest,
) ([]*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingconnection.AccountingConnection{}
	for _, row := range f.rows {
		if row.IntegrationType != req.IntegrationType ||
			row.Status == accountingconnection.ConnectionStatusDisconnected {
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
	req accountingconnectionports.ListDueAccountingConnectionsRequest,
) ([]*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingconnection.AccountingConnection{}
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
	req accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row := f.byType(req); row != nil {
		return f.withTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) LockWithTokens(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionByIDRequest,
) (*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if row, ok := f.rows[req.ID]; ok && sameTenant(row, req.TenantInfo) {
		return f.withTokens(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) Create(
	_ context.Context,
	entity *accountingconnection.AccountingConnection,
) (*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[entity.ID] = f.withoutTokens(entity)
	f.tokens[entity.ID] = storedTokens{
		access:         entity.AccessTokenCiphertext,
		refresh:        entity.RefreshTokenCiphertext,
		clientKey:      entity.ClientKeyCiphertext,
		accessExpires:  entity.AccessTokenExpiresAt,
		refreshExpires: entity.RefreshTokenExpiresAt,
	}
	return entity, nil
}

func (f *fakeConnections) Update(
	_ context.Context,
	entity *accountingconnection.AccountingConnection,
) (*accountingconnection.AccountingConnection, error) {
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
	req accountingconnectionports.StoreAccountingTokensRequest,
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
		clientKey:      req.ClientKeyCiphertext,
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
	req accountingconnectionports.MarkAccountingWebhookRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		for _, realm := range req.RealmIDs {
			if row.IntegrationType == req.IntegrationType && row.ExternalRealmID == realm &&
				row.Status != accountingconnection.ConnectionStatusDisconnected {
				at := req.ReceivedAt
				row.LastWebhookAt = &at
				count++
			}
		}
	}
	return count, nil
}

func (f *fakeConnections) SaveDriftCheck(
	context.Context,
	*accountingconnectionports.SaveAccountingDriftCheckRequest,
) error {
	return nil
}

func (f *fakeConnections) SaveChangeFeed(
	_ context.Context,
	req *accountingconnectionports.SaveAccountingChangeFeedRequest,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok || (req.OnlyIfEmpty && row.ChangeCursor != "") {
		return false, nil
	}
	if req.Cursor != "" {
		row.ChangeCursor = req.Cursor
	}
	if req.ReadAt != nil {
		at := *req.ReadAt
		row.ChangesReadAt = &at
	}
	row.ChangesErrorCategory = req.ErrorCategory
	row.ChangesErrorMessage = req.ErrorMessage
	return true, nil
}

func (f *fakeConnections) MarkReferenceRefresh(
	_ context.Context,
	req accountingconnectionports.MarkAccountingReferenceRefreshRequest,
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
	req accountingconnectionports.ListActiveAccountingConnectionsRequest,
) ([]*accountingconnection.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingconnection.AccountingConnection, 0, len(f.rows))
	for _, row := range f.rows {
		if row.IsActive() && row.ID.String() > req.AfterID.String() {
			clone := *row
			out = append(out, &clone)
		}
	}
	slices.SortFunc(out, func(a, b *accountingconnection.AccountingConnection) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	if req.Limit > 0 && len(out) > req.Limit {
		out = out[:req.Limit]
	}
	return out, nil
}

type fakeStates struct {
	mu     sync.Mutex
	states map[string]*accountingconnectionports.AccountingOAuthState
	ttls   map[string]time.Duration
}

func newFakeStates() *fakeStates {
	return &fakeStates{
		states: map[string]*accountingconnectionports.AccountingOAuthState{},
		ttls:   map[string]time.Duration{},
	}
}

func (f *fakeStates) Save(
	_ context.Context,
	state *accountingconnectionports.AccountingOAuthState,
	ttl time.Duration,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[state.State] = state
	f.ttls[state.State] = ttl
	return nil
}

func (f *fakeStates) Take(
	_ context.Context,
	state string,
) (*accountingconnectionports.AccountingOAuthState, error) {
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
	typ            integration.Type
	cc             *fakeClientCredentials
	available      bool
	exchangeGrant  *accountingconnectorports.AccountingTokenGrant
	exchangeErr    error
	refreshGrant   *accountingconnectorports.AccountingTokenGrant
	refreshErrs    []error
	refreshedWith  []string
	factsErrs      []error
	factsCalls     []string
	facts          accountingconnection.CompanyFacts
	revoked        []string
	exchangedCodes []string
	exchanges      []accountingconnectorports.AccountingCodeExchange
	authorizations []accountingconnectorports.AccountingAuthorizeRequest
	refreshRealms  []string
	revokeRealms   []string
	webhookErr     error
	webhookRealms  []string
	errCategories  map[error]accountingsync.ErrorCategory
	redirectURL    string
	verifyAppErr   error
	bound          []*accountingconnectorports.AccountingApp
	companies      []accountingconnectorports.AccountingCompany
	companiesErr   error
	released       []accountingconnectorports.AccountingCompany
}

var testInstanceApp = accountingconnectorports.AccountingApp{
	Source:               accountingconnection.AppSourceInstance,
	Environment:          accountingprovider.AppEnvironmentSandbox,
	ClientID:             "instance-client",
	ClientSecret:         "instance-secret",
	WebhookVerifierToken: instanceVerifier,
}

const instanceVerifier = "instance-verifier"

func newFakeConnector() *fakeConnector {
	return &fakeConnector{
		typ:         integration.TypeQuickBooksOnline,
		available:   true,
		redirectURL: "https://app.example.com/admin/integrations/quickbooks/callback",
		exchangeGrant: &accountingconnectorports.AccountingTokenGrant{
			AccessToken:     "access-1",
			RefreshToken:    "refresh-1",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: 100 * 24 * time.Hour,
		},
		refreshGrant: &accountingconnectorports.AccountingTokenGrant{
			AccessToken:     "access-2",
			RefreshToken:    "refresh-2",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: 100 * 24 * time.Hour,
		},
		facts: accountingconnection.CompanyFacts{
			CompanyName:  "Acme Freight",
			LegalName:    "Acme Freight LLC",
			Country:      "US",
			HomeCurrency: "USD",
		},
		errCategories: map[error]accountingsync.ErrorCategory{},
	}
}

func (f *fakeConnector) IntegrationType() integration.Type { return f.typ }

func (f *fakeConnector) InstanceApp() (*accountingconnectorports.AccountingApp, bool) {
	if !f.available {
		return nil, false
	}
	app := testInstanceApp
	return &app, true
}

func (f *fakeConnector) RedirectURL() string { return f.redirectURL }

func (f *fakeConnector) Bind(
	app *accountingconnectorports.AccountingApp,
) (accountingconnectorports.AccountingConnector, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bound = append(f.bound, app)
	return &boundConnector{fakeConnector: f, app: app}, nil
}

func (f *fakeConnector) lastBound() *accountingconnectorports.AccountingApp {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.bound) == 0 {
		return nil
	}
	return f.bound[len(f.bound)-1]
}

func (f *fakeConnector) VerifyApp(context.Context) error { return f.verifyAppErr }

type boundConnector struct {
	*fakeConnector
	app *accountingconnectorports.AccountingApp
}

func (b *boundConnector) VerifyWebhook(signature string, body []byte) error {
	if err := b.fakeConnector.VerifyWebhook(signature, body); err != nil {
		return err
	}
	if b.app.WebhookVerifierToken == "" || signature != b.app.WebhookVerifierToken {
		return errProvider
	}
	return nil
}

func (f *fakeConnector) AuthorizeURL(
	req accountingconnectorports.AccountingAuthorizeRequest,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.authorizations = append(f.authorizations, req)
	return "https://appcenter.intuit.com/connect/oauth2?state=" + req.State, nil
}

func (f *fakeConnector) ExchangeCode(
	_ context.Context,
	req *accountingconnectorports.AccountingCodeExchange,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchangedCodes = append(f.exchangedCodes, req.Code)
	f.exchanges = append(f.exchanges, *req)
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return f.exchangeGrant, nil
}

func (f *fakeConnector) Refresh(
	_ context.Context,
	req *accountingconnectorports.AccountingRefreshRequest,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshedWith = append(f.refreshedWith, req.RefreshToken)
	f.refreshRealms = append(f.refreshRealms, req.RealmID)
	if len(f.refreshErrs) > 0 {
		err := f.refreshErrs[0]
		f.refreshErrs = f.refreshErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	return f.refreshGrant, nil
}

func (f *fakeConnector) Revoke(
	_ context.Context,
	req *accountingconnectorports.AccountingRevokeRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked = append(f.revoked, req.Token)
	f.revokeRealms = append(f.revokeRealms, req.RealmID)
	return nil
}

func (f *fakeConnector) CompanyFacts(
	_ context.Context,
	_ string,
	accessToken string,
) (*accountingconnection.CompanyFacts, error) {
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

func (f *fakeConnector) Companies(
	_ context.Context,
	_ *accountingconnectorports.AccountingTokenGrant,
	callbackRealmID string,
) ([]accountingconnectorports.AccountingCompany, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.companiesErr != nil {
		return nil, f.companiesErr
	}
	if f.companies != nil {
		return append([]accountingconnectorports.AccountingCompany(nil), f.companies...), nil
	}
	return []accountingconnectorports.AccountingCompany{{ID: callbackRealmID}}, nil
}

func (f *fakeConnector) ReleaseCompanies(
	_ context.Context,
	_ *accountingconnectorports.AccountingTokenGrant,
	companies []accountingconnectorports.AccountingCompany,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released = append(f.released, companies...)
	return nil
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

type fakeRegistry struct {
	connector *fakeConnector
	bc        accountingconnectorports.AccountingProvider
}

func (r fakeRegistry) For(
	typ integration.Type,
) (accountingconnectorports.AccountingProvider, bool) {
	switch {
	case typ == r.connector.typ:
		return r.connector, true
	case typ == integration.TypeBusinessCentral && r.bc != nil:
		return r.bc, true
	default:
		return nil, false
	}
}

type fakeAudit struct {
	auditports.AuditService
	mu      sync.Mutex
	entries []*auditports.LogActionParams
}

func (f *fakeAudit) LogAction(params *auditports.LogActionParams, _ ...auditports.LogOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)
	return nil
}

type projection struct {
	resolved bool
	item     watchtowerports.WatchtowerItemInput
}

type fakeWatchtower struct {
	mu    sync.Mutex
	items map[string]projection
}

func newFakeWatchtower() *fakeWatchtower {
	return &fakeWatchtower{items: map[string]projection{}}
}

func (f *fakeWatchtower) Upsert(_ context.Context, input watchtowerports.WatchtowerItemInput) {
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

type fakeRefresher struct {
	mu        sync.Mutex
	requested []pulid.ID
	err       error
}

func (f *fakeRefresher) RequestReferenceRefresh(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requested = append(f.requested, connectionID)
	return f.err
}

type fakeApps struct {
	mu   sync.Mutex
	rows map[string]*accountingconnection.AccountingAppCredential
}

func newFakeApps() *fakeApps {
	return &fakeApps{rows: map[string]*accountingconnection.AccountingAppCredential{}}
}

func (f *fakeApps) GetByType(
	_ context.Context,
	req accountingconnectionports.GetAccountingAppCredentialRequest,
) (*accountingconnection.AccountingAppCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[integrationKey(req.TenantInfo, req.IntegrationType)]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting app not found")
	}
	clone := *row
	return &clone, nil
}

func (f *fakeApps) GetForWebhook(
	_ context.Context,
	id pulid.ID,
	integrationType integration.Type,
) (*accountingconnection.AccountingAppCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.ID == id && row.IntegrationType == integrationType {
			clone := *row
			return &clone, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting app not found")
}

func (f *fakeApps) Create(
	_ context.Context,
	entity *accountingconnection.AccountingAppCredential,
) (*accountingconnection.AccountingAppCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := integrationKey(
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.IntegrationType,
	)
	clone := *entity
	f.rows[key] = &clone
	return entity, nil
}

func (f *fakeApps) Update(
	_ context.Context,
	entity *accountingconnection.AccountingAppCredential,
) (*accountingconnection.AccountingAppCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := integrationKey(
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		entity.IntegrationType,
	)
	current, ok := f.rows[key]
	if !ok || current.Version != entity.Version {
		return nil, errortypes.NewNotFoundError("Accounting app not found")
	}
	entity.Version++
	clone := *entity
	f.rows[key] = &clone
	return entity, nil
}

func (f *fakeApps) Delete(
	_ context.Context,
	req accountingconnectionports.GetAccountingAppCredentialRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, integrationKey(req.TenantInfo, req.IntegrationType))
	return nil
}

type fakePoller struct {
	mu     sync.Mutex
	polled []pulid.ID
}

func (f *fakePoller) PollNow(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.polled = append(f.polled, connectionID)
	return nil
}

func (f *fakePoller) calls() []pulid.ID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pulid.ID{}, f.polled...)
}
