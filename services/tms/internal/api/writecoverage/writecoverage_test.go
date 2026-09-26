package writecoverage_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/api/writecoverage"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func loadReport(t *testing.T) writecoverage.Report {
	t.Helper()

	report, err := writecoverage.Load(".")
	require.NoError(t, err)

	return report
}

func writeByKey(t *testing.T, writes []writecoverage.Write, key string) writecoverage.Write {
	t.Helper()

	idx := slices.IndexFunc(writes, func(w writecoverage.Write) bool { return w.Key == key })
	require.NotEqualf(t, -1, idx, "no write %q", key)

	return writes[idx]
}

func twinRoutes(write writecoverage.Write) []string {
	routes := make([]string, 0, len(write.Twins))
	for _, twin := range write.Twins {
		for _, route := range twin.Routes {
			routes = append(routes, route.String())
		}
	}

	return routes
}

func TestCoverageIsCurrent(t *testing.T) {
	t.Parallel()

	report := loadReport(t)
	problems := report.Problems()
	if len(problems) > 0 {
		t.Fatalf("%d problems:\n  - %s", len(problems), strings.Join(problems, "\n  - "))
	}

	want, err := os.ReadFile(writecoverage.DocumentFile)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(writecoverage.Render(&report)),
		writecoverage.DocumentDisplayPath+" is stale; run "+
			writecoverage.GenerateCommand+" in services/tms and commit it")

	t.Logf("agent write coverage: %d pending writes", report.Pending())
}

func TestRenderIsStable(t *testing.T) {
	t.Parallel()

	report := loadReport(t)
	first := writecoverage.Render(&report)

	reversed := report
	reversed.Writes = slices.Clone(report.Writes)
	slices.Reverse(reversed.Writes)

	assert.Equal(t, string(first), string(writecoverage.Render(&reversed)),
		"the order writes are found in must not move the document")
}

func TestEveryProtectedRouteIsInTheRouteTable(t *testing.T) {
	t.Parallel()

	table, err := api.RouteTable()
	require.NoError(t, err)

	registered := make(map[string]struct{}, len(table))
	for _, info := range table {
		registered[info.Method+" "+info.Path] = struct{}{}
	}

	parsed, err := routelint.ProtectedRoutes("..", "../handlers")
	require.NoError(t, err)

	missing := make([]string, 0)
	for _, route := range parsed {
		if _, ok := registered[route.Key()]; !ok {
			missing = append(missing, route.Package+": "+route.Key())
		}
	}
	assert.Emptyf(t, missing,
		"routes the router registers that the zero-valued route table did not; a "+
			"registration that depends on configuration or a dependency hides them:\n%s",
		strings.Join(missing, "\n"))
}

func TestEnumerationFindsBothSurfaces(t *testing.T) {
	t.Parallel()

	report := loadReport(t)

	create := writeByKey(t, report.Writes, "mutation createShipment")
	assert.Equal(t, writecoverage.KindMutation, create.Kind)
	assert.Equal(t, "shipment", create.Domain)
	assert.Equal(t, "mutationResolver.CreateShipment", create.Handler)
	assert.Contains(t, twinRoutes(create), "POST /api/v1/shipments/",
		"the REST create reaches the same service call and is merged into the mutation")

	hold := writeByKey(t, report.Writes, "POST /api/v1/shipments/:shipmentID/holds/")
	assert.Equal(t, writecoverage.KindRoute, hold.Kind)
	assert.Equal(t, "shipmenthandler.createHold", hold.Handler)

	for idx := range report.Writes {
		write := &report.Writes[idx]
		if write.Kind != writecoverage.KindRoute {
			continue
		}
		for _, route := range write.Routes {
			assert.NotEqualf(t, "GET", route.Method, "%s is a read", route.String())
		}
	}
}

func TestTwinsFollowTheHandlerNameWhenSeveralMutationsMatch(t *testing.T) {
	t.Parallel()

	report := loadReport(t)

	patch := writeByKey(t, report.Writes, "mutation patchTractor")
	update := writeByKey(t, report.Writes, "mutation updateTractor")
	assert.Equal(t, []string{"PATCH /api/v1/tractors/:tractorID/"}, twinRoutes(patch))
	assert.Equal(t, []string{"PUT /api/v1/tractors/:tractorID/"}, twinRoutes(update))
}

func TestClosureHandlersAreListedSeparately(t *testing.T) {
	t.Parallel()

	report := loadReport(t)

	approve := writeByKey(t, report.Writes,
		"POST /api/v1/rate-agreements/:rateAgreementID/approve/")
	reject := writeByKey(t, report.Writes,
		"POST /api/v1/rate-agreements/:rateAgreementID/reject/")
	assert.Equal(t, "rateagreementhandler.review", approve.Handler)
	assert.Len(t, approve.Routes, 1)
	assert.Len(t, reject.Routes, 1)
}

func sampleWrites() []writecoverage.Write {
	return []writecoverage.Write{
		{
			Key:    "mutation createThing",
			Kind:   writecoverage.KindMutation,
			Domain: "thing",
			Endpoint: writecoverage.Endpoint{
				Handler: "mutationResolver.CreateThing",
				Calls:   []string{"things.Service.Create"},
			},
			Twins: []writecoverage.Endpoint{{
				Handler: "thinghandler.create",
				Routes:  []writecoverage.Route{{Method: "POST", Path: "/api/v1/things/"}},
			}},
		},
		{
			Key:    "DELETE /api/v1/things/:id/",
			Kind:   writecoverage.KindRoute,
			Domain: "thing",
			Endpoint: writecoverage.Endpoint{
				Handler: "thinghandler.delete",
				Routes:  []writecoverage.Route{{Method: "DELETE", Path: "/api/v1/things/:id/"}},
			},
		},
	}
}

func sampleTools() map[string]serviceports.ToolPolicy {
	return map[string]serviceports.ToolPolicy{
		"create_thing": {Name: "create_thing", Kind: agent.ToolKindAction},
	}
}

func TestCheckPassesACompleteMapping(t *testing.T) {
	t.Parallel()

	mapping := writecoverage.Mapping{Writes: map[string]writecoverage.Decision{
		"mutation createThing":       {Tools: []string{"create_thing"}},
		"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
	}}

	assert.Empty(t, writecoverage.Check(sampleWrites(), mapping, sampleTools()),
		"a pending write is backlog, not a failure")
}

func TestCheckReportsEveryWayAnEntryGoesWrong(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries map[string]writecoverage.Decision
		want    []string
	}{
		{
			name: "a write without an entry",
			entries: map[string]writecoverage.Decision{
				"mutation createThing": {Tools: []string{"create_thing"}},
			},
			want: []string{`"DELETE /api/v1/things/:id/" (thing, handled by thinghandler.delete) ` +
				"has no entry in " + writecoverage.MappingDisplayPath},
		},
		{
			name: "an entry for a write that is gone",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
				"mutation retiredThing":      {Pending: "Retire a thing."},
			},
			want: []string{`"mutation retiredThing" in ` + writecoverage.MappingDisplayPath +
				" names a write the app no longer exposes"},
		},
		{
			name: "an entry for a route now merged into its mutation",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
				"POST /api/v1/things/":       {Pending: "Create a thing."},
			},
			want: []string{`"POST /api/v1/things/" is now merged into "mutation createThing"`},
		},
		{
			name: "a tool that does not exist",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"make_thing"}},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
			},
			want: []string{`tool "make_thing" is not a registered agent tool`},
		},
		{
			name: "an exemption without a reason",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {Exempt: writecoverage.CategorySecurity},
			},
			want: []string{"exempt: security needs a reason:"},
		},
		{
			name: "an exemption category that does not exist",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {Exempt: "someday", Reason: "Later."},
			},
			want: []string{`exempt: "someday" is not a category`},
		},
		{
			name: "more than one decision",
			entries: map[string]writecoverage.Decision{
				"mutation createThing": {
					Tools:   []string{"create_thing"},
					Pending: "Create a thing.",
				},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
			},
			want: []string{"set exactly one of tools:, exempt: or pending:"},
		},
		{
			name: "no decision",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {},
			},
			want: []string{"give it tools:, exempt: with a reason:, or pending:"},
		},
		{
			name: "a reason without an exemption",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing"}},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing.", Reason: "Why."},
			},
			want: []string{"reason: belongs to an exemption"},
		},
		{
			name: "a tool listed twice",
			entries: map[string]writecoverage.Decision{
				"mutation createThing":       {Tools: []string{"create_thing", "create_thing"}},
				"DELETE /api/v1/things/:id/": {Pending: "Delete a thing."},
			},
			want: []string{`tool "create_thing" is listed twice`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			problems := writecoverage.Check(
				sampleWrites(),
				writecoverage.Mapping{Writes: tt.entries},
				sampleTools(),
			)
			require.Len(t, problems, len(tt.want), strings.Join(problems, "\n"))
			for idx, want := range tt.want {
				assert.Contains(t, problems[idx], want)
				assert.Contains(t, problems[idx], writecoverage.MappingDisplayPath,
					"every message names the file to edit")
			}
		})
	}
}

func TestParseMappingRejectsUnknownFieldsAndDuplicateKeys(t *testing.T) {
	t.Parallel()

	_, err := writecoverage.ParseMapping([]byte(
		"writes:\n  \"mutation createThing\":\n    tool: [create_thing]\n"))
	require.Error(t, err, "a misspelled key must not read as an empty decision")

	_, err = writecoverage.ParseMapping([]byte(
		"writes:\n  \"mutation a\":\n    pending: A.\n  \"mutation a\":\n    pending: B.\n"))
	require.Error(t, err, "a key listed twice must not silently keep the last decision")

	mapping, err := writecoverage.ParseMapping([]byte(
		"writes:\n  \"mutation a\":\n    exempt: read-only\n    reason: Computes only.\n"))
	require.NoError(t, err)
	assert.Equal(t, writecoverage.StateExempt, mapping.Writes["mutation a"].State())
}
