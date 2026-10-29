package agentextensionservice

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensiontest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	svc      *Service
	repo     *agentextensiontest.MemoryRepo
	usage    *agentextensiontest.MemoryUsage
	web      *agentextensiontest.FakeWeb
	parallel *agentextensiontest.FakeWeb
	audit    *agentextensiontest.RecordingAudit
	tenant   pagination.TenantInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		repo:  agentextensiontest.NewMemoryRepo(),
		usage: &agentextensiontest.MemoryUsage{Cost: decimal.Zero},
		web: &agentextensiontest.FakeWeb{
			Kind:       agentextension.TypeExa,
			SearchCost: decimal.RequireFromString("0.005"),
			ReadCost:   decimal.RequireFromString("0.001"),
		},
		parallel: &agentextensiontest.FakeWeb{
			Kind:       agentextension.TypeParallel,
			SearchCost: decimal.RequireFromString("0.001"),
			ReadCost:   decimal.RequireFromString("0.001"),
		},
		audit: &agentextensiontest.RecordingAudit{},
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
		Encryption:   agentextensiontest.EncryptionService(),
		AuditService: f.audit,
		Connectors: []agentextensionports.ExtensionConnector{
			f.web,
			f.parallel,
		},
	})
	f.svc.now = func() int64 { return agentextensiontest.Now }

	return f
}

func (f *fixture) enable(
	t *testing.T,
	configuration map[string]string,
) *agentextensionports.AgentExtensionConfigResponse {
	t.Helper()

	return f.enableType(t, agentextension.TypeExa, configuration)
}

func (f *fixture) enableType(
	t *testing.T,
	typ agentextension.Type,
	configuration map[string]string,
) *agentextensionports.AgentExtensionConfigResponse {
	t.Helper()

	return agentextensiontest.EnableType(t, f.svc, f.tenant, typ, configuration)
}

func TestCatalogListsExaTurnedOffUntilSetUp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	catalog, err := f.svc.ListCatalog(t.Context(), f.tenant)
	require.NoError(t, err)

	require.Len(t, catalog.Items, len(agentextension.AllTypes()))
	item := catalog.Items[0]
	assert.Equal(t, agentextension.TypeExa, item.Type)
	assert.False(t, item.Enabled)
	assert.False(t, item.Configured)
	assert.Equal(t, agentextension.AvailabilitySelectedAgents, item.Availability)
	assert.Equal(t, agentextension.DefaultDailyRequestLimit, item.DailyRequestLimit)
	assert.Len(t, item.Tools, 5, "Exa searches, reads and watches the web")
	assert.Equal(t, agentextension.CapabilityWebResearch, item.Capability)
	assert.Equal(t, "Web research", item.CapabilityLabel)
	assert.Empty(t, item.CapabilityHeldBy)
	assert.NotEmpty(t, item.DataNotice)
	require.Len(t, catalog.Categories, len(agentextension.AllCategories()))
	assert.Equal(t, "Web research", catalog.Categories[0].Label)
	assert.Equal(t, "Browser automation", catalog.Categories[1].Label)

	byType := make(
		map[agentextension.Type]agentextensionports.AgentExtensionCatalogItem,
		len(catalog.Items),
	)
	for idx := range catalog.Items {
		byType[catalog.Items[idx].Type] = catalog.Items[idx]
	}
	parallel := byType[agentextension.TypeParallel]
	parallelTools := make([]string, 0, len(parallel.Tools))
	for _, tool := range parallel.Tools {
		parallelTools = append(parallelTools, tool.Name)
	}
	assert.Equal(t, []string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		parallelTools, "a vendor that cannot watch lists no watch tools")

	portal := byType[agentextension.TypeBrowserUse]
	assert.Equal(t, agentextension.CapabilityPortalAutomation, portal.Capability)
	assert.Equal(t, agentextension.DefaultPortalDailyTasks, portal.DailyRequestLimit)
	assert.Len(t, portal.Tools, 3)
}

func TestEnablingEncryptsTheKeyAndNeverReturnsIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	resp := f.enable(
		t,
		map[string]string{"apiKey": agentextensiontest.APIKey, "dailyRequestLimit": "50"},
	)

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
	assert.NotContains(t, sealed, agentextensiontest.APIKey)
	assert.Equal(t, f.tenant.UserID, stored.EnabledByID)
	require.NotNil(t, stored.EnabledAt)

	require.Len(t, f.audit.Entries, 1)
	entry := f.audit.Entries[0]
	assert.Equal(t, permission.ResourceAgentExtension, entry.Resource)
	assert.Equal(t, permission.OpCreate, entry.Operation)
	configuration, _ := entry.CurrentState["configuration"].(map[string]any)
	assert.Equal(t, true, configuration[agentextension.ConfigKeyAPIKey])
}

func TestSavingWithoutTheKeyKeepsTheStoredOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	before, err := f.repo.GetByType(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)

	_, err = f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo:    f.tenant,
			Enabled:       true,
			Availability:  agentextension.AvailabilitySelectedAgents,
			Configuration: map[string]string{"resultsPerSearch": "3"},
			Version:       first.Version,
		},
	)
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
	_, err := f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Enabled:    true,
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "configuration.apiKey", multiErr.Errors[0].Field)
}

func TestInvalidSettingsAreRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Configuration: map[string]string{
				"apiKey":           agentextensiontest.APIKey,
				"resultsPerSearch": "40",
			},
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "configuration.resultsPerSearch", multiErr.Errors[0].Field)
}

func TestAStaleVersionIsRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})

	_, err := f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Enabled:    false,
			Version:    0,
		},
	)

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
}

func TestUnknownExtensionIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.GetConfig(t.Context(), f.tenant, agentextension.Type("Nope"))
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestActiveCapabilitiesNeedOnAndSetUp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	active, err := f.svc.ActiveCapabilities(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Empty(t, active)

	resp := f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	active, err = f.svc.ActiveCapabilities(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Equal(t, agentextension.ActiveExtension{
		Type:         agentextension.TypeExa,
		Availability: agentextension.AvailabilityAllAgents,
	}, active[agentextension.CapabilityWebResearch])

	_, err = f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Enabled:    false,
			Version:    resp.Version,
		},
	)
	require.NoError(t, err)
	active, err = f.svc.ActiveCapabilities(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Empty(t, active)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	active, err = f.svc.ActiveCapabilities(t.Context(), other)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestSearchRanksOfficialSitesFirstAndMeters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(
		t,
		map[string]string{
			"apiKey":          agentextensiontest.APIKey,
			"excludedDomains": "spam.example.com",
		},
	)
	f.web.Results = []agentextensionports.WebSearchProviderResult{
		{
			Title:         "ELD vendor blog",
			URL:           "https://blog.eld-vendor.com/hos",
			PublishedDate: "2019-02-01T00:00:00.000Z",
		},
		{
			Title:         "HOS summary",
			URL:           "https://www.fmcsa.dot.gov/regulations/hours-of-service",
			PublishedDate: "2024-03-01T00:00:00.000Z",
		},
		{Title: "Bad", URL: "javascript:alert(1)"},
	}

	outcome, err := f.svc.SearchWeb(t.Context(), f.tenant, agentextensionports.WebSearchQuery{
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
	assert.Equal(t, agentextensiontest.Now, outcome.RetrievedAt)

	require.Len(t, f.web.Searches, 1)
	sent := f.web.Searches[0]
	assert.Equal(t, agentextensiontest.APIKey, sent.Settings[agentextension.ConfigKeyAPIKey])
	assert.Equal(t, agentextension.DefaultResultsPerSearch, sent.NumResults)
	assert.Equal(t, []string{"spam.example.com"}, sent.ExcludeDomains)
	assert.Equal(t, "2025-09-23T00:00:00Z", sent.StartPublishedDate)
	assert.Equal(t, 1500, sent.ExcerptCharacters)

	assert.Equal(t, 1, f.usage.Requests)
	assert.True(t, decimal.RequireFromString("0.005").Equal(f.usage.Cost))
}

func TestSearchRefusesQueriesCarryingPrivateDataBeforeSending(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})

	_, err := f.svc.SearchWeb(t.Context(), f.tenant, agentextensionports.WebSearchQuery{
		Query: "late load shp_01J8ZK3M4N5P6Q7R8S9T0V1W2X for customer",
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "record IDs")
	assert.Empty(t, f.web.Searches)
	assert.Zero(t, f.usage.Requests)
}

func TestSearchWhenTurnedOffTellsTheModelWhere(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	resp := f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	_, err := f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Version:    resp.Version,
		},
	)
	require.NoError(t, err)

	_, err = f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "ifta quarterly deadline"},
	)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "turned off")
	assert.Empty(t, f.web.Searches)
}

func TestSearchStopsAtTheDailyLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey, "dailyRequestLimit": "1"})

	_, err := f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "ifta quarterly deadline"},
	)
	require.NoError(t, err)

	_, err = f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "irp renewal"},
	)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "daily limit of 1 web requests")
	assert.Len(t, f.web.Searches, 1)
}

func TestProviderFailuresAreExplainedAndCounted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "bad key",
			err:  &restx.APIError{StatusCode: http.StatusUnauthorized},
			want: "rejected the organization's API key",
		},
		{
			name: "no credits",
			err:  &restx.APIError{StatusCode: http.StatusPaymentRequired},
			want: "out of credits",
		},
		{
			name: "rate limited",
			err:  &restx.APIError{StatusCode: http.StatusTooManyRequests},
			want: "limiting how fast",
		},
		{
			name: "bad request",
			err:  &restx.APIError{StatusCode: http.StatusBadRequest, Message: "query too vague"},
			want: "query too vague",
		},
		{name: "timeout", err: context.DeadlineExceeded, want: "took too long"},
		{name: "unreachable", err: &restx.TransportError{}, want: "could not be reached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
			f.web.SearchErr = tt.err

			_, err := f.svc.SearchWeb(
				t.Context(),
				f.tenant,
				agentextensionports.WebSearchQuery{Query: "cvsa brake safety week"},
			)
			require.True(t, IsToolError(err))
			assert.Contains(t, err.Error(), tt.want)
			assert.NotContains(t, err.Error(), agentextensiontest.APIKey)
			assert.Equal(t, 1, f.usage.Failures)
		})
	}
}

func TestReadNeedsARefFromASearch(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	f.web.Results = []agentextensionports.WebSearchProviderResult{
		{Title: "HOS", URL: "https://www.fmcsa.dot.gov/regulations/hours-of-service"},
	}
	f.web.PageText = strings.Repeat("a", PagePartCharacters) + strings.Repeat("b", 100)

	outcome, err := f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "hours of service"},
	)
	require.NoError(t, err)
	hit := outcome.Hits[0]

	_, err = f.svc.ReadWebPage(t.Context(), f.tenant, agentextensionports.WebPageQuery{
		URL: "https://attacker.example/collect?load=shp_123",
		Ref: hit.Ref,
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "Only pages a web_search returned")

	otherTenant := f.tenant
	otherTenant.BuID = pulid.MustNew("bu_")
	assert.NotEqual(t, hit.Ref, resultRef(refKey(agentextensiontest.APIKey), otherTenant, hit.URL))

	page, err := f.svc.ReadWebPage(
		t.Context(),
		f.tenant,
		agentextensionports.WebPageQuery{URL: hit.URL, Ref: hit.Ref},
	)
	require.NoError(t, err)
	assert.True(t, page.Official)
	assert.True(t, page.HasMore)
	assert.Equal(t, 1, page.Part)
	assert.Len(t, page.Text, PagePartCharacters)
	assert.Equal(t, PagePartCharacters+1, f.web.Reads[0].MaxCharacters)

	page, err = f.svc.ReadWebPage(
		t.Context(),
		f.tenant,
		agentextensionports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: 2},
	)
	require.NoError(t, err)
	assert.False(t, page.HasMore)
	assert.Equal(t, strings.Repeat("b", 100), page.Text)

	_, err = f.svc.ReadWebPage(
		t.Context(),
		f.tenant,
		agentextensionports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: 3},
	)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "no part 3")

	_, err = f.svc.ReadWebPage(
		t.Context(),
		f.tenant,
		agentextensionports.WebPageQuery{URL: hit.URL, Ref: hit.Ref, Part: MaxPageParts + 1},
	)
	require.True(t, IsToolError(err))
}

func TestReadExplainsAPageThatCannotBeRetrieved(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	f.web.MissingPage = true
	target := "https://www.ecfr.gov/current/title-49/section-395.8"

	_, err := f.svc.ReadWebPage(t.Context(), f.tenant, agentextensionports.WebPageQuery{
		URL: target,
		Ref: resultRef(refKey(agentextensiontest.APIKey), f.tenant, target),
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "could not be retrieved")
	assert.Equal(t, 1, f.usage.Failures)
}

func TestConnectionTestReportsLatencyAndIgnoresTheLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey, "dailyRequestLimit": "1"})
	_, err := f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "ifta quarterly deadline"},
	)
	require.NoError(t, err)

	resp, err := f.svc.TestConnection(t.Context(), f.tenant, agentextension.TypeExa)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "Exa answered in")
	assert.Equal(t, 1, f.web.Searches[1].NumResults)
}

func TestConnectionTestSurfacesAProviderRefusal(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	f.web.SearchErr = &restx.APIError{StatusCode: http.StatusUnauthorized}

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

func TestOnlyOneVendorServesACapability(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})

	current, err := f.svc.GetConfig(t.Context(), f.tenant, agentextension.TypeParallel)
	require.NoError(t, err)
	_, err = f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeParallel,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo:    f.tenant,
			Enabled:       true,
			Availability:  agentextension.AvailabilityAllAgents,
			Configuration: map[string]string{"apiKey": "parallel-key"},
			Version:       current.Version,
		},
	)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "enabled", multiErr.Errors[0].Field)
	assert.Equal(t,
		"{0} is already on and gives agents the same tools. Turn it off before turning on {1}.",
		multiErr.Errors[0].Message,
	)
	assert.Equal(t, []any{"Exa", "Parallel"}, multiErr.Errors[0].Args)

	catalog, err := f.svc.ListCatalog(t.Context(), f.tenant)
	require.NoError(t, err)
	held := map[agentextension.Type]string{}
	for _, item := range catalog.Items {
		held[item.Type] = item.CapabilityHeldBy
	}
	assert.Empty(t, held[agentextension.TypeExa])
	assert.Equal(t, "Exa", held[agentextension.TypeParallel])
}

func TestSwitchingVendorsKeepsTheToolsAndMovesTheCalls(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	exa := f.enable(t, map[string]string{"apiKey": agentextensiontest.APIKey})
	_, err := f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeExa,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo: f.tenant,
			Version:    exa.Version,
		},
	)
	require.NoError(t, err)

	f.enableType(t, agentextension.TypeParallel, map[string]string{
		"apiKey":     "parallel-key",
		"searchType": agentextension.ParallelModeFast,
	})

	active, err := f.svc.ActiveCapabilities(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Equal(t, agentextension.ActiveExtension{
		Type:         agentextension.TypeParallel,
		Availability: agentextension.AvailabilityAllAgents,
	}, active[agentextension.CapabilityWebResearch])
	assert.Equal(t,
		[]string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		active.Grants(),
	)

	f.parallel.Results = []agentextensionports.WebSearchProviderResult{
		{Title: "HOS", URL: "https://www.fmcsa.dot.gov/regulations/hours-of-service"},
	}
	outcome, err := f.svc.SearchWeb(
		t.Context(),
		f.tenant,
		agentextensionports.WebSearchQuery{Query: "hours of service"},
	)
	require.NoError(t, err)
	require.Len(t, outcome.Hits, 1)

	assert.Empty(t, f.web.Searches)
	require.Len(t, f.parallel.Searches, 1)
	assert.Equal(t, "parallel-key", f.parallel.Searches[0].Settings[agentextension.ConfigKeyAPIKey])
	assert.Equal(
		t,
		agentextension.ParallelModeFast,
		f.parallel.Searches[0].Settings[agentextension.ConfigKeySearchType],
	)

	f.parallel.PageText = "page"
	_, err = f.svc.ReadWebPage(t.Context(), f.tenant, agentextensionports.WebPageQuery{
		URL: outcome.Hits[0].URL,
		Ref: outcome.Hits[0].Ref,
	})
	require.NoError(t, err)
	assert.Len(t, f.parallel.Reads, 1)
	assert.Empty(t, f.web.Reads)
}

func TestParallelSettingsRefuseAnExaDepth(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	current, err := f.svc.GetConfig(t.Context(), f.tenant, agentextension.TypeParallel)
	require.NoError(t, err)

	_, err = f.svc.UpdateConfig(
		t.Context(),
		agentextension.TypeParallel,
		&agentextensionports.UpdateAgentExtensionRequest{
			TenantInfo:   f.tenant,
			Enabled:      true,
			Availability: agentextension.AvailabilitySelectedAgents,
			Configuration: map[string]string{
				"apiKey":     "parallel-key",
				"searchType": agentextension.ExaSearchTypeDeep,
			},
			Version: current.Version,
		},
	)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "configuration.searchType", multiErr.Errors[0].Field)
	assert.Equal(t, "Search depth must be fast, basic or advanced", multiErr.Errors[0].Message)
}
