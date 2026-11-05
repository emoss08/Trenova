package chainrun

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/emoss08/trenova/internal/testutil/apichain/chainsnap"
	"github.com/stretchr/testify/require"
)

func (r *runner) probes() {
	for _, probe := range r.plan.Chain.Probes {
		switch probe {
		case apichain.ProbeUnauthenticated:
			r.refuseEveryStep(probe, Anonymous(r.app.URL))
		case apichain.ProbeNoPermission:
			email := noPermissionUser(r.t, r.app.Admin, r.actor)
			session, err := Login(r.ctx, r.app.URL, email)
			require.NoError(r.t, err, "log in as the no-permission user")
			r.refuseEveryStep(probe, session)
		case apichain.ProbeOtherTenant:
			r.otherTenant()
		}
	}
}

func (r *runner) refuseEveryStep(probe apichain.Probe, session *Session) {
	for _, call := range r.plan.Calls {
		if call.Step.ExpectError != "" {
			continue
		}
		result, err := r.replay(session, call)
		require.NoError(r.t, err, "%s probe: step %s", probe, call.Step.ID)
		r.record(probe, call, result)
		if !result.Failed() && !call.Step.IsOpenTo(probe) {
			r.t.Errorf("%s probe: step %s (%s) succeeded and must be refused: %d %s",
				probe, call.Step.ID, call.Label(), result.Status, describe(result.Body))
		}
	}
}

func (r *runner) otherTenant() {
	other, err := otherOrganization(r.ctx, r.app.Admin, r.actor)
	require.NoError(r.t, err)
	session, err := Login(r.ctx, r.app.URL, r.actor.Email)
	require.NoError(r.t, err, "log in for the other-tenant probe")
	require.NoError(r.t, session.SwitchOrganization(r.ctx, other))

	before := r.readState()
	owned := slices.Sorted(maps.Keys(r.owned))
	for _, call := range r.plan.Calls {
		request := r.sent[call.Step.ID]
		if call.Step.ExpectError != "" || !mentions(request, owned) {
			continue
		}
		result, replayErr := r.replay(session, call)
		require.NoError(r.t, replayErr, "other-tenant probe: step %s", call.Step.ID)
		r.record(apichain.ProbeOtherTenant, call, result)

		if result.Failed() || call.Step.IsOpenTo(apichain.ProbeOtherTenant) {
			continue
		}
		if leaked := leakedIDs(result.Body, unsent(owned, request)); len(leaked) > 0 {
			r.t.Errorf("other-tenant probe: step %s (%s) answered another tenant with %s: %s",
				call.Step.ID, call.Label(), strings.Join(leaked, ", "), describe(result.Body))
		}
	}

	after := r.readState()
	for _, id := range slices.Sorted(maps.Keys(before)) {
		if !reflect.DeepEqual(before[id], after[id]) {
			r.t.Errorf("other-tenant probe: the actor's read %s changed while another tenant "+
				"replayed the chain:\nbefore: %s\nafter:  %s",
				id, describeAll(before[id]), describeAll(after[id]))
		}
	}
}

func (r *runner) readState() map[string]Result {
	state := map[string]Result{}
	for _, call := range r.plan.Calls {
		if !call.IsRead() {
			continue
		}
		result, err := r.replay(r.session, call)
		require.NoError(r.t, err, "re-read %s", call.Step.ID)
		state[call.Step.ID] = result
	}

	return state
}

func unsent(owned []string, request any) []string {
	text := describeAll(request)

	return slices.DeleteFunc(slices.Clone(owned), func(id string) bool {
		return strings.Contains(text, id)
	})
}

func (r *runner) record(probe apichain.Probe, call Call, result Result) {
	r.snap.Probes = append(r.snap.Probes, chainsnap.ProbeRecord{
		Probe:  string(probe),
		Step:   call.Step.ID,
		Status: result.Status,
		Code:   result.Code,
	})
}

func mentions(request any, ids []string) bool {
	text := describeAll(request)
	for _, id := range ids {
		if strings.Contains(text, id) {
			return true
		}
	}

	return false
}

func leakedIDs(body any, ids []string) []string {
	if body == nil {
		return nil
	}
	text := describeAll(body)
	var leaked []string
	for _, id := range ids {
		if strings.Contains(text, id) {
			leaked = append(leaked, id)
		}
	}

	return leaked
}
