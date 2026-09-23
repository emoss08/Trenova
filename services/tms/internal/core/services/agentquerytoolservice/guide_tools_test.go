package agentquerytoolservice

import (
	"context"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGuide struct {
	matches     []serviceports.ProductGuideMatch
	destination *serviceports.ProductGuideDestination
	err         error
	searched    *serviceports.ProductGuideSearchRequest
	resolved    *serviceports.ProductGuideDestinationRequest
}

func (s *stubGuide) Search(
	_ context.Context,
	req *serviceports.ProductGuideSearchRequest,
) ([]serviceports.ProductGuideMatch, error) {
	s.searched = req

	return s.matches, s.err
}

func (s *stubGuide) Destination(
	_ context.Context,
	req *serviceports.ProductGuideDestinationRequest,
) (*serviceports.ProductGuideDestination, error) {
	s.resolved = req

	return s.destination, s.err
}

func (s *stubGuide) PageForPath(string) (*productguide.Page, bool) { return nil, false }

func rateMatrices() *productguide.Page {
	return &productguide.Page{
		Path:       "/billing/configuration-files/rate-matrices",
		Name:       "Rate matrices",
		Breadcrumb: []string{"Billing", "Configuration files", "Rate matrices"},
		Summary:    "Rates by lane.",
		Related:    []string{"/billing/configuration-files/rate-zones"},
	}
}

func TestFindInTrenova_AnswersWithTheBestPagesStepsAndTheOthersNamed(t *testing.T) {
	t.Parallel()

	best := rateMatrices()
	other := &productguide.Page{
		Path:       "/billing/configuration-files/rate-zones",
		Name:       "Rate zones",
		Breadcrumb: []string{"Billing", "Configuration files", "Rate zones"},
		Tasks:      []productguide.Task{{Title: "Add a rate zone", Steps: []string{"Open it."}}},
	}
	guide := &stubGuide{matches: []serviceports.ProductGuideMatch{
		{
			Page:    best,
			Task:    &productguide.Task{Title: "Add a rate matrix", Steps: []string{"Open it.", "Save."}},
			CanOpen: true,
		},
		{Page: other, Task: &other.Tasks[0], CanOpen: false, Missing: []string{"Rate zone access"}},
	}}
	tool := newFindInTrenovaTool(guide)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"question": "how do I add a rate matrix",
		"page":     "/billing/invoices",
	}))
	require.NoError(t, err)

	found, ok := result.(guideSearchResult)
	require.True(t, ok)
	require.Len(t, found.Answers, 2)

	first := found.Answers[0]
	assert.Equal(t, "/billing/configuration-files/rate-matrices", first.Path)
	assert.Equal(t, "Billing › Configuration files › Rate matrices", first.Location)
	assert.Equal(t, "Rates by lane.", first.Summary)
	require.NotNil(t, first.Task)
	assert.Equal(t, []string{"Open it.", "Save."}, first.Task.Steps)
	assert.Equal(t, []string{"/billing/configuration-files/rate-zones"}, first.Related)

	second := found.Answers[1]
	assert.False(t, second.CanOpen)
	assert.Equal(t, []string{"Rate zone access"}, second.Missing)
	require.NotNil(t, second.Task)
	assert.Empty(t, second.Task.Steps, "only the best answer carries its steps")

	require.NotNil(t, guide.searched)
	assert.Equal(t, "how do I add a rate matrix", guide.searched.Query)
	assert.Equal(t, "/billing/invoices", guide.searched.Page)
	assert.Equal(t, guideAnswerLimit, guide.searched.Limit)
	require.NotNil(t, guide.searched.Actor, "the answer depends on who is asking")
	assert.Equal(t, guide.searched.Actor.UserID, guide.searched.TenantInfo.UserID)
}

// Nothing found is said, with an instruction not to guess, rather than being
// an empty list the model fills in itself.
func TestFindInTrenova_SaysSoWhenNothingMatches(t *testing.T) {
	t.Parallel()

	tool := newFindInTrenovaTool(&stubGuide{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{"question": "the moon"}))
	require.NoError(t, err)

	found, ok := result.(guideSearchResult)
	require.True(t, ok)
	assert.NotNil(t, found.Answers)
	assert.Empty(t, found.Answers)
	assert.Contains(t, found.Note, "do not guess")
}

func TestFindInTrenova_NeedsAQuestion(t *testing.T) {
	t.Parallel()

	guide := &stubGuide{}
	_, err := newFindInTrenovaTool(guide).Query(t.Context(), testParams(map[string]any{}))

	require.Error(t, err)
	assert.Nil(t, guide.searched)
}

func TestFindInTrenova_RefusesACallForNobody(t *testing.T) {
	t.Parallel()

	params := testParams(map[string]any{"question": "where are invoices"})
	params.Actor = nil

	_, err := newFindInTrenovaTool(&stubGuide{}).Query(t.Context(), params)

	require.ErrorIs(t, err, ErrMissingActor)
}

func TestOpenPage_ReturnsWhereTheAppIsGoing(t *testing.T) {
	t.Parallel()

	page := rateMatrices()
	guide := &stubGuide{destination: &serviceports.ProductGuideDestination{
		Path:  page.Path + "?panelType=create",
		Page:  page,
		Label: page.Name,
	}}
	tool := newOpenPageTool(guide)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"page":   page.Path,
		"action": "create",
	}))
	require.NoError(t, err)

	moved, ok := result.(NavigationResult)
	require.True(t, ok)
	assert.Equal(t, page.Path+"?panelType=create", moved.Path)
	assert.Equal(t, "Rate matrices", moved.Name)
	assert.Equal(t, "Billing › Configuration files › Rate matrices", moved.Location)
	assert.Equal(t, page.Path, moved.Page)

	require.NotNil(t, guide.resolved)
	assert.True(t, guide.resolved.Create)
	assert.Equal(t, page.Path, guide.resolved.Page)
}

func TestOpenPage_PassesARecordThrough(t *testing.T) {
	t.Parallel()

	page := rateMatrices()
	guide := &stubGuide{destination: &serviceports.ProductGuideDestination{
		Path: "/billing/invoices?item=inv_1", Page: page, Label: "Invoice",
	}}

	_, err := newOpenPageTool(guide).Query(t.Context(), testParams(map[string]any{
		"entity":   "invoice",
		"recordId": "inv_1",
	}))
	require.NoError(t, err)

	require.NotNil(t, guide.resolved)
	assert.Equal(t, "invoice", guide.resolved.Entity)
	assert.Equal(t, "inv_1", guide.resolved.RecordID)
	assert.False(t, guide.resolved.Create)
}

// A refusal — an unknown page, one the person may not open — reaches the
// model as the guide worded it, so it can tell the person why.
func TestOpenPage_CarriesTheRefusal(t *testing.T) {
	t.Parallel()

	refusal := errortypes.NewBusinessError("Invoices is not open to this person")
	tool := newOpenPageTool(&stubGuide{err: refusal})

	_, err := tool.Query(t.Context(), testParams(map[string]any{"page": "/billing/invoices"}))

	require.ErrorIs(t, err, refusal)
}

// Both act for the person in the conversation, so the runtime withholds them
// from every run nobody is watching.
func TestGuideTools_AreSelfScoped(t *testing.T) {
	t.Parallel()

	assert.True(t, serviceports.IsSelfScoped(newFindInTrenovaTool(&stubGuide{})))
	assert.True(t, serviceports.IsSelfScoped(newOpenPageTool(&stubGuide{})))
}

// The record kinds open_page offers are exactly the ones the registry knows
// how to open.
func TestOpenPage_OffersTheRegistrysRecordKinds(t *testing.T) {
	t.Parallel()

	schema := newOpenPageTool(&stubGuide{}).ParamSchema()
	properties, _ := schema["properties"].(map[string]any)
	entity, _ := properties["entity"].(map[string]any)
	offered, _ := entity["enum"].([]string)

	require.Len(t, offered, len(productguide.Default.Records))
	for _, name := range offered {
		_, ok := productguide.Default.Record(name)
		assert.True(t, ok, name)
	}
}
