package checkcallservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const carrierLinkKeyPrefix = "ctl:"

type resolvedLink struct {
	ctx    context.Context
	link   *checkcall.TrackingLink
	tenant pagination.TenantInfo
	move   *shipment.ShipmentMove
}

func invalidLinkError() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This tracking link is no longer valid",
	)
}

func (s *Service) resolveToken(ctx context.Context, token string) (*resolvedLink, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, invalidLinkError()
	}
	link, err := s.links.GetByHash(ctx, tokenutils.Hash(token))
	if err != nil {
		return nil, err
	}
	if !link.IsUsable(s.now()) {
		return nil, invalidLinkError()
	}

	tenant := pagination.TenantInfo{OrgID: link.OrganizationID, BuID: link.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	move, err := s.loadMove(ctx, tenant, link.ShipmentMoveID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, invalidLinkError()
		}
		return nil, err
	}
	if !moveIsClosed(move) {
		assignment, aErr := s.carrierAssignments.GetActiveByMoveID(ctx, tenant, move.ID)
		if aErr != nil && !errortypes.IsNotFoundError(aErr) {
			return nil, aErr
		}
		if assignment == nil || assignment.CarrierID != link.CarrierID {
			return nil, invalidLinkError()
		}
	}
	return &resolvedLink{ctx: ctx, link: link, tenant: tenant, move: move}, nil
}

func (s *Service) PreviewByToken(
	ctx context.Context,
	token string,
) (*checkcallports.PublicTrackingView, error) {
	resolved, err := s.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	ctx = resolved.ctx

	view := &checkcallports.PublicTrackingView{
		CompanyName: s.companyName(ctx, resolved.tenant),
		CarrierName: s.carrierName(ctx, resolved.tenant, resolved.link.CarrierID),
		Closed:      moveIsClosed(resolved.move),
		ExpiresAt:   resolved.link.ExpiresAt,
	}
	if sh, shErr := s.shipmentRepo.GetByID(ctx, shipmentByID(resolved)); shErr == nil && sh != nil {
		view.ShipmentProNumber = sh.ProNumber
	}
	latest, err := s.repo.LatestForMove(ctx, resolved.tenant, resolved.move.ID)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		at := latest.ReportedAt
		view.LastCheckCallAt = &at
	}

	stops := resolved.move.ActiveStopsBySequence()
	view.Stops = make([]checkcallports.PublicTrackingStop, 0, len(stops))
	for _, stop := range stops {
		row := checkcallports.PublicTrackingStop{
			ID:          stop.ID,
			Sequence:    stop.Sequence,
			Type:        string(stop.Type),
			WindowStart: stop.ScheduledWindowStart,
			WindowEnd:   stop.ScheduledWindowEnd,
			ArrivedAt:   stop.ActualArrival,
			DepartedAt:  stop.ActualDeparture,
		}
		if stop.Location != nil {
			row.LocationName = stop.Location.Name
			row.City = stop.Location.City
			if stop.Location.State != nil {
				row.StateCode = stop.Location.State.Abbreviation
			}
		}
		view.Stops = append(view.Stops, row)
	}
	return view, nil
}

func (s *Service) PostByToken(
	ctx context.Context,
	token string,
	input *checkcallports.PublicCheckCallInput,
) (*checkcallports.PublicCheckCallResult, error) {
	resolved, err := s.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if moveIsClosed(resolved.move) {
		return nil, errortypes.NewBusinessError("This load is no longer being tracked")
	}
	if input == nil {
		return nil, errortypes.NewValidationError("", errortypes.ErrInvalid, "Request is required")
	}
	clientEventID := strings.TrimSpace(input.ClientEventID)
	if err = validateClientEventID(clientEventID); err != nil {
		return nil, err
	}

	details := input.Details
	reportedAt, multiErr := normalizeDetails(normalizeInput{
		details: &details,
		now:     s.now(),
		policy:  clampFutureTimes,
	})
	if multiErr != nil {
		return nil, multiErr
	}
	if details.Kind.IsStopVisit() && findStop(resolved.move, details.StopID) == nil {
		return nil, errortypes.NewValidationError(
			"stopId",
			errortypes.ErrInvalid,
			"Choose a stop on this load",
		)
	}

	if clientEventID == "" {
		clientEventID = checkcall.NewID().String()
	}
	result, err := s.record(resolved.ctx, &recordInput{
		tenant:     resolved.tenant,
		move:       resolved.move,
		source:     checkcall.SourceCarrier,
		sourceKey:  carrierLinkKeyPrefix + resolved.link.ID.String() + ":" + clientEventID,
		linkID:     resolved.link.ID,
		details:    &details,
		reportedAt: reportedAt,
	})
	if err != nil {
		return nil, err
	}

	if tErr := s.links.Touch(
		resolved.ctx,
		resolved.tenant,
		resolved.link.ID,
		s.now(),
	); tErr != nil {
		s.l.Warn("failed to record carrier tracking link use",
			zap.String("linkId", resolved.link.ID.String()),
			zap.Error(tErr))
	}
	return &checkcallports.PublicCheckCallResult{
		Outcome:  result.Outcome,
		Reason:   result.Reason,
		Replayed: result.Replayed,
	}, nil
}

func shipmentByID(resolved *resolvedLink) *shipmentports.GetShipmentByIDRequest {
	return &shipmentports.GetShipmentByIDRequest{
		ID:         resolved.move.ShipmentID,
		TenantInfo: resolved.tenant,
	}
}
