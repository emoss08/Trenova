package findingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelreviewservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
)

type stubEscalator struct {
	occurrence *detention.DetentionOccurrence
	escalated  *detentionreview.EscalateParams
	guard      agenttoolstest.WriteGuard
}

func (s *stubEscalator) Escalate(
	_ context.Context,
	p detentionreview.EscalateParams,
) (*detention.DetentionOccurrence, error) {
	if err := s.guard.Write(); err != nil {
		return nil, err
	}
	s.escalated = &p
	if err := s.occurrence.Escalate(timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return s.occurrence, nil
}

func (s *stubEscalator) GetOccurrenceDetail(
	context.Context,
	*detentionports.GetDetentionOccurrenceByIDRequest,
) (*detentionreview.OccurrenceDetail, error) {
	copied := *s.occurrence

	return &detentionreview.OccurrenceDetail{Occurrence: &copied}, nil
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
	tool := NewEscalateDetentionTool(stub)

	require.Equal(t, permission.ResourceDetention, tool.Policy().Resource)
	require.Equal(t, permission.OpUpdate, tool.Policy().Operation)
	require.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
	require.True(t, tool.Policy().Reversible)

	params := agenttoolstest.DeskParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"reason":       "the notice window closed and the customer has no recipients",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Contains(t, preview.Summary, "90 billable minutes")
	for _, change := range preview.Changes {
		for _, field := range change.Fields {
			require.NotContains(t, []string{"status", "billableAmount", "billableMinutes"},
				field.Path, "escalating must not touch the charge")
		}
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
	tool := NewEscalateDetentionTool(stub)

	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"occurrenceId": occurrenceID.String(),
			"reason":       "please look at this",
		}),
	)
	require.ErrorContains(t, err, "waived")
}

// One ask covering every paper, keyed so a retried proposal does not tell
// the driver twice.
func TestRequestCredentialRenewal_AsksOncePerDriver(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	first, second := pulid.MustNew("wcred_"), pulid.MustNew("wcred_")
	stub := &agenttoolstest.StubCredentialActor{}
	tool := NewRequestCredentialRenewalTool(stub)

	require.True(t, tool.Policy().Idempotent)
	require.Equal(t, permission.ResourceWorkerCredential, tool.Policy().Resource)

	params := agenttoolstest.DeskParams(map[string]any{
		"workerId":      workerID.String(),
		"credentialIds": []any{first.String(), second.String()},
		"note":          "Your CDL and medical card are both due this month.",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.Asked)
	require.Equal(t, workerID, stub.Asked.WorkerID)
	require.Equal(t, []pulid.ID{first, second}, stub.Asked.CredentialIDs)
	require.Equal(t, "key-1", stub.Asked.CorrelationID)
}

func TestRequestCredentialRenewal_RefusesAnEmptyAsk(t *testing.T) {
	t.Parallel()

	tool := NewRequestCredentialRenewalTool(&agenttoolstest.StubCredentialActor{})
	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"workerId":      pulid.MustNew("wrk_").String(),
			"credentialIds": []any{},
			"note":          "please renew",
		}),
	)
	require.ErrorContains(t, err, "at least one credential")
}

type stubWorkerHolder struct {
	held  *worker.Worker
	given *workerservice.DispatchHoldRequest
	guard agenttoolstest.WriteGuard
}

func (s *stubWorkerHolder) SetDispatchHold(
	_ context.Context,
	req workerservice.DispatchHoldRequest,
) (*worker.Worker, error) {
	if err := s.guard.Write(); err != nil {
		return nil, err
	}
	s.given = &req
	if err := workerservice.ApplyDispatchHold(s.held, &req); err != nil {
		return nil, err
	}

	return s.held, nil
}

func (s *stubWorkerHolder) Get(
	context.Context,
	workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	copied := *s.held

	return &copied, nil
}

// The hold is its own permission, not an edit of the worker record: taking
// somebody off the board until a paper is renewed is a dispatch decision,
// and an agent may never edit employment.
func TestPlaceWorkerDispatchHold_IsItsOwnPermission(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	stub := &stubWorkerHolder{held: &worker.Worker{ID: workerID, CanBeAssigned: true}}
	tool := NewPlaceWorkerDispatchHoldTool(stub)

	require.Equal(t, permission.ResourceWorkerDispatchHold, tool.Policy().Resource)
	require.Equal(t, permission.OpCreate, tool.Policy().Operation)
	require.True(t, tool.Policy().Reversible)
	require.False(t, agentceiling.Allows(permission.ResourceWorker, permission.OpUpdate),
		"editing a worker record stays closed to an agent")

	params := agenttoolstest.DeskParams(map[string]any{
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
	tool := NewPlaceWorkerDispatchHoldTool(stub)

	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"workerId": workerID.String(),
			"reason":   "medical card lapsed",
		}),
	)
	require.ErrorContains(t, err, "already off dispatch")
}

type stubCarrierIntel struct {
	event        *carrierintelevent.CarrierIntelEvent
	acknowledged []pulid.ID
	resolved     *carrierintelreviewservice.ResolveEventRequest
	guard        agenttoolstest.WriteGuard
}

func (s *stubCarrierIntel) AcknowledgeEvents(
	_ context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (int, error) {
	if err := s.guard.Write(); err != nil {
		return 0, err
	}
	s.acknowledged = ids
	if !s.event.Acknowledge(tenant.UserID, timeutils.NowUnix()) {
		return 0, nil
	}

	return len(ids), nil
}

func (s *stubCarrierIntel) ResolveEvent(
	_ context.Context,
	req *carrierintelreviewservice.ResolveEventRequest,
) (*carrierintelevent.CarrierIntelEvent, error) {
	if err := s.guard.Write(); err != nil {
		return nil, err
	}
	s.resolved = req
	if err := carrierintelreviewservice.PlanResolve(s.event, req, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return s.event, nil
}

func (s *stubCarrierIntel) GetEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrierintelevent.CarrierIntelEvent, error) {
	copied := *s.event

	return &copied, nil
}

func openFinding() *carrierintelevent.CarrierIntelEvent {
	return &carrierintelevent.CarrierIntelEvent{
		ID:       pulid.MustNew("ciev_"),
		Status:   carrierintelevent.EventStatusOpen,
		Severity: carrierintel.SeverityHigh,
		Category: carrierintel.SectionAuthority,
	}
}

func TestResolveCarrierIntelEvent_SaysWhatCameOfIt(t *testing.T) {
	t.Parallel()

	event := openFinding()
	stub := &stubCarrierIntel{event: event}
	tool := NewResolveCarrierIntelEventTool(stub)

	require.Equal(t, "carrier_resolve_intelligence_event", tool.Name())

	params := agenttoolstest.DeskParams(map[string]any{
		"eventId":    event.ID.String(),
		"resolution": string(carrierintelevent.EventResolutionCarrierBlocked),
		"note":       "authority revoked; not to be tendered",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.resolved)
	require.Equal(t, carrierintelevent.EventResolutionCarrierBlocked, stub.resolved.Resolution)
	require.Empty(t, stub.acknowledged)
}

// Closing a finding has to name an outcome the domain knows. Anything else
// would record a decision nobody can read back.
func TestResolveCarrierIntelEvent_RefusesAnOutcomeItDoesNotHave(t *testing.T) {
	t.Parallel()

	event := openFinding()
	tool := NewResolveCarrierIntelEventTool(&stubCarrierIntel{event: event})

	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"eventId":    event.ID.String(),
			"resolution": "SeemedFine",
			"note":       "looked alright",
		}),
	)
	require.ErrorContains(t, err, "not an outcome")
}

func TestAcknowledgeCarrierIntelEvent_TakesNoOutcome(t *testing.T) {
	t.Parallel()

	event := openFinding()
	stub := &stubCarrierIntel{event: event}
	tool := NewAcknowledgeCarrierIntelEventTool(stub)

	require.Equal(t, "carrier_acknowledge_intelligence_event", tool.Name())
	require.NotContains(t, tool.ParamSchema()["required"], "resolution")

	params := agenttoolstest.DeskParams(map[string]any{
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
	event.Status = carrierintelevent.EventStatusResolved

	for name, tool := range map[string]agenttoolports.AgentTool{
		"acknowledge": NewAcknowledgeCarrierIntelEventTool(&stubCarrierIntel{event: event}),
		"resolve":     NewResolveCarrierIntelEventTool(&stubCarrierIntel{event: event}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tool.(agenttoolports.ToolValidator).Validate(
				t.Context(),
				agenttoolstest.DeskParams(map[string]any{
					"eventId":    event.ID.String(),
					"resolution": string(carrierintelevent.EventResolutionCarrierUpdated),
					"note":       "already done",
				}),
			)
			require.ErrorContains(t, err, "already")
		})
	}
}
