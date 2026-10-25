package customerstatustrigger

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingRequester struct {
	requested []pulid.ID
}

func (r *recordingRequester) RequestEvaluation(
	_ context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	r.requested = append(r.requested, shipmentID)
}

type fakeFacts struct {
	customertrackingports.FactRepository
	tenants []pagination.TenantInfo
	err     error
	calls   int
}

func (f *fakeFacts) HasNotifyingCustomers(
	_ context.Context,
	tenant pagination.TenantInfo,
) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	return slices.Contains(f.tenants, tenant), nil
}

var notifying = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

func newTrigger(facts *fakeFacts) (*Trigger, *recordingRequester) {
	requester := &recordingRequester{}
	return New(Params{Requester: requester, Facts: facts, Logger: zap.NewNop()}), requester
}

func event(eventType shipmentevent.Type, tenant pagination.TenantInfo) *shipmentevent.Event {
	return &shipmentevent.Event{
		Type:           eventType,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ShipmentID:     pulid.MustNew("shp_"),
	}
}

func TestTrigger_EvaluatesStopAndStatusChangesForNotifyingOrganizations(t *testing.T) {
	facts := &fakeFacts{tenants: []pagination.TenantInfo{notifying}}
	trigger, requester := newTrigger(facts)

	arrived := event(shipmentevent.TypeMoveArrived, notifying)
	require.NoError(t, trigger.OnShipmentEvent(t.Context(), arrived))
	require.NoError(t, trigger.OnShipmentEvent(t.Context(), event(shipmentevent.TypeCommentPosted, notifying)))

	assert.Equal(t, []pulid.ID{arrived.ShipmentID}, requester.requested)
}

func TestTrigger_SkipsOrganizationsWithoutRules(t *testing.T) {
	facts := &fakeFacts{tenants: []pagination.TenantInfo{notifying}}
	trigger, requester := newTrigger(facts)
	quiet := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	require.NoError(t, trigger.OnShipmentEvent(t.Context(), event(shipmentevent.TypeMoveDeparted, quiet)))
	trigger.StopEtasRefreshed(t.Context(), quiet, []pulid.ID{pulid.MustNew("shp_")})

	assert.Empty(t, requester.requested)
	assert.Equal(t, 2, facts.calls, "each change asks the repository's cached read")
}

func TestTrigger_EtaRefreshesAskForEachShipment(t *testing.T) {
	trigger, requester := newTrigger(&fakeFacts{tenants: []pagination.TenantInfo{notifying}})
	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")

	trigger.StopEtasRefreshed(t.Context(), notifying, []pulid.ID{first, second})

	assert.Equal(t, []pulid.ID{first, second}, requester.requested)
}

func TestTrigger_WhenTheAnswerCannotBeReadItEvaluates(t *testing.T) {
	trigger, requester := newTrigger(&fakeFacts{err: errors.New("database unavailable")})

	require.NoError(t, trigger.OnShipmentEvent(t.Context(), event(shipmentevent.TypeStatusChanged, notifying)))

	assert.Len(t, requester.requested, 1)
}
