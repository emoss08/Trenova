package agentevalgate_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/internal/core/services/productguideservice"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/require"
)

const (
	selectionSuite  = "evals/toolselection.yaml"
	paraphraseSuite = "evals/toolselection.paraphrase.yaml"
	selectionFloors = "evals/toolselection.floors.json"
	guideSuite      = "evals/guideselection.yaml"
	guideFloors     = "evals/guideselection.floors.json"
	minSelection    = 40
	minParaphrase   = 20
	minGuide        = 20

	floorsCommand = "go test -tags nofitz -count=1 -run 'AgainstFloors' " +
		"./internal/core/services/agentevalgate/ -update"
)

type evalSuites struct {
	tools      *agentevalgate.SelectionSuite
	paraphrase *agentevalgate.SelectionSuite
	guide      *agentevalgate.SelectionSuite
}

func loadSuites(t *testing.T, kit *agentevalgate.Kit) evalSuites {
	t.Helper()

	tools, err := agentevalgate.LoadSelectionSuite(selectionSuite)
	require.NoError(t, err)
	paraphrase, err := agentevalgate.LoadSelectionSuite(paraphraseSuite)
	require.NoError(t, err)
	guide, err := agentevalgate.LoadSelectionSuite(guideSuite)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(tools.Cases), minSelection,
		"the selection suite keeps at least %d requests", minSelection)
	require.GreaterOrEqual(t, len(paraphrase.Cases), minParaphrase,
		"the paraphrase suite keeps at least %d requests", minParaphrase)
	require.GreaterOrEqual(t, len(guide.Cases), minGuide,
		"the guide suite keeps at least %d questions", minGuide)
	require.Empty(t, kit.UnknownTools(agentevalgate.Combined(tools, paraphrase)),
		"every expected tool must be registered; a renamed tool renames its cases")

	return evalSuites{tools: tools, paraphrase: paraphrase, guide: guide}
}

func (s evalSuites) inputs(kit *agentevalgate.Kit) agentevalgate.EmbeddingInputs {
	documents := kit.Catalog.Items()
	documents = append(documents, agentevalgate.GuideItems()...)

	return agentevalgate.EmbeddingInputs{
		Documents: documents,
		Queries:   agentevalgate.Combined(s.tools, s.paraphrase, s.guide).Requests(),
	}
}

func decodeFloors(t *testing.T, raw []byte) agentevalgate.SelectionFloorSet {
	t.Helper()

	var floors agentevalgate.SelectionFloorSet
	require.NoError(t, sonic.Unmarshal(raw, &floors))

	return floors
}

func readFloors(t *testing.T, path string) agentevalgate.SelectionFloorSet {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err, "create the floors with: %s", floorsCommand)

	return decodeFloors(t, raw)
}

func existingFloors(t *testing.T, path string) agentevalgate.SelectionFloorSet {
	t.Helper()

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return agentevalgate.SelectionFloorSet{}
	}
	require.NoError(t, err)

	return decodeFloors(t, raw)
}

func writeFloors(t *testing.T, path string, floors agentevalgate.SelectionFloorSet) {
	t.Helper()

	encoded, err := agentevalgate.MarshalJSON(floors)
	require.NoError(t, err)
	require.NoError(t, agentevalgate.WriteFile(path, encoded))
}

func recordedFixture(
	t *testing.T,
	suites evalSuites,
	kit *agentevalgate.Kit,
) (*agentevalgate.EmbeddingFixture, bool) {
	t.Helper()

	fixture, err := agentevalgate.LoadEmbeddingFixture(agentevalgate.EmbeddingFixturePath)
	if errors.Is(err, agentevalgate.ErrEmbeddingFixtureMissing) {
		return nil, false
	}
	require.NoError(t, err)
	require.NoError(t, fixture.Stale(suites.inputs(kit)))

	return fixture, true
}

func hybridFixture(
	t *testing.T,
	suites evalSuites,
	kit *agentevalgate.Kit,
) *agentevalgate.EmbeddingFixture {
	t.Helper()

	fixture, ok := recordedFixture(t, suites, kit)
	if ok {
		return fixture
	}

	message := "the hybrid gate needs " + agentevalgate.EmbeddingFixturePath +
		", which has not been recorded. Record it with:\n  " +
		agentevalgate.RecordEmbeddingsCommand
	if strings.EqualFold(os.Getenv(agentevalgate.RequireHybridEnv), "true") {
		t.Fatal(message)
	}
	t.Skip(message)

	return nil
}

func evaluate(
	t *testing.T,
	suite *agentevalgate.SelectionSuite,
	find agentevalgate.Finder,
) agentevalgate.SelectionReport {
	t.Helper()

	report, err := agentevalgate.EvaluateWith(suite, find)
	require.NoError(t, err)

	return report
}

func enforce(
	t *testing.T,
	label string,
	report agentevalgate.SelectionReport,
	floors *agentevalgate.SelectionFloors,
) {
	t.Helper()

	t.Logf("%s: recall@5 %.2f, top-1 %.2f over %d requests",
		label, report.RecallAt5, report.Top1, len(report.Outcomes))
	if misses := report.Misses(); misses != "" {
		t.Logf("%s, requests whose first answer was not an expected one:\n%s", label, misses)
	}
	require.Emptyf(t, report.Spurious,
		"%s found something for a request nothing answers:\n%s",
		label, report.SpuriousMatches())
	require.NotNilf(t, floors, "%s has no floors yet; create them with: %s",
		label, floorsCommand)

	require.GreaterOrEqualf(t, report.RecallAt5, floors.RecallAt5,
		"%s recall@5 fell below its floor: the ranking no longer reaches what a request "+
			"needs\n%s", label, report.Misses())
	require.GreaterOrEqualf(t, report.Top1, floors.Top1,
		"%s top-1 fell below its floor: the ranking now puts something else first\n%s",
		label, report.Misses())
}

func updateFloors(
	t *testing.T,
	path string,
	keyword agentevalgate.SelectionReport,
	hybrid func(*agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport,
	suites evalSuites,
	kit *agentevalgate.Kit,
) {
	t.Helper()

	require.Empty(t, keyword.Spurious, keyword.SpuriousMatches())
	floors := existingFloors(t, path)
	floors.Keyword = keyword.Floors()
	if fixture, ok := recordedFixture(t, suites, kit); ok {
		report := hybrid(fixture)
		require.Empty(t, report.Spurious, report.SpuriousMatches())
		hybridFloors := report.Floors()
		floors.Hybrid = &hybridFloors
	}
	writeFloors(t, path, floors)
}

func TestToolSelectionAgainstFloors(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suites := loadSuites(t, kit)

	keyword := evaluate(t, &agentevalgate.SelectionSuite{
		Cases:   suites.tools.Cases,
		Nothing: agentevalgate.Combined(suites.tools, suites.paraphrase).Nothing,
	}, kit.KeywordFinder())
	paraphrase := evaluate(t, &agentevalgate.SelectionSuite{Cases: suites.paraphrase.Cases},
		kit.KeywordFinder())
	t.Logf("paraphrases by keyword alone (not gated): recall@5 %.2f, top-1 %.2f",
		paraphrase.RecallAt5, paraphrase.Top1)

	if *update {
		updateFloors(t, selectionFloors, keyword,
			func(fixture *agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport {
				return evaluate(t, agentevalgate.Combined(suites.tools, suites.paraphrase),
					kit.HybridFinder(fixture))
			}, suites, kit)
		return
	}

	floors := readFloors(t, selectionFloors)
	enforce(t, "find_tools by keyword", keyword, &floors.Keyword)
}

func TestToolSelectionAgainstFloorsHybrid(t *testing.T) {
	t.Parallel()

	if *update {
		t.Skip("the hybrid floors are written by TestToolSelectionAgainstFloors -update")
	}

	kit := newKit(t)
	suites := loadSuites(t, kit)
	fixture := hybridFixture(t, suites, kit)

	report := evaluate(t, agentevalgate.Combined(suites.tools, suites.paraphrase),
		kit.HybridFinder(fixture))
	enforce(t, "find_tools by keyword and meaning", report,
		readFloors(t, selectionFloors).Hybrid)
}

func TestGuideSelectionAgainstFloors(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suites := loadSuites(t, kit)
	for _, selection := range suites.guide.Cases {
		for _, path := range selection.Expect {
			_, ok := productguide.Default.Page(path)
			require.Truef(t, ok, "%q expects %s, which the guide does not list",
				selection.Request, path)
		}
	}
	guide := productguideservice.NewOffline(productguide.Default)

	keyword := evaluate(t, suites.guide, agentevalgate.GuideKeywordFinder(guide))

	if *update {
		updateFloors(t, guideFloors, keyword,
			func(fixture *agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport {
				return evaluate(t, suites.guide, agentevalgate.GuideHybridFinder(guide, fixture))
			}, suites, kit)
		return
	}

	floors := readFloors(t, guideFloors)
	enforce(t, "find_in_trenova by keyword", keyword, &floors.Keyword)
}

func TestGuideSelectionAgainstFloorsHybrid(t *testing.T) {
	t.Parallel()

	if *update {
		t.Skip("the hybrid floors are written by TestGuideSelectionAgainstFloors -update")
	}

	kit := newKit(t)
	suites := loadSuites(t, kit)
	fixture := hybridFixture(t, suites, kit)
	guide := productguideservice.NewOffline(productguide.Default)

	report := evaluate(t, suites.guide, agentevalgate.GuideHybridFinder(guide, fixture))
	enforce(t, "find_in_trenova by keyword and meaning", report,
		readFloors(t, guideFloors).Hybrid)
}

func TestParaphrasesShareNoWordWithTheirTool(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suites := loadSuites(t, kit)
	for _, selection := range suites.paraphrase.Cases {
		words := make(map[string]struct{}, 8)
		for _, token := range agentsearch.Tokens(selection.Request) {
			words[agentsearch.Singularize(token)] = struct{}{}
		}
		for _, name := range selection.Expect {
			for _, part := range strings.Split(name, "_") {
				_, shared := words[agentsearch.Singularize(part)]
				require.Falsef(t, shared, "%q shares %q with %s; a paraphrase names the "+
					"need, not the tool", selection.Request, part, name)
			}
		}
	}

	requests := suites.tools.Requests()
	for _, request := range suites.paraphrase.Requests() {
		require.Falsef(t, slices.Contains(requests, request), "%q is in both suites", request)
	}
}
