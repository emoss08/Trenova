package chainrun

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/emoss08/trenova/internal/testutil/apichain/chainsnap"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const manifestPath = "internal/api/graphql/persisted-documents.json"

func RunDir(t *testing.T, dir string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	absolute, err := filepath.Abs(dir)
	require.NoError(t, err)
	chains, err := apichain.LoadDir(absolute)
	require.NoError(t, err)
	require.NotEmpty(t, chains, "no chains in %s", absolute)

	root, err := seedtest.ServiceRoot()
	require.NoError(t, err)
	manifest, err := apichain.LoadManifest(filepath.Join(root, manifestPath))
	require.NoError(t, err)
	routes, err := api.RouteTable()
	require.NoError(t, err)

	plans := make([]*Plan, 0, len(chains))
	for _, chain := range chains {
		plan, prepareErr := Prepare(chain, manifest, routes)
		require.NoError(t, prepareErr)
		plans = append(plans, plan)
	}

	for _, plan := range plans {
		t.Run(plan.Chain.Name, func(t *testing.T) {
			run(t, plan)
		})
	}
}

type runner struct {
	t       *testing.T
	ctx     context.Context
	plan    *Plan
	app     *App
	actor   Actor
	scope   *apichain.Scope
	session *Session
	norm    *chainsnap.Normalizer
	snap    *chainsnap.Snapshot
	sent    map[string]any
	owned   map[string]bool
}

func run(t *testing.T, plan *Plan) {
	chain := plan.Chain
	app := Boot(t, chain.Name)
	ctx := t.Context()

	actor, err := resolveActor(ctx, app.Admin, chain.Actor)
	require.NoError(t, err)
	fixtures, err := resolveFixtures(ctx, app.Admin, actor, chain.Fixtures)
	require.NoError(t, err)

	unique := strings.ToLower(pulid.MustNew("").String()[16:])
	scope := apichain.NewScope()
	scope.Fixtures = fixtures
	scope.Run = runValues(actor, unique, time.Now().Unix())

	named := map[string]string{
		actor.OrganizationID: "run.organization",
		actor.BusinessUnitID: "run.businessUnit",
		actor.UserID:         "run.user",
	}
	owned := map[string]bool{}
	for _, name := range slices.Sorted(maps.Keys(fixtures)) {
		if id, ok := fixtures[name].(string); ok {
			named[id] = "fixture." + name
			owned[id] = true
		}
	}

	session, err := Login(ctx, app.URL, actor.Email)
	require.NoError(t, err, "log in (API log: %s)", app.LogPath)

	r := &runner{
		t:       t,
		ctx:     ctx,
		plan:    plan,
		app:     app,
		actor:   actor,
		scope:   scope,
		session: session,
		norm:    chainsnap.NewNormalizer(named, chain.Volatile, map[string]string{unique: "<run.unique>"}),
		snap:    &chainsnap.Snapshot{Chain: chain.Name},
		sent:    map[string]any{},
		owned:   owned,
	}

	for _, call := range plan.Calls {
		r.step(call)
	}
	r.probes()
	r.compare()
}

func (r *runner) step(call Call) {
	t := r.t
	step := call.Step

	befores := make([]any, len(step.Effects))
	for i := range step.Effects {
		effect := &step.Effects[i]
		if !effect.NeedsBefore() {
			continue
		}
		befores[i] = r.read(step.ID, effect, "before")
	}

	r.app.Effects.Drain()
	r.app.RoundTrips()

	request, result, err := r.execute(r.session, call, r.scope)
	require.NoError(t, err, "step %s (%s)", step.ID, call.Label())
	trips := r.app.RoundTrips()
	side := r.app.Effects.Drain()

	r.requireOutcome(call, result)

	exports := make(map[string]any, len(step.Export))
	for _, name := range slices.Sorted(maps.Keys(step.Export)) {
		value, ok := apichain.Get(result.Body, step.Export[name])
		require.True(t, ok, "step %s (%s): export %s: nothing at %s in %s",
			step.ID, call.Label(), name, step.Export[name], describe(result.Body))
		exports[name] = value
		if id, isText := value.(string); isText {
			r.owned[id] = true
		}
	}
	if !call.IsRead() {
		r.claim(result.Body)
	}
	r.scope.Steps[step.ID] = apichain.StepValues{
		Request:  request,
		Response: result.Body,
		Exports:  exports,
	}
	r.sent[step.ID] = request

	for _, path := range slices.Sorted(maps.Keys(step.Expect)) {
		want, resolveErr := r.scope.Resolve(step.Expect[path])
		require.NoError(t, resolveErr, "step %s: expect %s", step.ID, path)
		got, found := apichain.Get(result.Body, path)
		if !found || !sameValue(want, got) {
			t.Fatalf("step %s (%s): expect %s = %v, got %v (found: %t)",
				step.ID, call.Label(), path, want, got, found)
		}
	}

	record := chainsnap.StepRecord{
		ID:         step.ID,
		Call:       call.Label(),
		Write:      !call.IsRead(),
		Request:    r.norm.Value(nil, request),
		Status:     result.Status,
		Code:       result.Code,
		Response:   r.norm.Value(nil, result.Body),
		RoundTrips: trips,
	}
	if !side.Empty() {
		record.Side = r.norm.Value(nil, plain(side))
	}

	for i := range step.Effects {
		effect := &step.Effects[i]
		after := r.read(step.ID, effect, "after")
		expected := effectExpectation(effect)
		resolved, resolveErr := r.scope.Resolve(expected)
		require.NoError(t, resolveErr, "step %s: effect %d", step.ID, i+1)
		if checkErr := checkEffect(effect, resolved, befores[i], after); checkErr != nil {
			t.Fatalf("step %s (%s): effect on %s %s: %v",
				step.ID, call.Label(), effect.Read, effect.Path, checkErr)
		}
		record.Effects = append(record.Effects, chainsnap.EffectRecord{
			Read:   effect.Read,
			Path:   effect.Path,
			Rule:   effectRule(effect),
			Before: r.norm.Value(nil, befores[i]),
			After:  r.norm.Value(nil, after),
		})
	}

	r.snap.Steps = append(r.snap.Steps, record)
}

func (r *runner) requireOutcome(call Call, result Result) {
	step := call.Step
	if step.ExpectError != "" {
		if result.Code != step.ExpectError {
			r.t.Fatalf("step %s (%s): want error %s, got %d %s: %s (API log: %s)",
				step.ID, call.Label(), step.ExpectError, result.Status, result.Code,
				describe(result.Body), r.app.LogPath)
		}
		return
	}
	if result.Failed() {
		r.t.Fatalf("step %s (%s) failed: %d %s: %s (API log: %s)",
			step.ID, call.Label(), result.Status, result.Code, describe(result.Body),
			r.app.LogPath)
	}
}

func (r *runner) read(stepID string, effect *apichain.Effect, when string) any {
	call, _ := r.plan.Call(effect.Read)
	_, result, err := r.execute(r.session, call, r.scope)
	require.NoError(r.t, err, "step %s: effect read %s %s the write", stepID, effect.Read, when)
	if result.Failed() {
		r.t.Fatalf("step %s: effect read %s %s the write failed: %d %s: %s",
			stepID, effect.Read, when, result.Status, result.Code, describe(result.Body))
	}
	value, ok := apichain.Get(result.Body, effect.Path)
	require.True(r.t, ok, "step %s: effect read %s %s the write: nothing at %s in %s",
		stepID, effect.Read, when, effect.Path, describe(result.Body))

	return value
}

func (r *runner) execute(session *Session, call Call, scope *apichain.Scope) (any, Result, error) {
	step := call.Step
	if step.IsGraphQL() {
		vars, err := scope.Resolve(anyMap(step.Vars))
		if err != nil {
			return nil, Result{}, err
		}
		result, err := session.GraphQL(r.ctx, call.Operation, vars)
		return map[string]any{"variables": vars}, result, err
	}

	path, err := scope.ResolveString(step.Route())
	if err != nil {
		return nil, Result{}, err
	}
	body, err := scope.Resolve(step.Body)
	if err != nil {
		return nil, Result{}, err
	}
	result, err := session.REST(r.ctx, step.Method(), path, body)
	request := map[string]any{"path": path}
	if body != nil {
		request["body"] = body
	}

	return request, result, err
}

func (r *runner) replay(session *Session, call Call) (Result, error) {
	request := r.sent[call.Step.ID]
	if call.Step.IsGraphQL() {
		vars, _ := apichain.Get(request, "variables")
		return session.GraphQL(r.ctx, call.Operation, vars)
	}
	path, _ := apichain.Get(request, "path")
	body, _ := apichain.Get(request, "body")

	return session.REST(r.ctx, call.Step.Method(), fmt.Sprint(path), body)
}

func (r *runner) claim(body any) {
	var walk func(key string, value any)
	walk = func(key string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for k, v := range typed {
				walk(k, v)
			}
		case []any:
			for _, v := range typed {
				walk(key, v)
			}
		case string:
			if key == "id" && chainsnap.IsID(typed) {
				r.owned[typed] = true
			}
		}
	}
	walk("", body)
}

func (r *runner) compare() {
	outcome, err := chainsnap.Check(r.plan.Chain.GoldenPath(), r.snap)
	require.NoError(r.t, err)
	if outcome.Updated {
		r.t.Logf("wrote %s: review it like code before committing", r.plan.Chain.GoldenPath())
		return
	}
	if len(outcome.Diffs) == 0 {
		return
	}

	var report strings.Builder
	fmt.Fprintf(&report, "%s drifted from %s (%s=1 rewrites it; review that diff like code):\n",
		r.plan.Chain.Name, filepath.Base(r.plan.Chain.GoldenPath()), chainsnap.UpdateEnv)
	for _, diff := range outcome.Diffs {
		if diff.Step == "" {
			fmt.Fprintf(&report, "  %s\n", diff)
			continue
		}
		fmt.Fprintf(&report, "  step %s (%s): %s\n", diff.Step, diff.Call, diff)
		if suspects := chainsnap.Suspects(r.snap, diff); len(suspects) > 0 {
			fmt.Fprintf(&report, "    suspect writes: %s\n", strings.Join(suspects, ", "))
		}
	}
	fmt.Fprintf(&report, "API log: %s", r.app.LogPath)
	r.t.Error(report.String())
}

func effectExpectation(effect *apichain.Effect) any {
	switch {
	case effect.Equals != nil:
		return effect.Equals
	case effect.Increase != nil:
		return effect.Increase
	case effect.Decrease != nil:
		return effect.Decrease
	default:
		return nil
	}
}

func anyMap(vars map[string]any) any {
	if vars == nil {
		return nil
	}

	return vars
}
