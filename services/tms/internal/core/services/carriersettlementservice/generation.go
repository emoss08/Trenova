package carriersettlementservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type GenerateBatchRequest struct {
	TenantInfo  pagination.TenantInfo
	PeriodStart int64
	PeriodEnd   int64
	Name        string
	Notes       string
}

// PeriodBounds aliases the shared settlement period type.
type PeriodBounds = settlementshared.PeriodBounds

func ResolveCurrentPeriod(control *tenant.CarrierSettlementControl, now int64) PeriodBounds {
	return settlementshared.ResolvePeriod(
		control.PayPeriodFrequency,
		control.PeriodEndDayOfWeek,
		control.PayDelayDays,
		now,
	)
}

func (s *Service) GenerateBatch(
	ctx context.Context,
	req *GenerateBatchRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlementBatch, error) {
	if err := requireActor(actor, "Carrier settlement batch generation"); err != nil {
		return nil, err
	}
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	bounds := PeriodBounds{PeriodStart: req.PeriodStart, PeriodEnd: req.PeriodEnd}
	if bounds.PeriodStart == 0 || bounds.PeriodEnd == 0 {
		bounds = ResolveCurrentPeriod(control, now)
	} else {
		bounds.PayDate = time.Unix(bounds.PeriodEnd, 0).UTC().
			AddDate(0, 0, control.PayDelayDays).Unix()
	}
	if bounds.PeriodEnd <= bounds.PeriodStart {
		return nil, errortypes.NewValidationError(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}

	batch, err := s.resolveOpenBatch(ctx, req, bounds, actor, now)
	if err != nil {
		return nil, err
	}

	carrierIDs, err := s.costEventRepo.ListCarrierIDsWithPendingEvents(
		ctx,
		repositories.ListCarriersWithPendingEventsRequest{
			TenantInfo: req.TenantInfo,
			PeriodEnd:  bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}

	var generated int
	for _, carrierID := range carrierIDs {
		settlement, genErr := s.GenerateForCarrier(ctx, &GenerateForCarrierRequest{
			TenantInfo:  req.TenantInfo,
			CarrierID:   carrierID,
			PeriodStart: bounds.PeriodStart,
			PeriodEnd:   bounds.PeriodEnd,
			PayDate:     bounds.PayDate,
			BatchID:     &batch.ID,
		}, actor)
		if genErr != nil {
			s.l.Error("failed to generate carrier settlement",
				zap.Error(genErr), zap.String("carrierId", carrierID.String()))
			continue
		}
		if settlement != nil {
			generated++
		}
	}

	batch, err = s.batchRepo.RecalculateAggregates(ctx, req.TenantInfo, batch.ID)
	if err != nil {
		return nil, err
	}

	s.l.Info("carrier settlement batch generated",
		zap.String("batchId", batch.ID.String()),
		zap.Int("newSettlements", generated),
		zap.Int("settlements", batch.SettlementCount))
	return batch, nil
}

func (s *Service) resolveOpenBatch(
	ctx context.Context,
	req *GenerateBatchRequest,
	bounds PeriodBounds,
	actor *serviceports.RequestActor,
	now int64,
) (*carriersettlement.CarrierSettlementBatch, error) {
	existing, err := s.batchRepo.GetForPeriod(
		ctx,
		req.TenantInfo,
		bounds.PeriodStart,
		bounds.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	name := req.Name
	if name == "" {
		name = "Carrier pay period ending " +
			time.Unix(bounds.PeriodEnd, 0).UTC().AddDate(0, 0, -1).Format("Jan 2, 2006")
	}

	batch := &carriersettlement.CarrierSettlementBatch{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         carriersettlement.BatchStatusOpen,
		Name:           name,
		PeriodStart:    bounds.PeriodStart,
		PeriodEnd:      bounds.PeriodEnd,
		PayDate:        bounds.PayDate,
		CurrencyCode:   money.DefaultCurrencyCode,
		Notes:          req.Notes,
		GeneratedByID:  actor.UserID,
		GeneratedAt:    &now,
	}
	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.batchRepo.Create(ctx, batch)
}

type GenerateForCarrierRequest struct {
	TenantInfo  pagination.TenantInfo
	CarrierID   pulid.ID
	PeriodStart int64
	PeriodEnd   int64
	PayDate     int64
	BatchID     *pulid.ID
}

func (s *Service) GenerateForCarrier(
	ctx context.Context,
	req *GenerateForCarrierRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	if err := requireActor(actor, "Carrier settlement generation"); err != nil {
		return nil, err
	}
	exists, err := s.settlementRepo.ExistsForCarrierPeriod(
		ctx,
		req.TenantInfo,
		req.CarrierID,
		req.PeriodStart,
		req.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, nil //nolint:nilnil // nil settlement means this carrier period is already settled
	}

	var created *carriersettlement.CarrierSettlement
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		settlement, events, buildErr := s.buildSettlement(txCtx, req)
		if buildErr != nil {
			return buildErr
		}
		if settlement == nil {
			return nil
		}

		number, seqErr := s.generator.GenerateCarrierSettlementNumber(
			txCtx,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			"",
			"",
		)
		if seqErr != nil {
			return seqErr
		}
		settlement.SettlementNumber = number

		var txErr error
		created, txErr = s.settlementRepo.Create(txCtx, settlement)
		if txErr != nil {
			return txErr
		}

		eventIDs := make([]pulid.ID, 0, len(events))
		adjustmentIDs := make([]pulid.ID, 0, len(events))
		for _, event := range events {
			eventIDs = append(eventIDs, event.ID)
			if event.EventType == carriersettlement.CostEventTypeAdjustment {
				adjustmentIDs = append(adjustmentIDs, event.ID)
			}
		}
		if txErr = s.costEventRepo.MarkAttached(
			txCtx, req.TenantInfo, eventIDs, created.ID,
		); txErr != nil {
			return txErr
		}
		return s.invoiceMatchRepo.UpdateSettlementForAdjustmentEvents(
			txCtx, req.TenantInfo, adjustmentIDs, created.ID,
		)
	})
	if err != nil {
		// The partial unique index on (carrier, period) makes concurrent
		// generation idempotent: the loser treats the period as already settled.
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, nil //nolint:nilnil // nil settlement means this carrier period is already settled
		}
		return nil, err
	}
	if created == nil {
		return nil, nil //nolint:nilnil // nil settlement means the carrier had no pending cost
	}

	s.logSettlementAudit(ctx, created, nil, actor.UserID, permission.OpCreate,
		"Carrier settlement generated for period")
	return created, nil
}

func (s *Service) buildSettlement(
	ctx context.Context,
	req *GenerateForCarrierRequest,
) (*carriersettlement.CarrierSettlement, []*carriersettlement.CostEvent, error) {
	events, err := s.costEventRepo.ListPendingByCarrier(
		ctx,
		&repositories.ListPendingCostEventsRequest{
			TenantInfo: req.TenantInfo,
			CarrierID:  req.CarrierID,
			PeriodEnd:  req.PeriodEnd,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if len(events) == 0 {
		return nil, nil, nil
	}

	settlement := &carriersettlement.CarrierSettlement{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CarrierID:      req.CarrierID,
		BatchID:        req.BatchID,
		Status:         carriersettlement.StatusDraft,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		PayDate:        req.PayDate,
		CurrencyCode:   money.DefaultCurrencyCode,
	}

	lines, shipmentCount := buildEventLines(events)
	settlement.Lines = lines
	settlement.ShipmentCount = shipmentCount
	if len(events) > 0 && events[0].CurrencyCode != "" {
		settlement.CurrencyCode = events[0].CurrencyCode
	}
	settlement.SyncTotals()

	return settlement, events, nil
}

func buildEventLines(
	events []*carriersettlement.CostEvent,
) (lines []*carriersettlement.CarrierSettlementLine, shipmentCount int) {
	lines = make([]*carriersettlement.CarrierSettlementLine, 0, len(events))
	shipments := make(map[pulid.ID]struct{}, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.ShipmentID != nil && !event.ShipmentID.IsNil() {
			shipments[*event.ShipmentID] = struct{}{}
		}
		eventID := event.ID
		description := event.Description
		if description == "" {
			description = event.EventType.String()
		}
		lines = append(lines, &carriersettlement.CarrierSettlementLine{
			EventType:   event.EventType,
			Description: description,
			AmountMinor: event.AmountMinor,
			CostEventID: &eventID,
			ShipmentID:  event.ShipmentID,
			MoveID:      event.MoveID,
			ProNumber:   event.ProNumber,
		})
	}
	return lines, len(shipments)
}
