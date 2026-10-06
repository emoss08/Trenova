package accountingconnectionservice

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const bcRealm = "00000000000000000000000000000001_00000000000000000000000000000002_Production"

var errSubscriptionGone = errors.New("subscription not found")

type fakeSubscriber struct {
	mu           sync.Mutex
	seq          int
	created      []services.AccountingSubscribeRequest
	renewed      []services.AccountingRenewSubscriptionRequest
	unsubscribed []string
	subscribeErr error
	renewErr     error
}

func (f *fakeSubscriber) grant(id, url string) *accountingsync.WebhookSubscriptionGrant {
	return &accountingsync.WebhookSubscriptionGrant{
		ExternalID:      id,
		NotificationURL: url,
		ETag:            "W/\"" + id + "\"",
		ExpiresAt:       time.Now().Add(72 * time.Hour),
	}
}

func (f *fakeSubscriber) subscribe(req *services.AccountingSubscribeRequest) (*accountingsync.WebhookSubscriptionGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	f.seq++
	f.created = append(f.created, *req)
	return f.grant("sub-"+strconv.Itoa(f.seq), req.NotificationURL), nil
}

func (f *fakeSubscriber) renew(req *services.AccountingRenewSubscriptionRequest) (*accountingsync.WebhookSubscriptionGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.renewErr != nil {
		return nil, f.renewErr
	}
	f.renewed = append(f.renewed, *req)
	return f.grant(req.ExternalID, req.NotificationURL), nil
}

func (f *fakeSubscriber) clientStateOf(resource string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for idx := len(f.created) - 1; idx >= 0; idx-- {
		if f.created[idx].Resource == resource {
			return f.created[idx].ClientState
		}
	}
	return ""
}

type bcProvider struct {
	*fakeConnector
	sub  *fakeSubscriber
	base string
}

func (p *bcProvider) IntegrationType() integration.Type { return integration.TypeBusinessCentral }

func (p *bcProvider) Bind(app *services.AccountingApp) (services.AccountingConnector, error) {
	bound, err := p.fakeConnector.Bind(app)
	if err != nil {
		return nil, err
	}
	inner, ok := bound.(*boundConnector)
	if !ok {
		return nil, errProvider
	}
	return &bcConnector{boundConnector: inner, sub: p.sub}, nil
}

func (p *bcProvider) NotificationBaseURL() string { return p.base }

type wireNotification struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientState    string `json:"clientState"`
	Resource       string `json:"resource"`
}

func (p *bcProvider) ParseNotifications(body []byte) ([]services.AccountingWebhookNotification, error) {
	var envelope struct {
		Value []wireNotification `json:"value"`
	}
	if err := sonic.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	out := make([]services.AccountingWebhookNotification, 0, len(envelope.Value))
	for _, n := range envelope.Value {
		out = append(out, services.AccountingWebhookNotification{
			SubscriptionID: n.SubscriptionID,
			ClientState:    n.ClientState,
			Resource:       n.Resource,
		})
	}
	return out, nil
}

func (p *bcProvider) ValidationToken(raw string) (string, bool) { return raw, raw != "" }

type bcConnector struct {
	*boundConnector
	sub *fakeSubscriber
}

func (c *bcConnector) IntegrationType() integration.Type { return integration.TypeBusinessCentral }

func (c *bcConnector) SubscriptionResources() []string { return []string{"customers", "salesInvoices"} }

func (c *bcConnector) SubscriptionRenewWindow() time.Duration { return 24 * time.Hour }

func (c *bcConnector) Subscribe(
	_ context.Context,
	req *services.AccountingSubscribeRequest,
) (*accountingsync.WebhookSubscriptionGrant, error) {
	return c.sub.subscribe(req)
}

func (c *bcConnector) RenewSubscription(
	_ context.Context,
	req *services.AccountingRenewSubscriptionRequest,
) (*accountingsync.WebhookSubscriptionGrant, error) {
	return c.sub.renew(req)
}

func (c *bcConnector) Unsubscribe(_ context.Context, req *services.AccountingUnsubscribeRequest) error {
	c.sub.mu.Lock()
	defer c.sub.mu.Unlock()
	c.sub.unsubscribed = append(c.sub.unsubscribed, req.ExternalID)
	return nil
}

func (c *bcConnector) IsSubscriptionGone(err error) bool {
	return errors.Is(err, errSubscriptionGone)
}

func (c *bcConnector) VerifyWebhook(string, []byte) error { return errProvider }

type fakeSubscriptions struct {
	mu   sync.Mutex
	rows map[pulid.ID]*accountingsync.AccountingWebhookSubscription
	conn *fakeConnections
}

func (f *fakeSubscriptions) ListByConnection(
	_ context.Context,
	req repositories.ListAccountingWebhookSubscriptionsRequest,
) ([]*accountingsync.AccountingWebhookSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingWebhookSubscription, 0, len(f.rows))
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID && row.OrganizationID == req.TenantInfo.OrgID {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeSubscriptions) ListByExternalIDs(
	_ context.Context,
	typ integration.Type,
	ids []string,
) ([]*accountingsync.AccountingWebhookSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingWebhookSubscription, 0, len(ids))
	for _, row := range f.rows {
		for _, id := range ids {
			if row.IntegrationType == typ && row.ExternalSubscriptionID == id {
				copied := *row
				out = append(out, &copied)
			}
		}
	}
	return out, nil
}

func (f *fakeSubscriptions) ListConnections(
	_ context.Context,
	req repositories.ListWebhookSubscriptionConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	f.conn.mu.Lock()
	defer f.conn.mu.Unlock()
	out := make([]*accountingsync.AccountingConnection, 0, len(f.conn.rows))
	for _, conn := range f.conn.rows {
		if conn.IntegrationType == integration.TypeBusinessCentral &&
			(req.AfterID.IsNil() || conn.ID > req.AfterID) {
			out = append(out, f.conn.withoutTokens(conn))
		}
	}
	return out, nil
}

func (f *fakeSubscriptions) Create(
	_ context.Context,
	entity *accountingsync.AccountingWebhookSubscription,
) (*accountingsync.AccountingWebhookSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entity.ID = pulid.MustNew("acctwhs_")
	copied := *entity
	f.rows[entity.ID] = &copied
	return entity, nil
}

func (f *fakeSubscriptions) Update(
	_ context.Context,
	entity *accountingsync.AccountingWebhookSubscription,
) (*accountingsync.AccountingWebhookSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rows[entity.ID]; !ok {
		return nil, errortypes.NewNotFoundError("subscription not found")
	}
	entity.Version++
	copied := *entity
	f.rows[entity.ID] = &copied
	return entity, nil
}

func (f *fakeSubscriptions) Delete(
	_ context.Context,
	req repositories.DeleteAccountingWebhookSubscriptionRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, req.ID)
	return nil
}

func (f *fakeSubscriptions) all() []*accountingsync.AccountingWebhookSubscription {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingWebhookSubscription, 0, len(f.rows))
	for _, row := range f.rows {
		copied := *row
		out = append(out, &copied)
	}
	return out
}

type bcHarness struct {
	*harness
	bc   *bcProvider
	subs *fakeSubscriptions
	conn *accountingsync.AccountingConnection
}

func newBCHarness(t *testing.T) *bcHarness {
	t.Helper()
	h := newHarness(t)
	bc := &bcProvider{
		fakeConnector: h.connector,
		sub:           &fakeSubscriber{},
		base:          "https://api.example.com/api/v1",
	}
	subs := &fakeSubscriptions{
		rows: map[pulid.ID]*accountingsync.AccountingWebhookSubscription{},
		conn: h.connections,
	}
	h.svc = New(Params{
		Logger:        zap.NewNop(),
		DB:            dbtest.NopConnection{},
		Connections:   h.connections,
		Apps:          h.apps,
		States:        h.states,
		Integrations:  h.integrations,
		Connectors:    fakeRegistry{connector: h.connector, bc: bc},
		Encryption:    h.encryption,
		AuditService:  h.audit,
		Watchtower:    h.watchtower,
		Publisher:     h.events,
		Refresher:     h.refresher,
		Poller:        h.poller,
		Subscriptions: subs,
	})
	bh := &bcHarness{harness: h, bc: bc, subs: subs}
	bh.conn = bh.seedConnection(t)
	return bh
}

func (h *bcHarness) seedConnection(t *testing.T) *accountingsync.AccountingConnection {
	t.Helper()
	now := timeutils.NowUnix()
	conn := &accountingsync.AccountingConnection{
		ID:              pulid.MustNew("acctc_"),
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		IntegrationType: integration.TypeBusinessCentral,
		ExternalRealmID: bcRealm,
		Status:          accountingsync.ConnectionStatusConnected,
		AppSource:       accountingsync.AppSourceInstance,
		ConnectedAt:     now,
	}
	conn.BindApp(testInstanceApp.Identity())
	conn.EnableSync(accountingsync.SyncSettings{StartDate: now}, now)
	grant, err := h.svc.seal(conn, &services.AccountingTokenGrant{
		AccessToken:     "bc-access",
		RefreshToken:    "bc-refresh",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 90 * 24 * time.Hour,
	}, now)
	require.NoError(t, err)
	h.connections.mu.Lock()
	h.connections.rows[conn.ID] = conn
	h.connections.tokens[conn.ID] = storedTokens{
		access:         grant.AccessTokenCiphertext,
		refresh:        grant.RefreshTokenCiphertext,
		accessExpires:  grant.AccessTokenExpiresAt,
		refreshExpires: grant.RefreshTokenExpiresAt,
	}
	h.connections.mu.Unlock()
	return conn
}

func (h *bcHarness) sweep(t *testing.T) *services.AccountingSubscriptionSweep {
	t.Helper()
	sweep, err := h.svc.SyncWebhookSubscriptions(
		t.Context(),
		&services.SyncAccountingWebhookSubscriptionsRequest{Limit: 10},
	)
	require.NoError(t, err)
	return sweep
}

func (h *bcHarness) notify(t *testing.T, appID string, body any) error {
	t.Helper()
	raw, err := sonic.Marshal(body)
	require.NoError(t, err)
	return h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeBusinessCentral,
		AppID:           appID,
		Body:            raw,
	})
}

func notification(id, state string) map[string]any {
	return map[string]any{"value": []map[string]string{{"subscriptionId": id, "clientState": state}}}
}

func TestTheSweepSubscribesEveryResourceAndSealsItsSecret(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	sweep := h.sweep(t)

	assert.Equal(t, 2, sweep.Created)
	rows := h.subs.all()
	require.Len(t, rows, 2)
	for _, row := range rows {
		assert.Equal(t, accountingsync.WebhookSubscriptionActive, row.Status)
		assert.Equal(t, "https://api.example.com/api/v1/webhooks/accounting/businesscentral/", row.NotificationURL)
		state := h.bc.sub.clientStateOf(row.Resource)
		require.NotEmpty(t, state)
		assert.NotContains(t, row.ClientStateCiphertext, state)
	}

	again := h.sweep(t)
	assert.Zero(t, again.Created)
	assert.Zero(t, again.Renewed)
}

func TestSubscriptionsNearExpiryAreRenewedWithTheirSecret(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	soon := timeutils.NowUnix() + 3600
	for _, row := range h.subs.all() {
		row.ExpiresAt = &soon
		_, err := h.subs.Update(t.Context(), row)
		require.NoError(t, err)
	}

	sweep := h.sweep(t)
	assert.Equal(t, 2, sweep.Renewed)
	for _, renewed := range h.bc.sub.renewed {
		assert.Equal(t, h.bc.sub.clientStateOf(resourceOf(h, renewed.ExternalID)), renewed.ClientState)
	}
}

func resourceOf(h *bcHarness, externalID string) string {
	for _, row := range h.subs.all() {
		if row.ExternalSubscriptionID == externalID {
			return row.Resource
		}
	}
	return ""
}

func TestASubscriptionTheProviderLostIsCreatedAgain(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	soon := timeutils.NowUnix() + 60
	for _, row := range h.subs.all() {
		row.ExpiresAt = &soon
		_, err := h.subs.Update(t.Context(), row)
		require.NoError(t, err)
	}
	h.bc.sub.renewErr = errSubscriptionGone

	sweep := h.sweep(t)
	assert.Equal(t, 2, sweep.Created)
	assert.Len(t, h.bc.sub.created, 4)
}

func TestWithoutAPublicAddressNothingSubscribes(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.bc.base = "http://localhost:3001/api/v1"
	h.sweep(t)

	assert.Empty(t, h.bc.sub.created)
	for _, row := range h.subs.all() {
		assert.Equal(t, accountingsync.WebhookSubscriptionFailed, row.Status)
		assert.Equal(t, errNoNotificationURL.Error(), row.LastError)
	}
	status, err := h.svc.Status(t.Context(), h.tenant, integration.TypeBusinessCentral)
	require.NoError(t, err)
	require.NotNil(t, status.WebhookSubscriptions)
	assert.False(t, status.WebhookSubscriptions.Configured)
	assert.Equal(t, 2, status.WebhookSubscriptions.Failed)
}

func TestARefusedSubscriptionIsMarkedAndBacksOff(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.bc.sub.subscribeErr = errors.New("handshake failed")
	sweep := h.sweep(t)
	assert.Equal(t, 1, sweep.Failed)
	for _, row := range h.subs.all() {
		assert.Equal(t, accountingsync.WebhookSubscriptionFailed, row.Status)
		assert.Equal(t, "handshake failed", row.LastError)
	}

	h.bc.sub.subscribeErr = nil
	again := h.sweep(t)
	assert.Zero(t, again.Created)
}

func TestAStoppedConnectionLosesItsSubscriptions(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	paused := timeutils.NowUnix()
	h.connections.mu.Lock()
	h.connections.rows[h.conn.ID].PausedAt = &paused
	h.connections.mu.Unlock()

	sweep := h.sweep(t)
	assert.Equal(t, 2, sweep.Removed)
	assert.Empty(t, h.subs.all())
	assert.ElementsMatch(t, []string{"sub-1", "sub-2"}, h.bc.sub.unsubscribed)
}

func TestANotificationWithItsSecretWakesItsConnection(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	row := h.subs.all()[0]
	state := h.bc.sub.clientStateOf(row.Resource)

	require.NoError(t, h.notify(t, "", notification(row.ExternalSubscriptionID, state)))
	assert.NotNil(t, h.connections.rows[h.conn.ID].LastWebhookAt)
	assert.Contains(t, h.poller.calls(), h.conn.ID)
}

func TestANotificationWithTheWrongSecretIsRefused(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	row := h.subs.all()[0]

	for _, body := range []any{
		notification(row.ExternalSubscriptionID, "forged"),
		notification("unknown-subscription", h.bc.sub.clientStateOf(row.Resource)),
		map[string]any{"value": []any{}},
	} {
		err := h.notify(t, "", body)
		require.Error(t, err)
		assert.True(t, errortypes.IsAuthenticationError(err))
	}
	assert.Nil(t, h.connections.rows[h.conn.ID].LastWebhookAt)
	assert.Empty(t, h.poller.calls())
}

func TestANotificationOnAnotherAppsPathIsRefused(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)
	row := h.subs.all()[0]
	other := &accountingsync.AccountingAppCredential{
		ID:              pulid.MustNew("acctapp_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		IntegrationType: integration.TypeBusinessCentral,
		Environment:     accountingsync.AppEnvironmentProduction,
		ClientID:        "other-client",
	}
	sealed, err := h.encryption.EncryptStringWithAAD("other-secret", h.svc.appAAD(other, clientSecretField))
	require.NoError(t, err)
	other.ClientSecretCiphertext = sealed
	h.apps.rows[integrationKey(pagination.TenantInfo{OrgID: other.OrganizationID, BuID: other.BusinessUnitID}, other.IntegrationType)] = other

	err = h.notify(t, other.ID.String(), notification(row.ExternalSubscriptionID, h.bc.sub.clientStateOf(row.Resource)))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Nil(t, h.connections.rows[h.conn.ID].LastWebhookAt)
}

func TestDisconnectingReleasesTheSubscriptions(t *testing.T) {
	t.Parallel()

	h := newBCHarness(t)
	h.sweep(t)

	_, err := h.svc.Disconnect(t.Context(), &services.DisconnectAccountingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeBusinessCentral,
	})
	require.NoError(t, err)
	assert.Empty(t, h.subs.all())
	assert.Len(t, h.bc.sub.unsubscribed, 2)
}

func TestTheProfileWithoutPaymentsForcesThePolicyOff(t *testing.T) {
	t.Parallel()

	conn := &accountingsync.AccountingConnection{IntegrationType: integration.TypeBusinessCentral}
	conn.SetInboundPayments(accountingsync.InboundPaymentsApply)
	assert.Equal(t, accountingsync.InboundPaymentsOff, conn.PaymentPolicy())
	assert.False(t, conn.ReadsPayments())
}
