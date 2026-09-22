package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func deskParams(params map[string]any) serviceports.ToolExecuteParams {
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")

	return serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Params:         params,
		IdempotencyKey: "key-1",
	}
}

// ------------------------------------------------------------ detention

type stubEscalator struct {
	occurrence *detention.DetentionOccurrence
	escalated  *detentionservice.EscalateParams
}

func (s *stubEscalator) Escalate(
	_ context.Context,
	p detentionservice.EscalateParams,
) (*detention.DetentionOccurrence, error) {
	s.escalated = &p

	return s.occurrence, nil
}

func (s *stubEscalator) GetOccurrenceDetail(
	context.Context,
	*repositories.GetDetentionOccurrenceByIDRequest,
) (*detentionservice.OccurrenceDetail, error) {
	return &detentionservice.OccurrenceDetail{Occurrence: s.occurrence}, nil
}

// Escalating leaves the money alone. Waiving, approving and disputing are a
// person's; a desk that could quietly change a charge is a desk that can
// give away revenue on a schedule.
func TestEscalateDetention_HandsOverWithoutTouchingTheCharge(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("do_")
	stub := &stubEscalator{occurrence: &detention.DetentionOccurrence{
		ID:              occurrenceID,
		Status:          detention.OccurrenceStatusAccruing,
		BillableMinutes: 90,
	}}
	tool := newEscalateDetentionTool(stub)

	require.Equal(t, permission.ResourceDetentionPolicy, tool.PermissionResource())
	require.Equal(t, permission.OpUpdate, tool.PermissionOperation())
	require.Equal(t, agent.TierPropose, tool.DefaultAutonomyTier())
	require.True(t, tool.Reversible())

	params := deskParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "the notice window closed and the customer has no recipients",
	})

	simulated, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	require.Contains(t, simulated.Summary, "90 billable minutes")
	for _, change := range simulated.Changes {
		require.NotContains(t, []string{"status", "billableAmount", "billableMinutes"},
			change.Field, "escalating must not touch the charge")
	}

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.escalated)
	require.Equal(t, occurrenceID, stub.escalated.OccurrenceID)
}

func TestEscalateDetention_RefusesAWaivedClock(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("do_")
	stub := &stubEscalator{occurrence: &detention.DetentionOccurrence{
		ID:     occurrenceID,
		Status: detention.OccurrenceStatusWaived,
	}}
	tool := newEscalateDetentionTool(stub)

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "please look at this",
	}))
	require.ErrorContains(t, err, "waived")
}

// ----------------------------------------------------------- credential

type stubCredentialActor struct {
	asked *workercredentialservice.RenewalRequest
}

func (s *stubCredentialActor) RequestRenewal(
	_ context.Context,
	req workercredentialservice.RenewalRequest,
) error {
	s.asked = &req

	return nil
}

// One ask covering every paper, keyed so a retried proposal does not tell
// the driver twice.
func TestRequestCredentialRenewal_AsksOncePerDriver(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	first, second := pulid.MustNew("wcred_"), pulid.MustNew("wcred_")
	stub := &stubCredentialActor{}
	tool := newRequestCredentialRenewalTool(stub)

	require.True(t, tool.RequiresIdempotencyKey())
	require.Equal(t, permission.ResourceWorkerCredential, tool.PermissionResource())

	params := deskParams(map[string]any{
		"workerId":      workerID.String(),
		"credentialIds": []any{first.String(), second.String()},
		"note":          "Your CDL and medical card are both due this month.",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.asked)
	require.Equal(t, workerID, stub.asked.WorkerID)
	require.Equal(t, []pulid.ID{first, second}, stub.asked.CredentialIDs)
	require.Equal(t, "key-1", stub.asked.CorrelationID)
}

func TestRequestCredentialRenewal_RefusesAnEmptyAsk(t *testing.T) {
	t.Parallel()

	tool := newRequestCredentialRenewalTool(&stubCredentialActor{})
	err := tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(map[string]any{
		"workerId":      pulid.MustNew("wrk_").String(),
		"credentialIds": []any{},
		"note":          "please renew",
	}))
	require.ErrorContains(t, err, "at least one credential")
}

// -------------------------------------------------------- dispatch hold

type stubWorkerHolder struct {
	held  *worker.Worker
	given *workerservice.DispatchHoldRequest
}

func (s *stubWorkerHolder) SetDispatchHold(
	_ context.Context,
	req workerservice.DispatchHoldRequest,
) (*worker.Worker, error) {
	s.given = &req

	return s.held, nil
}

func (s *stubWorkerHolder) Get(
	context.Context,
	repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return s.held, nil
}

// The hold is its own permission, not an edit of the worker record: taking
// somebody off the board until a paper is renewed is a dispatch decision,
// and an agent may never edit employment.
func TestPlaceWorkerDispatchHold_IsItsOwnPermission(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	stub := &stubWorkerHolder{held: &worker.Worker{ID: workerID, CanBeAssigned: true}}
	tool := newPlaceWorkerDispatchHoldTool(stub)

	require.Equal(t, permission.ResourceWorkerDispatchHold, tool.PermissionResource())
	require.Equal(t, permission.OpCreate, tool.PermissionOperation())
	require.True(t, tool.Reversible())
	require.False(t, permission.IsAgentAllowed(permission.ResourceWorker, permission.OpUpdate),
		"editing a worker record stays closed to an agent")

	params := deskParams(map[string]any{
		"workerId": workerID.String(),
		"reason":   "medical card lapsed on the 3rd",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.given)
	require.True(t, stub.given.Held)
	require.Equal(t, workerID, stub.given.WorkerID)
}

func TestPlaceWorkerDispatchHold_RefusesADriverAlreadyHeld(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	stub := &stubWorkerHolder{held: &worker.Worker{ID: workerID, CanBeAssigned: false}}
	tool := newPlaceWorkerDispatchHoldTool(stub)

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(map[string]any{
		"workerId": workerID.String(),
		"reason":   "medical card lapsed",
	}))
	require.ErrorContains(t, err, "already off dispatch")
}

// -------------------------------------------------------- carrier risk

type stubCarrierIntel struct {
	event        *carrierintel.CarrierIntelEvent
	acknowledged []pulid.ID
	resolved     *carrierintelservice.ResolveEventRequest
}

func (s *stubCarrierIntel) AcknowledgeEvents(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (int, error) {
	s.acknowledged = ids

	return len(ids), nil
}

func (s *stubCarrierIntel) ResolveEvent(
	_ context.Context,
	req *carrierintelservice.ResolveEventRequest,
) (*carrierintel.CarrierIntelEvent, error) {
	s.resolved = req

	return s.event, nil
}

func (s *stubCarrierIntel) GetEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrierintel.CarrierIntelEvent, error) {
	return s.event, nil
}

func openFinding() *carrierintel.CarrierIntelEvent {
	return &carrierintel.CarrierIntelEvent{
		ID:       pulid.MustNew("ciev_"),
		Status:   carrierintel.EventStatusOpen,
		Severity: carrierintel.SeverityHigh,
		Category: carrierintel.SectionAuthority,
	}
}

func TestResolveCarrierIntelEvent_SaysWhatCameOfIt(t *testing.T) {
	t.Parallel()

	event := openFinding()
	stub := &stubCarrierIntel{event: event}
	tool := newResolveCarrierIntelEventTool(stub)

	require.Equal(t, "resolve_carrier_intel_event", tool.Name())

	params := deskParams(map[string]any{
		"eventId":    event.ID.String(),
		"resolution": string(carrierintel.EventResolutionCarrierBlocked),
		"note":       "authority revoked; not to be tendered",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.resolved)
	require.Equal(t, carrierintel.EventResolutionCarrierBlocked, stub.resolved.Resolution)
	require.Empty(t, stub.acknowledged)
}

// Closing a finding has to name an outcome the domain knows. Anything else
// would record a decision nobody can read back.
func TestResolveCarrierIntelEvent_RefusesAnOutcomeItDoesNotHave(t *testing.T) {
	t.Parallel()

	event := openFinding()
	tool := newResolveCarrierIntelEventTool(&stubCarrierIntel{event: event})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(map[string]any{
		"eventId":    event.ID.String(),
		"resolution": "SeemedFine",
		"note":       "looked alright",
	}))
	require.ErrorContains(t, err, "not an outcome")
}

func TestAcknowledgeCarrierIntelEvent_TakesNoOutcome(t *testing.T) {
	t.Parallel()

	event := openFinding()
	stub := &stubCarrierIntel{event: event}
	tool := newAcknowledgeCarrierIntelEventTool(stub)

	require.Equal(t, "acknowledge_carrier_intel_event", tool.Name())
	require.NotContains(t, tool.ParamSchema()["required"], "resolution")

	params := deskParams(map[string]any{
		"eventId": event.ID.String(),
		"note":    "address change, nothing to do",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Equal(t, []pulid.ID{event.ID}, stub.acknowledged)
	require.Nil(t, stub.resolved)
}

func TestCarrierIntelEventTools_RefuseAFindingAlreadyClosed(t *testing.T) {
	t.Parallel()

	event := openFinding()
	event.Status = carrierintel.EventStatusResolved

	for name, tool := range map[string]serviceports.AgentTool{
		"acknowledge": newAcknowledgeCarrierIntelEventTool(&stubCarrierIntel{event: event}),
		"resolve":     newResolveCarrierIntelEventTool(&stubCarrierIntel{event: event}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tool.(serviceports.ToolValidator).Validate(
				t.Context(),
				deskParams(map[string]any{
					"eventId":    event.ID.String(),
					"resolution": string(carrierintel.EventResolutionCarrierUpdated),
					"note":       "already done",
				}),
			)
			require.ErrorContains(t, err, "already")
		})
	}
}

// ------------------------------------------------------- the repeat guard

type stubComments struct {
	serviceports.ShipmentCommentService
	items []*shipment.ShipmentComment
}

func (s *stubComments) ListByShipmentID(
	context.Context,
	*repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: s.items}, nil
}

func agentEmailComment(at int64) *shipment.ShipmentComment {
	return &shipment.ShipmentComment{
		CreatedAt: at,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "email_customer"},
	}
}

func TestAlreadyToldCustomer(t *testing.T) {
	t.Parallel()

	const now = int64(1_767_225_600)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("shp_")

	cases := []struct {
		name  string
		items []*shipment.ShipmentComment
		want  bool
	}{
		{
			name:  "nothing said yet",
			items: nil,
			want:  false,
		},
		{
			name:  "an email ten minutes ago",
			items: []*shipment.ShipmentComment{agentEmailComment(now - 600)},
			want:  true,
		},
		{
			name:  "an email just outside the window",
			items: []*shipment.ShipmentComment{agentEmailComment(now - 3601)},
			want:  false,
		},
		{
			name: "a dispatcher typing an update is not an email",
			items: []*shipment.ShipmentComment{{
				CreatedAt: now - 60,
				Type:      shipment.CommentTypeCustomerUpdate,
				Metadata:  map[string]any{"source": "user"},
			}},
			want: false,
		},
		{
			name: "another agent tool is not this one",
			items: []*shipment.ShipmentComment{{
				CreatedAt: now - 60,
				Type:      shipment.CommentTypeCustomerUpdate,
				Metadata:  map[string]any{"source": "agent", "tool": "add_shipment_comment"},
			}},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			told, err := alreadyToldCustomer(
				t.Context(), &stubComments{items: tc.items}, tenant, shipmentID, now,
			)
			require.NoError(t, err)
			require.Equal(t, tc.want, told)
		})
	}
}

// With no comment service wired the guard cannot read anything back, and a
// guard that cannot read must not silently refuse every send.
func TestAlreadyToldCustomer_SaysNothingWhenItCannotRead(t *testing.T) {
	t.Parallel()

	told, err := alreadyToldCustomer(
		t.Context(), nil,
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		pulid.MustNew("shp_"), 1_767_225_600,
	)
	require.NoError(t, err)
	require.False(t, told)
}
