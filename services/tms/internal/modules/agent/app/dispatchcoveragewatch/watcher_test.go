package dispatchcoveragewatch

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type controls struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
	err     error
	reads   int
}

func (c *controls) GetByOrgID(
	context.Context,
	dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	c.reads++
	return c.control, c.err
}

type requests struct {
	tenants []pagination.TenantInfo
}

func (r *requests) RequestRefresh(_ context.Context, tenant pagination.TenantInfo) {
	r.tenants = append(r.tenants, tenant)
}

func watcher(control *dispatchcontrol.DispatchControl, err error) (*Watcher, *controls, *requests) {
	c := &controls{control: control, err: err}
	r := &requests{}
	return New(Params{Controls: c, Requester: r, Logger: zap.NewNop()}), c, r
}

func event(kind agenttrigger.EventKind) *agenteventports.AgentEvent {
	return &agenteventports.AgentEvent{
		Kind:       kind,
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
}

var keepsCoverageProposed = &dispatchcontrol.DispatchControl{
	EnableAutoAssignment: true,
	AutoProposeCoverage:  true,
}

func TestObserveAgentEvent_AsksForCoverageWhenAMoveLosesItsDriver(t *testing.T) {
	t.Parallel()

	for _, kind := range []agenttrigger.EventKind{
		agenttrigger.EventShipmentCreated,
		agenttrigger.EventShipmentMoveUnassigned,
	} {
		w, _, r := watcher(keepsCoverageProposed, nil)
		e := event(kind)

		w.ObserveAgentEvent(t.Context(), e)

		assert.Equal(t, []pagination.TenantInfo{e.TenantInfo}, r.tenants, string(kind))
	}
}

func TestObserveAgentEvent_LeavesAnOrganizationThatDoesNotKeepCoverageProposed(t *testing.T) {
	t.Parallel()

	for _, control := range []*dispatchcontrol.DispatchControl{
		{EnableAutoAssignment: true, AutoProposeCoverage: false},
		{EnableAutoAssignment: false, AutoProposeCoverage: true},
	} {
		w, _, r := watcher(control, nil)

		w.ObserveAgentEvent(t.Context(), event(agenttrigger.EventShipmentCreated))

		assert.Empty(t, r.tenants)
	}
}

func TestObserveAgentEvent_IgnoresEventsThatUncoverNothing(t *testing.T) {
	t.Parallel()

	w, c, r := watcher(keepsCoverageProposed, nil)

	w.ObserveAgentEvent(t.Context(), event(agenttrigger.EventShipmentMoveArrived))

	assert.Empty(t, r.tenants)
	assert.Zero(t, c.reads, "an unrelated event costs no read of the organization's settings")
}

func TestObserveAgentEvent_DoesNotAskWhenTheSettingsCannotBeRead(t *testing.T) {
	t.Parallel()

	w, _, r := watcher(nil, errors.New("connection reset"))

	w.ObserveAgentEvent(t.Context(), event(agenttrigger.EventShipmentMoveUnassigned))

	assert.Empty(t, r.tenants)
}
