package agentevalgate_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/app/productguideservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/require"
)

const (
	selectionSuite  = "evals/toolselection.yaml"
	paraphraseSuite = "evals/toolselection.paraphrase.yaml"
	selectionFloors = "evals/toolselection.floors.json"
	guideSuite      = "evals/guideselection.yaml"
	guideFloors     = "evals/guideselection.floors.json"
	selectionSplits = "evals/toolselection.splits.yaml"
	minSelection    = 40
	minParaphrase   = 20
	minGuide        = 20

	floorsCommand = "go test -count=1 -run 'AgainstFloors|Splits' " +
		"./internal/modules/agent/app/agentevalgate/ -update"

	selectionSplitsHeader = "# The held-out find_tools cases, by desk and by chat or unattended. Generated from\n" +
		"# the rule in docs/engineering/desk-bench.md (Labels and splits) by running, in\n" +
		"# services/tms:\n" +
		"#   " + floorsCommand + "\n" +
		"# Do not edit by hand, and never edit a description, search term or prompt against\n" +
		"# a case listed here: the gate prints only the development cases' misses for that.\n"
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

type gate struct {
	label  string
	report agentevalgate.SelectionReport
	floors *agentevalgate.SelectionFloors
	held   map[string]struct{}
	quiet  bool
}

func enforce(t *testing.T, g gate) {
	t.Helper()

	t.Logf("%s: recall@5 %.2f, top-1 %.2f over %d requests",
		g.label, g.report.RecallAt5, g.report.Top1, len(g.report.Outcomes))
	if misses := g.report.MissesExcept(g.held); misses != "" && !g.quiet {
		t.Logf("%s, development requests whose first answer was not an expected one:\n%s",
			g.label, misses)
	}
	require.Emptyf(t, g.report.Spurious,
		"%s found something for a request nothing answers:\n%s",
		g.label, g.report.SpuriousMatches())
	require.NotNilf(t, g.floors, "%s has no floors yet; create them with: %s",
		g.label, floorsCommand)

	require.GreaterOrEqualf(t, g.report.RecallAt5, g.floors.RecallAt5,
		"%s recall@5 fell below its floor: the ranking no longer reaches what a request "+
			"needs\n%s", g.label, g.report.MissesExcept(g.held))
	require.GreaterOrEqualf(t, g.report.Top1, g.floors.Top1,
		"%s top-1 fell below its floor: the ranking now puts something else first\n%s",
		g.label, g.report.MissesExcept(g.held))
}

func heldOutFloors(floors agentevalgate.SelectionFloorSet) agentevalgate.SelectionFloorSet {
	if floors.HeldOut == nil {
		return agentevalgate.SelectionFloorSet{}
	}

	return *floors.HeldOut
}

type floorReports struct {
	keyword        agentevalgate.SelectionReport
	hybrid         func(*agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport
	heldOutKeyword *agentevalgate.SelectionReport
	heldOutHybrid  func(*agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport
}

func updateFloors(
	t *testing.T,
	path string,
	reports floorReports,
	suites evalSuites,
	kit *agentevalgate.Kit,
) {
	t.Helper()

	require.Empty(t, reports.keyword.Spurious, reports.keyword.SpuriousMatches())
	floors := existingFloors(t, path)
	floors.Keyword = reports.keyword.Floors()
	if reports.heldOutKeyword != nil {
		floors.HeldOut = &agentevalgate.SelectionFloorSet{Keyword: reports.heldOutKeyword.Floors()}
	}
	if fixture, ok := recordedFixture(t, suites, kit); ok {
		report := reports.hybrid(fixture)
		require.Empty(t, report.Spurious, report.SpuriousMatches())
		hybridFloors := report.Floors()
		floors.Hybrid = &hybridFloors
		if reports.heldOutHybrid != nil {
			heldOut := reports.heldOutHybrid(fixture).Floors()
			floors.HeldOut.Hybrid = &heldOut
		}
	}
	writeFloors(t, path, floors)
}

func selectionSplit(suites evalSuites) agentevalgate.SelectionSplit {
	return agentevalgate.SplitSelection(agentevalgate.Combined(suites.tools, suites.paraphrase))
}

func TestToolSelectionAgainstFloors(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suites := loadSuites(t, kit)
	split := selectionSplit(suites)
	_, heldTools := split.Partition(suites.tools)
	combined := agentevalgate.Combined(suites.tools, suites.paraphrase)
	_, heldCombined := split.Partition(combined)

	keyword := evaluate(t, &agentevalgate.SelectionSuite{
		Cases:   suites.tools.Cases,
		Nothing: combined.Nothing,
	}, kit.KeywordFinder())
	heldOut := evaluate(t, heldTools, kit.KeywordFinder())
	paraphrase := evaluate(t, &agentevalgate.SelectionSuite{Cases: suites.paraphrase.Cases},
		kit.KeywordFinder())
	t.Logf("paraphrases by keyword alone (not gated): recall@5 %.2f, top-1 %.2f",
		paraphrase.RecallAt5, paraphrase.Top1)

	if *update {
		updateFloors(t, selectionFloors, floorReports{
			keyword: keyword,
			hybrid: func(fixture *agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport {
				return evaluate(t, combined, kit.HybridFinder(fixture))
			},
			heldOutKeyword: &heldOut,
			heldOutHybrid: func(fixture *agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport {
				return evaluate(t, heldCombined, kit.HybridFinder(fixture))
			},
		}, suites, kit)
		return
	}

	floors := readFloors(t, selectionFloors)
	enforce(t, gate{
		label:  "find_tools by keyword",
		report: keyword,
		floors: &floors.Keyword,
		held:   split.HeldOut,
	})
	held := heldOutFloors(floors)
	enforce(t, gate{
		label:  "find_tools by keyword, held-out",
		report: heldOut,
		floors: &held.Keyword,
		quiet:  true,
	})
}

func TestToolSelectionAgainstFloorsHybrid(t *testing.T) {
	t.Parallel()

	if *update {
		t.Skip("the hybrid floors are written by TestToolSelectionAgainstFloors -update")
	}

	kit := newKit(t)
	suites := loadSuites(t, kit)
	fixture := hybridFixture(t, suites, kit)
	split := selectionSplit(suites)
	combined := agentevalgate.Combined(suites.tools, suites.paraphrase)
	_, heldCombined := split.Partition(combined)

	floors := readFloors(t, selectionFloors)
	enforce(t, gate{
		label:  "find_tools by keyword and meaning",
		report: evaluate(t, combined, kit.HybridFinder(fixture)),
		floors: floors.Hybrid,
		held:   split.HeldOut,
	})
	enforce(t, gate{
		label:  "find_tools by keyword and meaning, held-out",
		report: evaluate(t, heldCombined, kit.HybridFinder(fixture)),
		floors: heldOutFloors(floors).Hybrid,
		quiet:  true,
	})
}

func TestToolSelectionSplits_MatchTheLedger(t *testing.T) {
	t.Parallel()

	kit := newKit(t)
	suites := loadSuites(t, kit)
	ledger := selectionSplit(suites).Ledger

	encoded, err := ledger.Marshal(selectionSplitsHeader)
	require.NoError(t, err)
	if *update {
		require.NoError(t, agentevalgate.WriteFile(selectionSplits, encoded))
		return
	}

	raw, err := os.ReadFile(selectionSplits)
	require.NoError(t, err, "write the ledger with: %s", floorsCommand)
	recorded, err := evalsplit.ParseLedger(raw)
	require.NoError(t, err)
	diff := ledger.Diff(recorded)
	require.Truef(t, diff.Empty(), "the held-out selection cases changed.\njoined: %s\nleft: %s\n"+
		"A case added to a desk moves the quota, and a renamed or merged tool can move a case "+
		"to another desk; one that joins after living in development may have been tuned "+
		"against. Review, then rewrite the ledger with: %s",
		strings.Join(diff.Joined, "; "), strings.Join(diff.Left, "; "), floorsCommand)
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
		updateFloors(t, guideFloors, floorReports{
			keyword: keyword,
			hybrid: func(fixture *agentevalgate.EmbeddingFixture) agentevalgate.SelectionReport {
				return evaluate(t, suites.guide, agentevalgate.GuideHybridFinder(guide, fixture))
			},
		}, suites, kit)
		return
	}

	floors := readFloors(t, guideFloors)
	enforce(t, gate{label: "agent_find_page by keyword", report: keyword, floors: &floors.Keyword})
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
	enforce(t, gate{
		label:  "agent_find_page by keyword and meaning",
		report: report,
		floors: readFloors(t, guideFloors).Hybrid,
	})
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
