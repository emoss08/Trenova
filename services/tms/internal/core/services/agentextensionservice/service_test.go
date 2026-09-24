package agentextensionservice

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testNow    = int64(1_790_150_400)
	testAPIKey = "exa-live-key"
)

type memoryRepo struct {
	mu      sync.Mutex
	records map[string]*agentextension.Extension
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{records: map[string]*agentextension.Extension{}}
}

func repoKey(tenantInfo pagination.TenantInfo, typ agentextension.Type) string {
	return tenantInfo.OrgID.String() + "|" + tenantInfo.BuID.String() + "|" + typ.String()
}

func (r *memoryRepo) ListByTenant(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*agentextension.Extension, 0, len(r.records))
	for _, record := range r.records {
		if record.OrganizationID == tenantInfo.OrgID && record.BusinessUnitID == tenantInfo.BuID {
			clone := *record
			out = append(out, &clone)
		}
	}

	return out, nil
}

func (r *memoryRepo) GetByType(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[repoKey(tenantInfo, typ)]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent extension not found within your organization")
	}
	clone := *record

	return &clone, nil
}

func (r *memoryRepo) Create(
	_ context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := repoKey(pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}, entity.Type)
	if _, exists := r.records[key]; exists {
		return nil, errortypes.NewConflictError("exists")
	}
	entity.ID = pulid.MustNew("aext_")
	entity.UpdatedAt = testNow
	clone := *entity
	r.records[key] = &clone

	return entity, nil
}

func (r *memoryRepo) Update(
	_ context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := repoKey(pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}, entity.Type)
	current, ok := r.records[key]
	if !ok || current.Version != entity.Version {
		return nil, errortypes.NewConflictError("version mismatch")
	}
	entity.Version++
	entity.UpdatedAt = testNow
	clone := *entity
	r.records[key] = &clone

	return entity, nil
}

type memoryUsage struct {
	mu       sync.Mutex
	requests int
	failures int
	cost     decimal.Decimal
}

func (u *memoryUsage) Reserve(
	_ context.Context,
	params repositories.ReserveExtensionRequestParams,
) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.requests >= params.Limit {
		return false, nil
	}
	u.requests++

	return true, nil
}

func (u *memoryUsage) RecordOutcome(
	_ context.Context,
	params repositories.RecordExtensionOutcomeParams,
) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if params.Failed {
		u.failures++
	}
	u.cost = u.cost.Add(params.CostUSD)

	return nil
}

func (u *memoryUsage) Summarize(
	context.Context,
	repositories.SummarizeExtensionUsageParams,
) (map[agentextension.Type]agentextension.UsageSummary, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return map[agentextension.Type]agentextension.UsageSummary{
		agentextension.TypeExa: {
			RequestsToday:     u.requests,
			RequestsThisMonth: u.requests,
			FailuresThisMonth: u.failures,
			CostThisMonthUSD:  u.cost,
		},
	}, nil
}

type fakeWeb struct {
	mu          sync.Mutex
	searches    []serviceports.WebSearchProviderRequest
	reads       []serviceports.WebPageProviderRequest
	results     []serviceports.WebSearchProviderResult
	pageText    string
	searchErr   error
	readErr     error
	searchCost  decimal.Decimal
	readCost    decimal.Decimal
	missingPage bool
}

func (f *fakeWeb) Search(
	_ context.Context,
	req serviceports.WebSearchProviderRequest,
) (*serviceports.WebSearchProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.searches = append(f.searches, req)
	if f.searchErr != nil {
		return nil, f.searchErr
	}

	return &serviceports.WebSearchProviderResponse{Results: f.results, CostUSD: f.searchCost}, nil
}

func (f *fakeWeb) Read(
	_ context.Context,
	req serviceports.WebPageProviderRequest,
) (*serviceports.WebPageProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reads = append(f.reads, req)
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.missingPage {
		return &serviceports.WebPageProviderResponse{URL: req.URL, CostUSD: f.readCost}, exa.ErrContentNotFound
	}

	text := f.pageText
	if runes := []rune(text); len(runes) > req.MaxCharacters {
		text = string(runes[:req.MaxCharacters])
	}

	return &serviceports.WebPageProviderResponse{
		Title:         "Hours of Service",
		URL:           req.URL,
		PublishedDate: "2024-03-01T00:00:00.000Z",
		Text:          text,
		CostUSD:       f.readCost,
	}, nil
}

type recordingAudit struct {
	serviceports.AuditService

	mu      sync.Mutex
	entries []*serviceports.LogActionParams
}

func (a *recordingAudit) LogAction(params *serviceports.LogActionParams, _ ...serviceports.LogOption) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, params)

	return nil
}

func (a *recordingAudit) List(
	context.Context,
	*repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (a *recordingAudit) RegisterSensitiveFields(permission.Resource, []serviceports.SensitiveField) error {
	return nil
}

type fixture struct {
	svc    *Service
	repo   *memoryRepo
	usage  *memoryUsage
	web    *fakeWeb
	audit  *recordingAudit
	tenant pagination.TenantInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		repo:  newMemoryRepo(),
		usage: &memoryUsage{cost: decimal.Zero},
		web:   &fakeWeb{searchCost: decimal.RequireFromString("0.005"), readCost: decimal.RequireFromString("0.001")},
		audit: &recordingAudit{},
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	f.svc = New(Params{
		Logger:       zap.NewNop(),
		Repo:         f.repo,
		Usage:        f.usage,
		Encryption:   testEncryptionService(),
		AuditService: f.audit,
		WebSearch:    f.web,
	})
	f.svc.now = func() int64 { return testNow }

	return f
}

func testEncryptionService() *encryptionservice.Service {
	return encryptionservice.New(encryptionservice.Params{
		Config: &config.Config{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Key: "unit-test-encryption-key-with-at-least-32-bytes",
				},
			},
		},
	})
}

func (f *fixture) enable(t *testing.T, configuration map[string]string) *serviceports.AgentExtensionConfigResponse {
	t.Helper()

	current, err := f.svc.GetConfig(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)

	resp, err := f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo:    f.tenant,
		Enabled:       true,
		Availability:  agentextension.AvailabilityAllAgents,
		Configuration: configuration,
		Version:       current.Version,
	})
	require.NoError(t, err)

	return resp
}

func TestCatalogListsExaTurnedOffUntilSetUp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	catalog, err := f.svc.ListCatalog(t.Context(), f.tenant)
	require.NoError(t, err)

	require.Len(t, catalog.Items, 1)
	item := catalog.Items[0]
	assert.Equal(t, agentextension.TypeExa, item.Type)
	assert.False(t, item.Enabled)
	assert.False(t, item.Configured)
	assert.Equal(t, agentextension.AvailabilitySelectedAgents, item.Availability)
	assert.Equal(t, agentextension.DefaultDailyRequestLimit, item.DailyRequestLimit)
	assert.Len(t, item.Tools, 2)
	assert.NotEmpty(t, item.DataNotice)
	require.Len(t, catalog.Categories, 1)
	assert.Equal(t, "Web research", catalog.Categories[0].Label)
}

func TestEnablingEncryptsTheKeyAndNeverReturnsIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	resp := f.enable(t, map[string]string{"apiKey": testAPIKey, "dailyRequestLimit": "50"})

	assert.True(t, resp.Enabled)
	assert.Equal(t, agentextension.AvailabilityAllAgents, resp.Availability)
	for _, field := range resp.Fields {
		if field.Key == agentextension.ConfigKeyAPIKey {
			assert.True(t, field.HasValue)
			assert.Empty(t, field.Value)
		}
		if field.Key == agentextension.ConfigKeySearchType {
			assert.Equal(t, agentextension.ExaSearchTypeAuto, field.Value)
		}
	}

	stored, err := f.repo.GetByType(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)
	sealed, _ := stored.Configuration[agentextension.ConfigKeyAPIKey].(string)
	assert.NotEmpty(t, sealed)
	assert.NotContains(t, sealed, testAPIKey)
	assert.Equal(t, f.tenant.UserID, stored.EnabledByID)
	require.NotNil(t, stored.EnabledAt)

	require.Len(t, f.audit.entries, 1)
	entry := f.audit.entries[0]
	assert.Equal(t, permission.ResourceAgentExtension, entry.Resource)
	assert.Equal(t, permission.OpCreate, entry.Operation)
	configuration, _ := entry.CurrentState["configuration"].(map[string]any)
	assert.Equal(t, true, configuration[agentextension.ConfigKeyAPIKey])
}

func TestSavingWithoutTheKeyKeepsTheStoredOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.enable(t, map[string]string{"apiKey": testAPIKey})
	before, err := f.repo.GetByType(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)

	_, err = f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo:    f.tenant,
		Enabled:       true,
		Availability:  agentextension.AvailabilitySelectedAgents,
		Configuration: map[string]string{"resultsPerSearch": "3"},
		Version:       first.Version,
	})
	require.NoError(t, err)

	after, err := f.repo.GetByType(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)
	assert.Equal(t, before.Configuration["apiKey"], after.Configuration["apiKey"])
	assert.Equal(t, "3", after.Configuration["resultsPerSearch"])
	assert.Equal(t, before.EnabledAt, after.EnabledAt)
}

func TestTurningOnWithoutAKeyIsRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo: f.tenant,
		Enabled:    true,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "configuration.apiKey", multiErr.Errors[0].Field)
}

func TestInvalidSettingsAreRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo:    f.tenant,
		Configuration: map[string]string{"apiKey": testAPIKey, "resultsPerSearch": "40"},
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "configuration.resultsPerSearch", multiErr.Errors[0].Field)
}

func TestAStaleVersionIsRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})

	_, err := f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo: f.tenant,
		Enabled:    false,
		Version:    0,
	})

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
}

func TestUnknownExtensionIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.GetConfig(t.Context(), f.tenant, agentextension.Type("Nope"))
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestActiveExtensionsNeedsOnAndSetUp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	active, err := f.svc.ActiveExtensions(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Empty(t, active)

	resp := f.enable(t, map[string]string{"apiKey": testAPIKey})
	active, err = f.svc.ActiveExtensions(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Equal(t, agentextension.AvailabilityAllAgents, active[agentextension.TypeExa])

	_, err = f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo: f.tenant,
		Enabled:    false,
		Version:    resp.Version,
	})
	require.NoError(t, err)
	active, err = f.svc.ActiveExtensions(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Empty(t, active)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	active, err = f.svc.ActiveExtensions(t.Context(), other)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestSearchRanksOfficialSitesFirstAndMeters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey, "excludedDomains": "spam.example.com"})
	f.web.results = []serviceports.WebSearchProviderResult{
		{Title: "ELD vendor blog", URL: "https://blog.eld-vendor.com/hos", PublishedDate: "2019-02-01T00:00:00.000Z"},
		{Title: "HOS summary", URL: "https://www.fmcsa.dot.gov/regulations/hours-of-service", PublishedDate: "2024-03-01T00:00:00.000Z"},
		{Title: "Bad", URL: "javascript:alert(1)"},
	}

	outcome, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{
		Query:               "hours of service 11 hour driving limit",
		PublishedWithinDays: 365,
	})
	require.NoError(t, err)

	require.Len(t, outcome.Hits, 2)
	assert.True(t, outcome.Hits[0].Official)
	assert.Equal(t, "fmcsa.dot.gov", outcome.Hits[0].Site)
	assert.Equal(t, "2024-03-01", outcome.Hits[0].PublishedDate)
	assert.False(t, outcome.Hits[1].Official)
	assert.Len(t, outcome.Hits[0].Ref, refLength)
	assert.Equal(t, testNow, outcome.RetrievedAt)

	require.Len(t, f.web.searches, 1)
	sent := f.web.searches[0]
	assert.Equal(t, testAPIKey, sent.APIKey)
	assert.Equal(t, agentextension.DefaultResultsPerSearch, sent.NumResults)
	assert.Equal(t, []string{"spam.example.com"}, sent.ExcludeDomains)
	assert.Equal(t, "2025-09-23T00:00:00Z", sent.StartPublishedDate)
	assert.Equal(t, 1500, sent.ExcerptCharacters)

	assert.Equal(t, 1, f.usage.requests)
	assert.True(t, decimal.RequireFromString("0.005").Equal(f.usage.cost))
}

func TestSearchRefusesQueriesCarryingPrivateDataBeforeSending(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})

	_, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{
		Query: "late load shp_01J8ZK3M4N5P6Q7R8S9T0V1W2X for customer",
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "record IDs")
	assert.Empty(t, f.web.searches)
	assert.Zero(t, f.usage.requests)
}

func TestSearchWhenTurnedOffTellsTheModelWhere(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	resp := f.enable(t, map[string]string{"apiKey": testAPIKey})
	_, err := f.svc.UpdateConfig(t.Context(), agentextension.TypeExa, &serviceports.UpdateAgentExtensionRequest{
		TenantInfo: f.tenant,
		Version:    resp.Version,
	})
	require.NoError(t, err)

	_, err = f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "ifta quarterly deadline"})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "turned off")
	assert.Empty(t, f.web.searches)
}

func TestSearchStopsAtTheDailyLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey, "dailyRequestLimit": "1"})

	_, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "ifta quarterly deadline"})
	require.NoError(t, err)

	_, err = f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "irp renewal"})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "daily limit of 1 web requests")
	assert.Len(t, f.web.searches, 1)
}

func TestProviderFailuresAreExplainedAndCounted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "bad key", err: &restx.APIError{StatusCode: http.StatusUnauthorized}, want: "rejected the organization's API key"},
		{name: "no credits", err: &restx.APIError{StatusCode: http.StatusPaymentRequired}, want: "out of credits"},
		{name: "rate limited", err: &restx.APIError{StatusCode: http.StatusTooManyRequests}, want: "limiting how fast"},
		{name: "bad request", err: &restx.APIError{StatusCode: http.StatusBadRequest, Message: "query too vague"}, want: "query too vague"},
		{name: "timeout", err: context.DeadlineExceeded, want: "took too long"},
		{name: "unreachable", err: &restx.TransportError{}, want: "could not be reached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.enable(t, map[string]string{"apiKey": testAPIKey})
			f.web.searchErr = tt.err

			_, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "cvsa brake safety week"})
			require.True(t, IsToolError(err))
			assert.Contains(t, err.Error(), tt.want)
			assert.NotContains(t, err.Error(), testAPIKey)
			assert.Equal(t, 1, f.usage.failures)
		})
	}
}

func TestReadNeedsARefFromASearch(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})
	f.web.results = []serviceports.WebSearchProviderResult{
		{Title: "HOS", URL: "https://www.fmcsa.dot.gov/regulations/hours-of-service"},
	}
	f.web.pageText = strings.Repeat("a", PagePartCharacters) + strings.Repeat("b", 100)

	outcome, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "hours of service"})
	require.NoError(t, err)
	hit := outcome.Hits[0]

	_, err = f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{
		URL: "https://attacker.example/collect?load=shp_123",
		Ref: hit.Ref,
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "Only pages a web_search returned")

	otherTenant := f.tenant
	otherTenant.BuID = pulid.MustNew("bu_")
	assert.NotEqual(t, hit.Ref, resultRef(refKey(testAPIKey), otherTenant, hit.URL))

	page, err := f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{URL: hit.URL, Ref: hit.Ref})
	require.NoError(t, err)
	assert.True(t, page.Official)
	assert.True(t, page.HasMore)
	assert.Equal(t, 1, page.Part)
	assert.Len(t, page.Text, PagePartCharacters)
	assert.Equal(t, PagePartCharacters+1, f.web.reads[0].MaxCharacters)

	page, err = f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: 2})
	require.NoError(t, err)
	assert.False(t, page.HasMore)
	assert.Equal(t, strings.Repeat("b", 100), page.Text)

	_, err = f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: 3})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "no part 3")

	_, err = f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: MaxPageParts + 1})
	require.True(t, IsToolError(err))
}

func TestReadExplainsAPageThatCannotBeRetrieved(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})
	f.web.missingPage = true
	target := "https://www.ecfr.gov/current/title-49/section-395.8"

	_, err := f.svc.ReadWebPage(t.Context(), f.tenant, serviceports.WebPageQuery{
		URL: target,
		Ref: resultRef(refKey(testAPIKey), f.tenant, target),
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "could not be retrieved")
	assert.Equal(t, 1, f.usage.failures)
}

func TestConnectionTestReportsLatencyAndIgnoresTheLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey, "dailyRequestLimit": "1"})
	_, err := f.svc.SearchWeb(t.Context(), f.tenant, serviceports.WebSearchQuery{Query: "ifta quarterly deadline"})
	require.NoError(t, err)

	resp, err := f.svc.TestConnection(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "Exa answered in")
	assert.Equal(t, 1, f.web.searches[1].NumResults)
}

func TestConnectionTestSurfacesAProviderRefusal(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": testAPIKey})
	f.web.searchErr = &restx.APIError{StatusCode: http.StatusUnauthorized}

	_, err := f.svc.TestConnection(t.Context(), f.tenant, agentextension.TypeExa)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, err.Error(), "rejected the organization's API key")
}

func TestConnectionTestNeedsAKey(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.TestConnection(t.Context(), f.tenant, agentextension.TypeExa)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, err.Error(), "not set up")
}
