// Package ratesimulationservice runs a pricing change against freight that
// already moved.
//
// The run is deliberately incremental: shipments are walked a page at a time
// and results are written as they are produced. A year of a mid-sized carrier's
// freight is tens of thousands of shipments, and holding them — or their
// results — in memory would put a simulation's cost on the same machine that is
// rating today's loads.
package ratesimulationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/core/temporaljobs/ratesimjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	simmath "github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// shipmentPageSize is how many shipments one page of the walk carries.
//
// Small enough that a failure loses little, large enough that the walk is not
// dominated by round trips.
const shipmentPageSize = 200

// traceSampleLimit bounds the traces kept for rule coverage.
//
// Coverage only needs to know which rules ever won and which ever lost, and a
// few thousand shipments settle that for every rule an agreement has. Keeping
// every trace from a hundred thousand shipments would hold hundreds of
// megabytes to learn nothing more.
const traceSampleLimit = 5_000

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.RateSimulationRepository
	AgreementRepo repositories.RateAgreementRepository
	ShipmentRepo  repositories.ShipmentRepository
	RateEngine    services.RateEngine
	AuditService  services.AuditService
	Workflows     services.WorkflowStarter
}

type Service struct {
	l             *zap.Logger
	repo          repositories.RateSimulationRepository
	agreementRepo repositories.RateAgreementRepository
	shipmentRepo  repositories.ShipmentRepository
	engine        services.RateEngine
	audit         services.AuditService
	workflows     services.WorkflowStarter
	now           func() int64
}

var _ services.RateSimulationRunner = (*Service)(nil)

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.ratesimulation"),
		repo:          p.Repo,
		agreementRepo: p.AgreementRepo,
		shipmentRepo:  p.ShipmentRepo,
		engine:        p.RateEngine,
		audit:         p.AuditService,
		workflows:     p.Workflows,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateSimulationByIDRequest,
) (*ratesimulation.RateSimulation, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateSimulationsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulation], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListResults(
	ctx context.Context,
	req *repositories.ListRateSimulationResultsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulationResult], error) {
	return s.repo.ListResults(ctx, req)
}

// Create records a simulation to be run.
//
// It is deliberately not run here. A year of shipments takes minutes, and a
// request that waits for it will time out long before the answer exists.
func (s *Service) Create(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
	userID pulid.ID,
) (*ratesimulation.RateSimulation, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	entity.Status = ratesimulation.StatusPending
	if !userID.IsNil() {
		entity.RequestedBy = &userID
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	return s.enqueue(ctx, created)
}

// enqueue hands the run to a worker.
//
// A simulation that cannot be queued is marked failed rather than left Pending
// forever. A row that says "waiting" when nothing is coming is the worst of the
// three outcomes: it looks like progress.
func (s *Service) enqueue(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulation.RateSimulation, error) {
	if s.workflows == nil {
		return entity, nil
	}

	workflowID := "rate-simulation/" + entity.ID.String()

	if _, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowID,
			TaskQueue:             temporaltype.IntegrationTaskQueue,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		ratesimjobs.RunRateSimulationWorkflow,
		&ratesimjobs.RunSimulationPayload{
			OrgID:            entity.OrganizationID,
			BuID:             entity.BusinessUnitID,
			UserID:           idOrZero(entity.RequestedBy),
			RateSimulationID: entity.ID,
		},
	); err != nil {
		s.l.Error("failed to queue a rate simulation", zap.Error(err))

		entity.Status = ratesimulation.StatusFailed
		entity.Error = "The simulation could not be queued to run"

		if _, uErr := s.repo.Update(ctx, entity); uErr != nil {
			s.l.Error("failed to mark an unqueued simulation as failed", zap.Error(uErr))
		}

		return nil, err
	}

	entity.WorkflowID = workflowID

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		// The run is already queued and will do its work. Losing the workflow
		// id only costs the ability to cancel it from here.
		s.l.Warn("failed to record a simulation's workflow id", zap.Error(err))

		return entity, nil
	}

	return updated, nil
}

func idOrZero(id *pulid.ID) pulid.ID {
	if id == nil {
		return pulid.Nil
	}

	return *id
}

// Run replays the simulation's agreement against its window of shipments.
//
// Nothing it produces can reach a shipment: every rating is a simulation
// purpose quote and none are persisted. A failure part way through leaves the
// run marked failed with whatever it managed to price, which is more useful
// than an empty row that says only that something went wrong.
func (s *Service) Run(ctx context.Context, req *services.RunRateSimulationRequest) error {
	entity, err := s.repo.GetByID(ctx, &repositories.GetRateSimulationByIDRequest{
		RateSimulationID: req.RateSimulationID,
		TenantInfo:       req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if entity.Status.IsTerminal() {
		return nil
	}

	if err = s.markRunning(ctx, entity); err != nil {
		return err
	}

	summary, coverage, runErr := s.walk(ctx, req, entity)

	return s.finish(ctx, entity, &summary, coverage, runErr)
}

func (s *Service) markRunning(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) error {
	started := s.now()
	entity.Status = ratesimulation.StatusRunning
	entity.StartedAt = &started

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return err
	}

	*entity = *updated

	return nil
}

// walk replays every shipment in the window, a page at a time.
func (s *Service) walk(
	ctx context.Context,
	req *services.RunRateSimulationRequest,
	entity *ratesimulation.RateSimulation,
) (simmath.Summary, []*ratesimulation.RuleCoverage, error) {
	rules, err := s.simulatedRules(ctx, entity)
	if err != nil {
		return simmath.Summary{}, nil, err
	}

	// Every shipment in the run reads the same tenant's rate tables. Without
	// this the walk would re-read them tens of thousands of times.
	ctx = ratetablecache.With(ctx)
	ctx = effectiveversioncache.With(ctx)

	var (
		acc    simmath.Accumulator
		traces = make([]*ratetypes.Trace, 0, min(traceSampleLimit, shipmentPageSize))
		after  pulid.ID
		done   int
	)

	for {
		page, pageErr := s.repo.ListShipments(ctx, &repositories.ListSimulationShipmentsRequest{
			TenantInfo: req.TenantInfo,
			PartyType:  string(entity.PartyType),
			From:       entity.SampleFrom,
			To:         entity.SampleTo,
			AfterID:    after,
			Limit:      s.pageSize(entity, done),
		})
		if pageErr != nil {
			return acc.Summary(), buildCoverage(rules, traces), pageErr
		}

		if len(page.ShipmentIDs) == 0 {
			break
		}

		results, pageTraces := s.replayPage(ctx, req.TenantInfo, entity, page.ShipmentIDs, &acc)

		if err = s.repo.AppendResults(ctx, results); err != nil {
			return acc.Summary(), buildCoverage(rules, traces), err
		}

		if len(traces) < traceSampleLimit {
			traces = append(traces, pageTraces...)
		}

		done += len(page.ShipmentIDs)
		if req.Heartbeat != nil {
			req.Heartbeat(done)
		}

		if page.NextAfterID.IsNil() || s.reachedLimit(entity, done) {
			break
		}

		after = page.NextAfterID
	}

	return acc.Summary(), buildCoverage(rules, traces), nil
}

// pageSize keeps the walk from overshooting a requested sample limit.
func (s *Service) pageSize(entity *ratesimulation.RateSimulation, done int) int {
	if entity.SampleLimit <= 0 {
		return shipmentPageSize
	}

	remaining := entity.SampleLimit - done
	if remaining < shipmentPageSize {
		return remaining
	}

	return shipmentPageSize
}

func (s *Service) reachedLimit(entity *ratesimulation.RateSimulation, done int) bool {
	return entity.SampleLimit > 0 && done >= entity.SampleLimit
}

// replayPage rates one page of shipments against the simulated agreement.
//
// One shipment that will not rate does not stop the page: it becomes a row
// carrying the reason, which is exactly the row somebody needs to see.
func (s *Service) replayPage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *ratesimulation.RateSimulation,
	shipmentIDs []pulid.ID,
	acc *simmath.Accumulator,
) ([]*ratesimulation.RateSimulationResult, []*ratetypes.Trace) {
	results := make([]*ratesimulation.RateSimulationResult, 0, len(shipmentIDs))
	traces := make([]*ratetypes.Trace, 0, len(shipmentIDs))

	for _, shipmentID := range shipmentIDs {
		historical, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         shipmentID,
			TenantInfo: tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			s.l.Warn("failed to load a shipment for simulation",
				zap.String("shipmentId", shipmentID.String()),
				zap.Error(err),
			)

			continue
		}

		rated, rateErr := s.engine.RateShipment(ctx, &services.RateShipmentRequest{
			Shipment:   historical,
			TenantInfo: tenantInfo,
			PartyType:  entity.PartyType,
			PartyID:    partyOf(historical, entity.PartyType),
			// Left unset: the shipment's own date decides which terms apply,
			// which is what makes this a replay rather than a re-pricing.
			AsOf:                0,
			Purpose:             ratequote.PurposeSimulation,
			Persist:             false,
			SimulateAgreementID: &entity.RateAgreementID,
			UserID:              tenantInfo.UserID,
		})

		result := replayResult(historical, rated, rateErr)
		result.OrganizationID = entity.OrganizationID
		result.BusinessUnitID = entity.BusinessUnitID
		result.RateSimulationID = entity.ID

		results = append(results, result)
		acc.Add(simmath.Delta{
			Before: result.BeforeAmount,
			After:  result.AfterAmount,
			Failed: result.Failed(),
		})

		if rated != nil && rated.Quote != nil && rated.Quote.Trace != nil {
			traces = append(traces, rated.Quote.Trace)
		}
	}

	return results, traces
}

// partyOf is who the shipment is priced against on the side being simulated.
//
// The sell side takes it from the shipment's customer, which the engine does
// for itself, so nothing is supplied.
func partyOf(historical *shipment.Shipment, partyType rateagreement.PartyType) pulid.ID {
	if partyType != rateagreement.PartyTypeCarrier {
		return pulid.Nil
	}

	for _, move := range historical.Moves {
		if move == nil || move.CarrierAssignment == nil {
			continue
		}

		return move.CarrierAssignment.CarrierID
	}

	return pulid.Nil
}

// simulatedRules reads the agreement's own rules, which is what rule coverage
// is reported against. Nothing else in the run needs them.
func (s *Service) simulatedRules(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) ([]*rateagreement.RateAgreementRule, error) {
	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: entity.RateAgreementID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	return agreement.Rules, nil
}

func (s *Service) finish(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
	summary *simmath.Summary,
	coverage []*ratesimulation.RuleCoverage,
	runErr error,
) error {
	completed := s.now()

	entity.Summary = summary
	entity.RuleCoverage = coverage
	entity.CompletedAt = &completed

	if runErr != nil {
		entity.Status = ratesimulation.StatusFailed
		entity.Error = runErr.Error()
	} else {
		entity.Status = ratesimulation.StatusCompleted
	}

	if _, err := s.repo.Update(ctx, entity); err != nil {
		return err
	}

	return runErr
}
