package agentquerytoolservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResearcher struct {
	searched *serviceports.WebSearchQuery
	read     *serviceports.WebPageQuery
	tenant   pagination.TenantInfo
	outcome  *serviceports.WebSearchOutcome
	page     *serviceports.WebPage
	err      error
}

func (s *stubResearcher) SearchWeb(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	query serviceports.WebSearchQuery,
) (*serviceports.WebSearchOutcome, error) {
	s.tenant = tenantInfo
	s.searched = &query

	return s.outcome, s.err
}

func (s *stubResearcher) ReadWebPage(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	query serviceports.WebPageQuery,
) (*serviceports.WebPage, error) {
	s.tenant = tenantInfo
	s.read = &query

	return s.page, s.err
}

func TestWebSearchReturnsSourcesTheModelCanCite(t *testing.T) {
	t.Parallel()

	research := &stubResearcher{outcome: &serviceports.WebSearchOutcome{
		RetrievedAt: 1_790_150_400,
		Hits: []serviceports.WebSearchHit{
			{
				Ref:           "abcdEFGH12345678",
				Title:         "Summary of Hours of Service Regulations",
				URL:           "https://www.fmcsa.dot.gov/regulations/hours-of-service",
				Site:          "fmcsa.dot.gov",
				Official:      true,
				PublishedDate: "2024-03-01",
				Excerpts:      []string{"11-Hour Driving Limit"},
			},
			{Ref: "zyxw9876ZYXW5432", Title: "Blog", URL: "https://blog.example.com/hos", Site: "blog.example.com"},
		},
	}}
	params := testParams(map[string]any{
		"query":               " hours of service ",
		"publishedWithinDays": float64(30),
		"limit":               "3",
	})

	result, err := newWebSearchTool(research).Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, research.searched)
	assert.Equal(t, "hours of service", research.searched.Query)
	assert.Equal(t, 30, research.searched.PublishedWithinDays)
	assert.Equal(t, 3, research.searched.ResultLimit)
	assert.Equal(t, params.Actor.TenantInfo(), research.tenant)

	answer, ok := result.(webSearchAnswer)
	require.True(t, ok)
	assert.Equal(t, "2026-09-23", answer.RetrievedOn)
	require.Len(t, answer.Results, 2)
	assert.Equal(t, webSourceOfficial, answer.Results[0].Source)
	assert.Equal(t, webSourceWeb, answer.Results[1].Source)
	assert.Equal(t, "abcdEFGH12345678", answer.Results[0].Ref)
	assert.Contains(t, answer.Note, "[title](url)")
	assert.Contains(t, answer.Note, "untrusted")
}

func TestWebSearchSaysSoWhenNothingMatched(t *testing.T) {
	t.Parallel()

	research := &stubResearcher{outcome: &serviceports.WebSearchOutcome{}}
	result, err := newWebSearchTool(research).Query(t.Context(), testParams(map[string]any{"query": "xyzzy"}))
	require.NoError(t, err)

	answer, ok := result.(webSearchAnswer)
	require.True(t, ok)
	assert.Empty(t, answer.Results)
	assert.Equal(t, webNoResultsGuidance, answer.Note)
}

func TestWebSearchNeedsAQueryAndPassesRefusalsOn(t *testing.T) {
	t.Parallel()

	research := &stubResearcher{}
	_, err := newWebSearchTool(research).Query(t.Context(), testParams(map[string]any{}))
	require.Error(t, err)
	assert.Nil(t, research.searched)

	refusal := errors.New("this organization has used its daily limit")
	research.err = refusal
	_, err = newWebSearchTool(research).Query(t.Context(), testParams(map[string]any{"query": "irp"}))
	require.ErrorIs(t, err, refusal)
}

func TestWebSearchRefusesAMismatchedTenant(t *testing.T) {
	t.Parallel()

	params := testParams(map[string]any{"query": "ifta"})
	params.OrganizationID = testParams(nil).OrganizationID

	research := &stubResearcher{}
	_, err := newWebSearchTool(research).Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, research.searched)
}

func TestWebReadPassesTheRefAndSaysWhetherThePageContinues(t *testing.T) {
	t.Parallel()

	research := &stubResearcher{page: &serviceports.WebPage{
		Title:       "49 CFR 395.8",
		URL:         "https://www.ecfr.gov/current/title-49/section-395.8",
		Site:        "ecfr.gov",
		Official:    true,
		Text:        "Each motor carrier shall require every driver...",
		Part:        2,
		HasMore:     true,
		RetrievedAt: 1_790_150_400,
	}}

	result, err := newWebReadTool(research).Query(t.Context(), testParams(map[string]any{
		"url":  "https://www.ecfr.gov/current/title-49/section-395.8",
		"ref":  "abcdEFGH12345678",
		"part": float64(2),
	}))
	require.NoError(t, err)

	require.NotNil(t, research.read)
	assert.Equal(t, "abcdEFGH12345678", research.read.Ref)
	assert.Equal(t, 2, research.read.Part)

	answer, ok := result.(webPageAnswer)
	require.True(t, ok)
	assert.Equal(t, webSourceOfficial, answer.Source)
	assert.Equal(t, 2, answer.Part)
	assert.Contains(t, answer.Note, "part set to the next number")
}

func TestWebReadNeedsAURLAndARef(t *testing.T) {
	t.Parallel()

	research := &stubResearcher{}
	_, err := newWebReadTool(research).Query(t.Context(), testParams(map[string]any{
		"url": "https://www.ecfr.gov/current/title-49",
	}))
	require.Error(t, err)
	assert.Nil(t, research.read)
}

func TestWebToolsAreGatedOnWebResearch(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentQueryTool{newWebSearchTool(nil), newWebReadTool(nil)} {
		assert.Equal(t, permission.ResourceWebResearch, tool.PermissionResource(), tool.Name())
		assert.True(t, permission.IsAgentAllowed(tool.PermissionResource(), permission.OpRead), tool.Name())
	}
}

func TestEveryExtensionToolIsRegistered(t *testing.T) {
	t.Parallel()

	registered := make(map[string]struct{}, len(ToolProviders()))
	for _, provider := range ToolProviders() {
		fn := reflect.ValueOf(provider)
		args := make([]reflect.Value, fn.Type().NumIn())
		for idx := range args {
			args[idx] = reflect.Zero(fn.Type().In(idx))
		}
		out := fn.Call(args)
		if tool, ok := out[0].Interface().(serviceports.AgentQueryTool); ok {
			registered[tool.Name()] = struct{}{}
		}
	}

	for _, typ := range agentextension.AllTypes() {
		spec, _ := agentextension.SpecFor(typ)
		for _, name := range spec.Tools {
			assert.Contains(t, registered, name, "%s declares %s but no tool is registered", typ, name)
		}
	}
}
