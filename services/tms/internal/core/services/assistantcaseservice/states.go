package assistantcaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type StatesParams struct {
	fx.In

	Logger *zap.Logger
	Cases  repositories.AssistantCaseRepository
	Waits  repositories.AgentWaitRepository
	Etas   serviceports.ShipmentEtaReader `optional:"true"`
}

// States works out where cases stand. It holds nothing it could not read
// again: a case's state is the record's, the open waits' and the snooze's,
// read together each time.
type States struct {
	cases repositories.AssistantCaseRepository
	waits repositories.AgentWaitRepository
	etas  serviceports.ShipmentEtaReader
	now   func() int64
	l     *zap.Logger
}

func NewStates(p StatesParams) *States {
	return &States{
		cases: p.Cases,
		waits: p.Waits,
		etas:  p.Etas,
		now:   timeutils.NowUnix,
		l:     p.Logger.Named("service.assistant-case-states"),
	}
}

func ProvideStates(s *States) serviceports.AssistantCaseStates { return s }

// Attach sets Case on every conversation in the page that is a case: one
// query per kind of record, one for the open waits, and one for the
// estimates only when a case is snoozed to its ETA.
func (s *States) Attach(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	threads []*conversation.Thread,
) error {
	cases := make([]*conversation.Thread, 0, len(threads))
	for _, thread := range threads {
		if thread != nil && thread.IsCase() {
			cases = append(cases, thread)
		}
	}
	if len(cases) == 0 {
		return nil
	}

	refs := make([]deskcase.Ref, 0, len(cases))
	threadIDs := make([]pulid.ID, 0, len(cases))
	for _, thread := range cases {
		refs = append(refs, thread.CaseRef())
		threadIDs = append(threadIDs, thread.ID)
	}

	records, err := s.cases.ListRecords(ctx, &repositories.ListCaseRecordsRequest{
		TenantInfo: tenant,
		Refs:       refs,
	})
	if err != nil {
		return err
	}
	waits, err := s.waits.ListOpenByThreads(ctx, &repositories.ListOpenWaitsByThreadsRequest{
		TenantInfo: tenant,
		UserID:     userID,
		ThreadIDs:  threadIDs,
	})
	if err != nil {
		return err
	}
	etas := s.snoozedETAs(ctx, tenant, cases)

	now := s.now()
	for _, thread := range cases {
		record := records[thread.SubjectID]
		if record == nil {
			record = deskcase.Missing(thread.CaseRef())
		}
		thread.Case = deskcase.Resolve(&deskcase.Inputs{
			Record: record,
			Waits:  openWaits(waits[thread.ID]),
			Snooze: thread.Snooze(),
			ETA:    etas[thread.SubjectID],
			Now:    now,
		})
	}

	return nil
}

// summarize works out one case, for the case view and for the answers to
// the writes that change it.
func (s *States) summarize(
	ctx context.Context,
	tenant pagination.TenantInfo,
	thread *conversation.Thread,
) (*deskcase.Summary, error) {
	if err := s.Attach(ctx, tenant, thread.UserID, []*conversation.Thread{thread}); err != nil {
		return nil, err
	}

	return thread.Case, nil
}

// snoozedETAs reads the estimate for every shipment a case is snoozed to
// the ETA of. An estimate that cannot be read leaves the snooze at the time
// it was set to rather than failing the rail.
func (s *States) snoozedETAs(
	ctx context.Context,
	tenant pagination.TenantInfo,
	threads []*conversation.Thread,
) map[pulid.ID]*int64 {
	if s.etas == nil {
		return nil
	}

	shipmentIDs := make([]pulid.ID, 0)
	for _, thread := range threads {
		if thread.SnoozeAnchor == deskcase.SnoozeETA &&
			thread.SubjectType == agent.SubjectShipment {
			shipmentIDs = append(shipmentIDs, thread.SubjectID)
		}
	}
	if len(shipmentIDs) == 0 {
		return nil
	}

	estimates, err := s.etas.EtasByShipmentIDs(ctx, tenant, shipmentIDs)
	if err != nil {
		s.l.Warn("could not read the ETAs snoozed cases follow", zap.Error(err))
		return nil
	}

	out := make(map[pulid.ID]*int64, len(estimates))
	for id, estimate := range estimates {
		if estimate != nil && estimate.EstimatedArrival != nil {
			out[id] = estimate.EstimatedArrival
		}
	}

	return out
}

// openWaits says who each open wait is waiting on: a reply from the
// carrier or the customer, a reply about the record, or anything else.
func openWaits(waits []*agentwait.Wait) []deskcase.OpenWait {
	out := make([]deskcase.OpenWait, 0, len(waits))
	for _, wait := range waits {
		out = append(out, deskcase.OpenWait{Party: partyOf(wait), DueAt: wait.DueAt})
	}

	return out
}

func partyOf(wait *agentwait.Wait) deskcase.WaitingOn {
	if wait.Kind != agentwait.KindReply || wait.Condition == nil {
		return deskcase.WaitingOnEvent
	}

	switch {
	case wait.Condition.CarrierID.IsNotNil():
		return deskcase.WaitingOnCarrier
	case wait.Condition.CustomerID.IsNotNil():
		return deskcase.WaitingOnCustomer
	default:
		return deskcase.WaitingOnReply
	}
}
