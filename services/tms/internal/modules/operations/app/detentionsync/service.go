package detentionsync

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l                 *zap.Logger
	policyRepo        detentionports.DetentionPolicyRepository
	occurrenceRepo    detentionports.DetentionOccurrenceRepository
	evidenceRepo      detentionports.DetentionEvidenceRepository
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	agreementRepo     rateagreementports.RateAgreementRepository
	payAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	commentRepo       shipmentcommentports.ShipmentCommentRepository
	watchtower        watchtowerports.WatchtowerProjector
	publisher         agenteventports.AgentEventPublisher
	now               func() int64
	effects           *detentioneffects.Effects
}

// recordChargeComment narrates a detention charge onto the shipment thread.
//
// The charge lands on the invoice as one accessorial line with no explanation
// attached, and the person who has to defend it is rarely the person who was
// watching the clock. Writing the derivation into the shipment's comments puts
// the reason next to the money, where anyone reopening the shipment will find
// it.
func (s *Service) recordChargeComment(ctx context.Context, p chargeCommentParams) {
	if p.occurrence == nil || p.occurrence.ShipmentID.IsNil() {
		return
	}

	kind, ok := chargeCommentKindFor(p.occurrence, p.previous)
	if !ok {
		return
	}

	entity := &shipment.ShipmentComment{
		OrganizationID: p.occurrence.OrganizationID,
		BusinessUnitID: p.occurrence.BusinessUnitID,
		ShipmentID:     p.occurrence.ShipmentID,
		Comment:        buildChargeComment(kind, p),
		Type:           shipmentvocab.CommentTypeBilling,
		Visibility:     shipmentvocab.CommentVisibilityInternal,
		Priority:       chargeCommentPriority(kind, p.occurrence),
		Source:         shipmentvocab.CommentSourceSystem,
		Metadata: map[string]any{
			"event":                 "DetentionCharge" + string(kind),
			"detentionOccurrenceId": p.occurrence.ID.String(),
			"stopId":                p.occurrence.StopID.String(),
			"amount":                p.occurrence.BillableAmount.StringFixed(2),
			"currency":              p.occurrence.Currency,
		},
	}

	if _, err := s.commentRepo.Create(ctx, entity); err != nil {
		s.l.Warn("failed to comment detention charge on shipment",
			zap.String("occurrenceId", p.occurrence.ID.String()),
			zap.String("shipmentId", p.occurrence.ShipmentID.String()),
			zap.Error(err))
	}
}

// resolveContractPrices reads the accessorial schedule of the contract that
// priced this shipment.
//
// This is what makes the rate confirmation and the invoice agree on detention:
// both read the price from the contract rather than one reading the
// organization default and the other whatever was negotiated.
//
// A contract that cannot be read is logged and treated as absent. Detention at
// the organization's default rate is a smaller problem than a shipment whose
// stops cannot be evaluated at all.
func (s *Service) resolveContractPrices(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	ratedAt int64,
) *contractPrices {
	if entity.RateAgreementID == nil || entity.RateAgreementID.IsNil() {
		return nil
	}

	agreement, err := s.agreementRepo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: *entity.RateAgreementID,
		TenantInfo:      tenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		s.l.Warn("failed to load rate agreement for detention pricing",
			zap.String("shipmentId", entity.ID.String()),
			zap.String("agreementId", entity.RateAgreementID.String()),
			zap.Error(err),
		)

		return nil
	}

	prices := &contractPrices{
		byAccessorial: make(
			map[pulid.ID]*rateagreement.RateAgreementAccessorial,
			len(agreement.Accessorials),
		),
	}

	for _, row := range agreement.Accessorials {
		if row == nil || !row.IsEffectiveAt(ratedAt) {
			continue
		}

		prices.byAccessorial[row.AccessorialChargeID] = row
	}

	if len(prices.byAccessorial) == 0 {
		return nil
	}

	return prices
}

type Params struct {
	fx.In

	Logger            *zap.Logger
	PolicyRepo        detentionports.DetentionPolicyRepository
	OccurrenceRepo    detentionports.DetentionOccurrenceRepository
	EvidenceRepo      detentionports.DetentionEvidenceRepository
	AccessorialRepo   accessorialchargeports.AccessorialChargeRepository
	AgreementRepo     rateagreementports.RateAgreementRepository
	PayAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	CommentRepo       shipmentcommentports.ShipmentCommentRepository
	Watchtower        watchtowerports.WatchtowerProjector
	Publisher         agenteventports.AgentEventPublisher
	Now               timeutils.Clock
	Effects           *detentioneffects.Effects
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	now := p.Now
	return &Service{
		l:                 p.Logger.Named("service.detention"),
		policyRepo:        p.PolicyRepo,
		occurrenceRepo:    p.OccurrenceRepo,
		evidenceRepo:      p.EvidenceRepo,
		accessorialRepo:   p.AccessorialRepo,
		agreementRepo:     p.AgreementRepo,
		payAssignmentRepo: p.PayAssignmentRepo,
		commentRepo:       p.CommentRepo,
		watchtower:        p.Watchtower,
		publisher:         p.Publisher,
		now:               now,
		effects:           p.Effects,
	}
}

// SyncShipment recomputes every stop's detention for a shipment.
// defaultPolicyID is the organization's default policy from shipment control,
// the one that governs a stop no other policy covers.
//
// Frozen occurrences are never rewritten: once a charge has been billed,
// waived, or disputed the customer has seen a number, and silently changing it
// is how carriers lose credibility in a dispute.
func (s *Service) SyncShipment(
	ctx context.Context,
	entity *shipment.Shipment,
	defaultPolicyID pulid.ID,
) (*SyncShipmentResult, error) {
	if entity == nil {
		return &SyncShipmentResult{TotalAmount: decimal.Zero}, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	location, err := s.effects.TenantLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	log := s.l.With(
		zap.String("operation", "SyncShipment"),
		zap.String("shipmentId", entity.ID.String()),
	)

	result := &SyncShipmentResult{
		Occurrences: make([]*detention.DetentionOccurrence, 0, 4),
		TotalAmount: decimal.Zero,
	}

	now := s.now()

	// The contract's own accessorial prices, read once for the whole shipment.
	prices := s.resolveContractPrices(ctx, entity, tenantInfo, now)

	plan, err := s.planShipment(ctx, entity, tenantInfo, defaultPolicyID, now)
	if err != nil {
		return nil, err
	}

	accessorials, err := s.loadAccessorials(ctx, plan, tenantInfo)
	if err != nil {
		return nil, err
	}

	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero

	for _, sp := range plan {
		if sp.policy == nil {
			result.Skipped++
			result.Occurrences = append(result.Occurrences, sp.existing)
			shipmentAccrued = shipmentAccrued.Add(sp.existing.BillableAmount)
			result.TotalAmount = result.TotalAmount.Add(sp.existing.BillableAmount)
			continue
		}

		occurrence, computeErr := s.computeStop(ctx, &computeStopParams{
			shipment:        entity,
			move:            sp.move,
			stop:            sp.stop,
			existing:        sp.existing,
			policy:          sp.policy,
			accessorial:     accessorials[sp.policy.AccessorialChargeID],
			tenantInfo:      tenantInfo,
			location:        location,
			payRate:         sp.payRate,
			dayAccrued:      dayAccrued,
			shipmentAccrued: shipmentAccrued,
			prices:          prices,
			now:             now,
		})
		if errors.Is(computeErr, errStopNotCharged) {
			continue
		}
		if computeErr != nil {
			log.Error("failed to compute detention for stop",
				zap.String("stopId", sp.stop.ID.String()), zap.Error(computeErr))
			return nil, computeErr
		}

		result.Occurrences = append(result.Occurrences, occurrence)
		shipmentAccrued = shipmentAccrued.Add(occurrence.BillableAmount)
		result.TotalAmount = result.TotalAmount.Add(occurrence.BillableAmount)
	}

	return result, nil
}

// planShipment decides, before anything is computed, what each billable stop
// needs: its stored occurrence and, unless that is frozen, the policy that
// governs it. The occurrences, the candidate policies and the drivers' pay
// are each one read for the whole shipment, so recalculating costs the same
// reads however many stops and moves it has.
func (s *Service) planShipment(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	defaultPolicyID pulid.ID,
	now int64,
) ([]*stopPlan, error) {
	existing, err := s.loadOccurrences(ctx, entity, tenantInfo)
	if err != nil {
		return nil, err
	}

	candidates, err := s.loadResolutionCandidates(ctx, entity, tenantInfo)
	if err != nil {
		return nil, err
	}

	payRates := s.loadDriverPayRates(ctx, entity, tenantInfo, now)
	commodityIDs := shipmentCommodityIDs(entity)
	plan := make([]*stopPlan, 0, len(existing)+4)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		payRate := payRates[movePrimaryWorker(move)]

		for _, stop := range move.Stops {
			if !stopIsBillable(stop) {
				continue
			}

			sp := &stopPlan{move: move, stop: stop, existing: existing[stop.ID], payRate: payRate}
			if sp.existing == nil || !sp.existing.IsFrozen() {
				sp.policy = resolveStopPolicy(&resolveStopParams{
					shipment:        entity,
					stop:            stop,
					commodityIDs:    commodityIDs,
					candidates:      candidates,
					defaultPolicyID: defaultPolicyID,
				})
				if sp.policy == nil {
					continue
				}
			}

			plan = append(plan, sp)
		}
	}

	return plan, nil
}

var errStopNotCharged = errors.New("no detention policy charges this stop")

// loadResolutionCandidates reads the policies that could govern any billable
// stop on the shipment in one query, so recalculating a shipment costs one
// policy read however many stops it has. Each stop then picks its own from
// this set in memory.
func (s *Service) loadResolutionCandidates(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) ([]*detentionpolicy.DetentionPolicy, error) {
	req, ok := resolutionCandidatesRequest(entity, tenantInfo)
	if !ok {
		return nil, nil
	}

	return s.policyRepo.GetResolutionCandidates(ctx, req)
}

// loadOccurrences reads the stored occurrence of every billable stop on the
// shipment in one query, keyed by stop.
func (s *Service) loadOccurrences(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]*detention.DetentionOccurrence, error) {
	stopIDs := billableStopIDs(entity)
	if len(stopIDs) == 0 {
		return map[pulid.ID]*detention.DetentionOccurrence{}, nil
	}

	occurrences, err := s.occurrenceRepo.ListByStops(
		ctx,
		&detentionports.ListOccurrencesByStopsRequest{
			StopIDs:    stopIDs,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	byStop := make(map[pulid.ID]*detention.DetentionOccurrence, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence != nil {
			byStop[occurrence.StopID] = occurrence
		}
	}

	return byStop, nil
}

// loadAccessorials reads the accessorial charge of every policy the plan
// resolved, in one query. A policy whose charge cannot be found fails the
// recalculation, as reading it alone did.
func (s *Service) loadAccessorials(
	ctx context.Context,
	plan []*stopPlan,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]*accessorialcharge.AccessorialCharge, error) {
	ids := planAccessorialIDs(plan)
	if len(ids) == 0 {
		return map[pulid.ID]*accessorialcharge.AccessorialCharge{}, nil
	}

	charges, err := s.accessorialRepo.GetByIDs(
		ctx,
		accessorialchargeports.GetAccessorialChargesByIDsRequest{
			TenantInfo:           tenantInfo,
			AccessorialChargeIDs: ids,
		},
	)
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*accessorialcharge.AccessorialCharge, len(charges))
	for _, charge := range charges {
		if charge != nil {
			byID[charge.ID] = charge
		}
	}

	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			return nil, errortypes.NewNotFoundError(
				"The detention policy's accessorial charge was not found within your organization",
			)
		}
	}

	return byID, nil
}

func (s *Service) computeStop(
	ctx context.Context,
	p *computeStopParams,
) (*detention.DetentionOccurrence, error) {
	snapshot := snapshotPolicy(p)

	var noticeSentAt *int64
	if p.existing != nil {
		noticeSentAt = p.existing.NoticeSentAt
	}

	computed := detentionengine.Compute(&detentionengine.ComputeInput{
		Snapshot:         snapshot,
		StopType:         p.stop.Type,
		ScheduleType:     p.stop.ScheduleType,
		ArrivedAt:        p.stop.ActualArrival,
		DepartedAt:       p.stop.ActualDeparture,
		AppointmentStart: appointmentStart(p.stop),
		AppointmentEnd:   p.stop.ScheduledWindowEnd,
		NoticeSentAt:     noticeSentAt,
		Now:              p.now,
		Location:         p.location,
		DriverPayRate:    p.payRate,
		DayAccrued:       p.dayAccrued,
		ShipmentAccrued:  p.shipmentAccrued,
	})

	if !computed.Applicable {
		return nil, errStopNotCharged
	}

	for day, amount := range computed.DayAllocation {
		p.dayAccrued[day] = p.dayAccrued[day].Add(amount)
	}

	saved, err := s.occurrenceRepo.Upsert(ctx, buildOccurrence(p, &snapshot, &computed))
	if err != nil {
		return nil, err
	}

	s.recordRecalculation(ctx, saved, p.existing)
	s.effects.PublishBillingHoldChange(ctx, p.existing, saved, pulid.Nil)
	s.recordChargeComment(ctx, chargeCommentParams{
		occurrence: saved,
		previous:   p.existing,
		stop:       p.stop,
	})
	s.projectToWatchtower(ctx, saved, p)
	s.publishOccurrenceOpened(ctx, saved, p)

	return saved, nil
}

// snapshotPolicy freezes the governing policy at the contract's price, which
// wins over the organization default so the rate confirmation and the invoice
// agree on detention.
func snapshotPolicy(p *computeStopParams) detentionpolicy.PolicySnapshot {
	flatRate := p.accessorial.Amount
	rateUnit := p.accessorial.RateUnit

	if contractRate, contractUnit, priced := p.prices.priceFor(p.accessorial); priced {
		flatRate = contractRate
		rateUnit = contractUnit
	}

	return detentionpolicy.NewPolicySnapshot(&detentionpolicy.SnapshotInput{
		Policy:          p.policy,
		StopType:        string(p.stop.Type),
		FreeMinutes:     p.policy.FreeMinutesForStopType(p.stop.Type),
		AccessorialCode: p.accessorial.Code,
		FlatRate:        flatRate,
		FlatRateUnit:    detentionengine.TierUnitFromAccessorial(rateUnit),
		CapturedAt:      p.now,
	})
}

// recordRecalculation appends an evidence link whenever a stored amount moves,
// so the reason a number changed is never lost.
func (s *Service) recordRecalculation(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	existing *detention.DetentionOccurrence,
) {
	if saved == nil {
		return
	}

	if existing != nil && existing.BillableAmount.Equal(saved.BillableAmount) &&
		existing.Status == saved.Status {
		return
	}

	summary := "Detention recalculated to " + saved.BillableAmount.StringFixed(2) +
		" " + saved.Currency + " (" + string(saved.Status) + ")"

	entry := &detentionevidence.DetentionEvidence{
		OrganizationID:        saved.OrganizationID,
		BusinessUnitID:        saved.BusinessUnitID,
		DetentionOccurrenceID: saved.ID,
		Kind:                  detentionevidence.EvidenceKindRecalculated,
		Source:                detentionevidence.EvidenceSourceSystem,
		Summary:               summary,
		ObservedAt:            s.now(),
		RecordedAt:            s.now(),
	}

	if _, err := s.evidenceRepo.Append(ctx, &detentionports.AppendEvidenceRequest{
		Entry: entry,
	}); err != nil {
		s.l.Warn("failed to append detention recalculation evidence",
			zap.String("occurrenceId", saved.ID.String()), zap.Error(err))
	}
}

// loadDriverPayRates finds the detention rate each assigned driver is paid,
// so the AR and AP sides of the same dwell can be compared on one record. The
// assignments of every primary driver on a move with a billable stop are one
// read for the whole shipment, keyed by worker. A read that fails is logged
// and leaves the rates out: the customer's detention does not wait on the
// driver's pay.
func (s *Service) loadDriverPayRates(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	asOf int64,
) map[pulid.ID]decimal.NullDecimal {
	workerIDs := billableMoveWorkers(entity)
	if len(workerIDs) == 0 {
		return nil
	}

	assignments, err := s.payAssignmentRepo.GetEffectiveForWorkers(
		ctx,
		driverpayports.GetWorkerPayAssignmentsRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
			AsOf:       asOf,
		},
	)
	if err != nil {
		s.l.Warn("failed to load driver pay for detention",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	rates := make(map[pulid.ID]decimal.NullDecimal, len(assignments))
	for workerID, assignment := range assignments {
		if rate, ok := detentionPayRate(assignment); ok {
			rates[workerID] = decimal.NewNullDecimal(rate)
		}
	}

	return rates
}

// detentionPayRate is the rate of the first active detention component on
// the assignment's profile, after the assignment's own override of it, the
// rate settlement pays the driver.
func detentionPayRate(assignment *payassignment.WorkerPayAssignment) (decimal.Decimal, bool) {
	if assignment == nil || assignment.PayProfile == nil {
		return decimal.Zero, false
	}

	for _, comp := range assignment.PayProfile.Components {
		if comp == nil || !comp.IsActive || comp.Kind != driverpay.ComponentKindDetention {
			continue
		}
		if override, ok := assignment.OverrideFor(comp.ID); ok {
			return override, true
		}

		return comp.Rate, true
	}

	return decimal.Zero, false
}

// projectToWatchtower puts a running detention clock on the feed and takes
// it off once the occurrence closes. The names come from what the
// recalculation already loaded, so the live item reads the same as the one
// the nightly snapshot would write without a second round of queries.
func (s *Service) projectToWatchtower(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	p *computeStopParams,
) {
	if saved == nil {
		return
	}

	if !saved.IsOpen {
		s.watchtower.Resolve(
			ctx,
			p.tenantInfo,
			watchtower.SourceKindDetentionOccurrence,
			saved.ID.String(),
		)

		return
	}

	described := *saved
	if described.ShipmentProNumber == "" && p.shipment != nil {
		described.ShipmentProNumber = p.shipment.ProNumber
	}
	if described.CustomerName == "" && p.shipment != nil && p.shipment.Customer != nil {
		described.CustomerName = p.shipment.Customer.Name
	}
	if described.LocationName == "" && p.stop != nil && p.stop.Location != nil {
		described.LocationName = p.stop.Location.Name
	}

	s.watchtower.Upsert(ctx, watchtowersources.DescribeDetentionOccurrence(&described))
}

// publishOccurrenceOpened wakes the detention desk the moment a clock
// starts, and only then. Recalculation runs on every stop update, so an
// occurrence that was already running is not news; the event fires on the
// transition into open, not on every pass over one that already is.
func (s *Service) publishOccurrenceOpened(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	p *computeStopParams,
) {
	if saved == nil || !saved.IsOpen {
		return
	}
	if p.existing != nil && p.existing.IsOpen {
		return
	}

	agenteventports.PublishAgentEvent(ctx, s.publisher, agenteventports.AgentEvent{
		Kind:       agenttrigger.EventDetentionOccurrenceOpened,
		SubjectID:  saved.ID,
		TenantInfo: p.tenantInfo,
	})
}
