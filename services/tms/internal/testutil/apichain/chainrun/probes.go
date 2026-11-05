package chainrun

import (
	"maps"
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

	owned := slices.Sorted(maps.Keys(r.owned))
	for _, call := range r.plan.Calls {
		if call.Step.ExpectError != "" || !mentions(r.sent[call.Step.ID], owned) {
			continue
		}
		result, replayErr := r.replay(session, call)
		require.NoError(r.t, replayErr, "other-tenant probe: step %s", call.Step.ID)
		r.record(apichain.ProbeOtherTenant, call, result)

		if call.Step.IsOpenTo(apichain.ProbeOtherTenant) {
			continue
		}
		if !call.IsRead() && !result.Failed() {
			r.t.Errorf("other-tenant probe: step %s (%s) wrote with another tenant's records "+
				"and must be refused: %d %s", call.Step.ID, call.Label(), result.Status,
				describe(result.Body))
			continue
		}
		if leaked := leakedIDs(result.Body, owned); len(leaked) > 0 {
			r.t.Errorf("other-tenant probe: step %s (%s) answered another tenant with %s: %s",
				call.Step.ID, call.Label(), strings.Join(leaked, ", "), describe(result.Body))
		}
	}
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
