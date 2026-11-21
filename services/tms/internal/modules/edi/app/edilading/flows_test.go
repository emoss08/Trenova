package edilading

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type flowsFixture struct {
	flows   *Flows
	repo    *ladingportsmock.MockFlowRepository
	program *Program
	flow    string
	tenant  pagination.TenantInfo
	partner pulid.ID
}

func newFlowsFixture(t *testing.T) *flowsFixture {
	t.Helper()

	client := lading.Default()
	starter, err := client.Starter(t.Context(), &lading.StarterRequest{
		Name:        "partner",
		Label:       "Partner",
		Role:        lading.RoleCarrier,
		Them:        &[2]string{"ZZ", "PARTNER"},
		Us:          &[2]string{"ZZ", "TRENOVA"},
		Usage:       "production",
		Acknowledge: "997",
	})
	require.NoError(t, err)
	encoded, err := lading.Encode(
		&lading.Project{Files: starter.Files, Samples: starter.Samples},
		&lading.Context{Org: lading.Org{Name: "Trenova", SCAC: "TRNV", Zone: "America/Chicago"}},
	)
	require.NoError(t, err)
	check, err := client.Check(t.Context(), encoded)
	require.NoError(t, err)
	require.Zero(t, check.Errors)
	flow, _ := check.Receiving("204")
	require.NotNil(t, flow)

	repo := ladingportsmock.NewMockFlowRepository(t)
	repo.EXPECT().AppendEvent(mock.Anything, mock.Anything).Return(nil).Maybe()
	return &flowsFixture{
		flows: &Flows{
			l:       zap.NewNop(),
			runtime: &Runtime{client: client},
			flows:   repo,
		},
		repo: repo,
		program: &Program{
			Project: &ladingproject.LadingProject{
				ID:       pulid.MustNew("ldp_"),
				Settings: map[string]any{},
			},
			Version:  &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_")},
			Encoded:  encoded,
			Manifest: check,
		},
		flow:    flow.Name,
		tenant:  pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		partner: pulid.MustNew("edip_"),
	}
}

func (f *flowsFixture) request(key string, event lading.FlowEvent) *DispatchRequest {
	return &DispatchRequest{
		TenantInfo: f.tenant,
		PartnerID:  f.partner,
		Program:    f.program,
		Flow:       f.flow,
		Key:        key,
		Event:      event,
	}
}

func (f *flowsFixture) noInstance(key string) {
	f.repo.EXPECT().
		GetInstanceByKey(mock.Anything, mock.MatchedBy(func(req ladingports.GetFlowInstanceByKeyRequest) bool {
			return req.Key == key && req.Flow == f.flow && req.PartnerID == f.partner
		})).
		Return(nil, errortypes.NewNotFoundError("FlowInstance not found within your organization")).
		Once()
}

func loadEvent() lading.FlowEvent {
	return lading.FlowEvent{
		Kind: lading.EventLoad,
		Load: &lading.LoadState{Exists: true, State: lading.LoadOpen},
	}
}

func TestDispatch_AMessageTheFlowDoesNotTakeOpensNoInstance(t *testing.T) {
	t.Parallel()

	f := newFlowsFixture(t)
	f.noInstance("STRAY-1")

	dispatched, err := f.flows.Dispatch(t.Context(), f.request("STRAY-1", lading.FlowEvent{
		Kind:     lading.EventReceived,
		Set:      "856",
		Document: []byte(`{"key":"STRAY-1"}`),
	}))

	require.NoError(t, err)
	require.False(t, dispatched.Opened())
	require.Equal(t, "STRAY-1", dispatched.Instance.Key)
	f.repo.AssertNotCalled(t, "CreateInstance", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "SaveInstance", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "AppendEvent", mock.Anything, mock.Anything)
}

func TestDispatch_OpensTheInstanceWithTheStepsResult(t *testing.T) {
	t.Parallel()

	f := newFlowsFixture(t)
	f.noInstance("LOAD-1")
	shipmentID := pulid.MustNew("shp_")
	var created *ladingproject.FlowInstance
	f.repo.EXPECT().CreateInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, instance *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			created = instance
			instance.ID = pulid.MustNew("edfi_")
			return instance, nil
		}).
		Once()

	req := f.request("LOAD-1", loadEvent())
	req.ShipmentID = shipmentID
	dispatched, err := f.flows.Dispatch(t.Context(), req)

	require.NoError(t, err)
	require.True(t, dispatched.Opened())
	require.NotNil(t, created)
	require.True(
		t,
		lading.FlowStarted(created.State),
		"the instance is stored with the step's state",
	)
	require.Equal(t, shipmentID, created.ShipmentID)
	require.Equal(t, f.program.Version.ID, created.VersionID)
	f.repo.AssertNotCalled(t, "SaveInstance", mock.Anything, mock.Anything)
}

func TestDispatch_ADeliveryThatLosesTheRaceToOpenStepsTheStoredInstance(t *testing.T) {
	t.Parallel()

	f := newFlowsFixture(t)
	f.noInstance("LOAD-2")
	f.repo.EXPECT().CreateInstance(mock.Anything, mock.Anything).
		Return(nil, ladingports.ErrFlowInstanceExists).
		Once()
	stored := &ladingproject.FlowInstance{
		ID:             pulid.MustNew("edfi_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		EDIPartnerID:   f.partner,
		Flow:           f.flow,
		Key:            "LOAD-2",
		State:          []byte(`{"key":"LOAD-2","sends":1}`),
		Status:         ladingproject.FlowInstanceStatusOpen,
		Version:        3,
	}
	f.repo.EXPECT().
		GetInstanceByKey(mock.Anything, mock.MatchedBy(func(req ladingports.GetFlowInstanceByKeyRequest) bool {
			return req.Key == "LOAD-2"
		})).
		Return(stored, nil).
		Once()
	f.repo.EXPECT().
		SaveInstance(mock.Anything, mock.MatchedBy(func(instance *ladingproject.FlowInstance) bool {
			return instance.ID == stored.ID && instance.Version == 3
		})).
		RunAndReturn(func(_ context.Context, instance *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			return instance, nil
		}).
		Once()

	dispatched, err := f.flows.Dispatch(t.Context(), f.request("LOAD-2", loadEvent()))

	require.NoError(t, err)
	require.Equal(t, stored.ID, dispatched.Instance.ID)
	require.Contains(t, string(dispatched.Instance.State), `"sends":1`)
}

func TestDispatch_AWinnerThatRolledBackLeavesTheKeyToOpenAgain(t *testing.T) {
	t.Parallel()

	f := newFlowsFixture(t)
	f.noInstance("LOAD-3")
	f.repo.EXPECT().CreateInstance(mock.Anything, mock.Anything).
		Return(nil, ladingports.ErrFlowInstanceExists).
		Once()
	f.noInstance("LOAD-3")
	f.repo.EXPECT().CreateInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, instance *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			instance.ID = pulid.MustNew("edfi_")
			return instance, nil
		}).
		Once()

	dispatched, err := f.flows.Dispatch(t.Context(), f.request("LOAD-3", loadEvent()))

	require.NoError(t, err)
	require.True(t, dispatched.Opened())
}

func TestOpens_IsWhetherAStepLeavesAnythingAnInstanceMustHold(t *testing.T) {
	t.Parallel()

	empty := []byte(`{"documents":{},"load":{"exists":false,"state":""},"sends":0}`)
	wake := int64(1_700_000_000)
	tests := []struct {
		name string
		step lading.FlowStep
		want bool
	}{
		{
			name: "a rejected message",
			step: lading.FlowStep{
				State: empty,
				Actions: []lading.FlowAction{{
					Kind:     lading.ActionAcknowledge,
					Set:      "997",
					Rejected: true,
					Message:  &wasmmodule.Message{},
				}},
			},
			want: false,
		},
		{
			name: "an ignored message",
			step: lading.FlowStep{
				State:   empty,
				Actions: []lading.FlowAction{{Kind: lading.ActionIgnore}},
			},
			want: false,
		},
		{
			name: "a received document",
			step: lading.FlowStep{State: []byte(`{"documents":{"status":{}}}`)},
			want: true,
		},
		{name: "a timer", step: lading.FlowStep{State: empty, Wake: &wake}, want: true},
		{
			name: "a hold",
			step: lading.FlowStep{
				State:   empty,
				Actions: []lading.FlowAction{{Kind: lading.ActionHold}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, opens(&tt.step))
		})
	}
}

func TestApply_AClosedInstanceLeftWithWorkReopensAndAFinishedOneCloses(t *testing.T) {
	t.Parallel()

	req := &DispatchRequest{
		Program: &Program{
			Project: &ladingproject.LadingProject{ID: pulid.MustNew("ldp_")},
			Version: &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_")},
		},
		Event: lading.FlowEvent{Kind: lading.EventTrenova},
	}
	pending := &lading.FlowStep{State: []byte(
		`{"load":{"exists":true,"state":"invoiced"},"pending":[{"id":"s3","set":"210","control":"0007","due":1,"resends":0}]}`,
	)}
	done := &lading.FlowStep{
		State: []byte(`{"load":{"exists":true,"state":"invoiced"},"pending":[]}`),
	}

	closed := &ladingproject.FlowInstance{Status: ladingproject.FlowInstanceStatusClosed}
	apply(closed, req, pending)
	require.Equal(t, ladingproject.FlowInstanceStatusOpen, closed.Status)

	apply(closed, req, done)
	require.Equal(t, ladingproject.FlowInstanceStatusClosed, closed.Status)
}

func TestApply_AnEscalationFromARejectedAckCarriesTheAcknowledgmentsCodes(t *testing.T) {
	t.Parallel()

	req := &DispatchRequest{
		Program: &Program{
			Project: &ladingproject.LadingProject{ID: pulid.MustNew("ldp_")},
			Version: &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_")},
		},
		Event:     lading.FlowEvent{Kind: lading.EventAck, Control: "0042"},
		AckDetail: "acknowledgment code R; syntax error codes 5",
	}
	step := &lading.FlowStep{
		State: []byte(`{"load":{"exists":true,"state":"dispatched"},"pending":[]}`),
		Actions: []lading.FlowAction{{
			Kind: lading.ActionEscalate,
			Message: &wasmmodule.Message{
				Template: "The partner rejected {0} {1}: {2}",
				Args:     []any{"214", "0042", ""},
			},
		}},
	}
	instance := &ladingproject.FlowInstance{Status: ladingproject.FlowInstanceStatusOpen}

	apply(instance, req, step)

	require.Equal(t, ladingproject.FlowInstanceStatusEscalated, instance.Status)
	require.NotNil(t, instance.Reason)
	require.Contains(t, instance.Reason.String(), "syntax error codes 5")
	require.Contains(t, step.Actions[0].Message.String(), "syntax error codes 5")
}
