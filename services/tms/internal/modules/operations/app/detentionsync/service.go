package detentionsync

import (
	"context"

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
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// recordChargeComment narrates a detention charge onto the shipment thread.
//
// The charge lands on the invoice as one accessorial line with no explanation
// attached, and the person who has to defend it is rarely the person who was
// watching the clock. Writing the derivation into the shipment's comments puts
// the reason next to the money, where anyone reopening the shipment will find
// it.
func (s *Service) recordChargeComment(ctx context.Context, p chargeCommentParams) {
	if s.commentRepo == nil || p.occurrence == nil || p.occurrence.ShipmentID.IsNil() {
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
	if s.agreementRepo == nil || entity.RateAgreementID == nil ||
		entity.RateAgreementID.IsNil() {
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

type Service struct {
	l                 *zap.Logger
	policyRepo        detentionports.DetentionPolicyRepository
	occurrenceRepo    detentionports.DetentionOccurrenceRepository
	evidenceRepo      detentionports.DetentionEvidenceRepository
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	agreementRepo     rateagreementports.RateAgreementRepository
	payAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	payProfileRepo    driverpayports.PayProfileRepository
	commentRepo       shipmentcommentports.ShipmentCommentRepository
	watchtower        watchtowerports.WatchtowerProjector
	publisher         agenteventports.AgentEventPublisher
	now               func() int64
	effects           *detentioneffects.Effects
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
	PayProfileRepo    driverpayports.PayProfileRepository
	CommentRepo       shipmentcommentports.ShipmentCommentRepository
	Watchtower        watchtowerports.WatchtowerProjector `optional:"true"`
	Publisher         agenteventports.AgentEventPublisher `optional:"true"`
	Now               func() int64                        `optional:"true"`
	Effects           *detentioneffects.Effects
}

func New(p Params) *Service {
	now := p.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		l:                 p.Logger.Named("service.detention"),
		policyRepo:        p.PolicyRepo,
		occurrenceRepo:    p.OccurrenceRepo,
		evidenceRepo:      p.EvidenceRepo,
		accessorialRepo:   p.AccessorialRepo,
		agreementRepo:     p.AgreementRepo,
		payAssignmentRepo: p.PayAssignmentRepo,
		payProfileRepo:    p.PayProfileRepo,
		commentRepo:       p.CommentRepo,
		watchtower:        p.Watchtower,
		publisher:         p.Publisher,
		now:               now,
		effects:           p.Effects,
	}
}

// SyncShipment recomputes every stop's detention for a shipment.
//
// Frozen occurrences are never rewritten: once a charge has been billed,
// waived, or disputed the customer has seen a number, and silently changing it
// is how carriers lose credibility in a dispute.
func (s *Service) SyncShipment(
	ctx context.Context,
	entity *shipment.Shipment,
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

	commodityIDs := shipmentCommodityIDs(entity)
	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	now := s.now()

	// The contract's own accessorial prices, read once for the whole shipment.
	prices := s.resolveContractPrices(ctx, entity, tenantInfo, now)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		payRate := s.resolveDriverPayRate(ctx, tenantInfo, move, now)

		for _, stop := range move.Stops {
			if !stopIsBillable(stop) {
				continue
			}

			existing, err := s.occurrenceRepo.GetByStop(
				ctx,
				&detentionports.GetOccurrenceByStopRequest{
					StopID:     stop.ID,
					TenantInfo: tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}

			if existing != nil && existing.IsFrozen() {
				result.Skipped++
				result.Occurrences = append(result.Occurrences, existing)
				shipmentAccrued = shipmentAccrued.Add(existing.BillableAmount)
				result.TotalAmount = result.TotalAmount.Add(existing.BillableAmount)
				continue
			}

			occurrence, err := s.computeStop(ctx, computeStopParams{
				shipment:        entity,
				move:            move,
				stop:            stop,
				existing:        existing,
				tenantInfo:      tenantInfo,
				commodityIDs:    commodityIDs,
				location:        location,
				payRate:         payRate,
				dayAccrued:      dayAccrued,
				shipmentAccrued: shipmentAccrued,
				prices:          prices,
				now:             now,
			})
			if err != nil {
				log.Error("failed to compute detention for stop",
					zap.String("stopId", stop.ID.String()), zap.Error(err))
				return nil, err
			}

			if occurrence == nil {
				continue
			}

			result.Occurrences = append(result.Occurrences, occurrence)
			shipmentAccrued = shipmentAccrued.Add(occurrence.BillableAmount)
			result.TotalAmount = result.TotalAmount.Add(occurrence.BillableAmount)
		}
	}

	return result, nil
}

func (s *Service) computeStop(
	ctx context.Context,
	p computeStopParams,
) (*detention.DetentionOccurrence, error) {
	matchInput := detentionpolicy.MatchInput{
		CustomerID:     p.shipment.CustomerID,
		LocationID:     p.stop.LocationID,
		ShipmentTypeID: p.shipment.ShipmentTypeID,
		ServiceTypeID:  p.shipment.ServiceTypeID,
		CommodityIDs:   p.commodityIDs,
		StopType:       p.stop.Type,
		ScheduleType:   p.stop.ScheduleType,
		Timestamp:      *p.stop.ActualArrival,
	}

	candidates, err := s.policyRepo.GetResolutionCandidates(
		ctx,
		&detentionports.GetResolutionCandidatesRequest{
			TenantInfo: p.tenantInfo,
			CustomerID: p.shipment.CustomerID,
			LocationID: p.stop.LocationID,
		},
	)
	if err != nil {
		return nil, err
	}

	resolution := detentionengine.ResolvePolicy(candidates, &matchInput)
	if !resolution.Found() {
		return nil, nil
	}

	policy := resolution.Policy

	accessorial, err := s.accessorialRepo.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID:         policy.AccessorialChargeID,
			TenantInfo: &p.tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	// The contract's price wins over the organization default, which is what
	// makes the rate confirmation and the invoice agree on detention.
	flatRate := accessorial.Amount
	rateUnit := accessorial.RateUnit

	if contractRate, contractUnit, priced := p.prices.priceFor(accessorial); priced {
		flatRate = contractRate
		rateUnit = contractUnit
	}

	snapshot := detentionpolicy.NewPolicySnapshot(&detentionpolicy.SnapshotInput{
		Policy:          policy,
		StopType:        string(p.stop.Type),
		FreeMinutes:     policy.FreeMinutesForStopType(p.stop.Type),
		AccessorialCode: accessorial.Code,
		FlatRate:        flatRate,
		FlatRateUnit:    detentionengine.TierUnitFromAccessorial(rateUnit),
		CapturedAt:      p.now,
	})

	var noticeSentAt *int64
	if p.existing != nil {
		noticeSentAt = p.existing.NoticeSentAt
	}

	computed := detentionengine.Compute(detentionengine.ComputeInput{
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
		return nil, nil
	}

	for day, amount := range computed.DayAllocation {
		p.dayAccrued[day] = p.dayAccrued[day].Add(amount)
	}

	occurrence := buildOccurrence(p, policy, snapshot, computed)

	saved, err := s.occurrenceRepo.Upsert(ctx, occurrence)
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

	return saved, nil
}

// recordRecalculation appends an evidence link whenever a stored amount moves,
// so the reason a number changed is never lost.
func (s *Service) recordRecalculation(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	existing *detention.DetentionOccurrence,
) {
	if s.evidenceRepo == nil || saved == nil {
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

// resolveDriverPayRate finds the detention rate the assigned driver is paid, so
// the AR and AP sides of the same dwell can be compared on one record.
func (s *Service) resolveDriverPayRate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	asOf int64,
) decimal.NullDecimal {
	if s.payAssignmentRepo == nil || s.payProfileRepo == nil {
		return decimal.NullDecimal{}
	}

	if move.Assignment == nil || move.Assignment.PrimaryWorkerID == nil ||
		move.Assignment.PrimaryWorkerID.IsNil() {
		return decimal.NullDecimal{}
	}

	assignment, err := s.payAssignmentRepo.GetEffectiveForWorker(
		ctx,
		driverpayports.GetWorkerPayAssignmentRequest{
			TenantInfo: tenantInfo,
			WorkerID:   *move.Assignment.PrimaryWorkerID,
			AsOf:       asOf,
		},
	)
	if err != nil || assignment == nil {
		return decimal.NullDecimal{}
	}

	profile, err := s.payProfileRepo.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
		TenantInfo: tenantInfo,
		ID:         assignment.PayProfileID,
	})
	if err != nil || profile == nil {
		return decimal.NullDecimal{}
	}

	for _, comp := range profile.Components {
		if comp == nil || !comp.IsActive {
			continue
		}
		if comp.Kind == driverpay.ComponentKindDetention {
			return decimal.NewNullDecimal(comp.Rate)
		}
	}

	return decimal.NullDecimal{}
}

// projectToWatchtower puts a running detention clock on the feed and takes
// it off once the occurrence closes. The names come from what the
// recalculation already loaded, so the live item reads the same as the one
// the nightly snapshot would write without a second round of queries.
func (s *Service) projectToWatchtower(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	p computeStopParams,
) {
	if s.watchtower == nil || saved == nil {
		return
	}

	if !saved.IsOpen {
		s.watchtower.Resolve(
			ctx,
			p.tenantInfo,
			watchtower.SourceDetentionOccurrence,
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
	p computeStopParams,
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
