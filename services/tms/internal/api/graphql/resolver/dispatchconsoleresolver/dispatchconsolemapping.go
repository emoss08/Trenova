package dispatchconsoleresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/authctx"
)

func mapDispatchBoard(board *dispatchconsoleservice.Board) *gqlmodel.DispatchBoard {
	if board == nil {
		return nil
	}

	moves := make([]*gqlmodel.DispatchBoardMove, 0, len(board.Moves))
	for _, move := range board.Moves {
		moves = append(moves, mapDispatchBoardMove(move))
	}

	drivers := make([]*gqlmodel.DispatchBoardDriver, 0, len(board.Drivers))
	for _, driver := range board.Drivers {
		drivers = append(drivers, mapDispatchBoardDriver(driver))
	}

	return &gqlmodel.DispatchBoard{
		Moves:            moves,
		Drivers:          drivers,
		DriversTruncated: board.DriversTruncated,
		Summary:          mapDispatchSummary(board.Summary),
		WindowStart:      int(board.WindowStart),
		WindowEnd:        int(board.WindowEnd),
		GeneratedAt:      int(board.GeneratedAt),
	}
}

func (r *QueryResolver) attachCoverageProposals(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	moves []*dispatchconsoleservice.BoardMove,
	mapped []*gqlmodel.DispatchBoardMove,
) error {
	if len(moves) == 0 ||
		!r.HasPermission(ctx, authCtx, permission.ResourceAgentProposal, permission.OpRead) {
		return nil
	}

	targets := make([]dispatchconsoleports.CoverageMove, 0, len(moves))
	for _, move := range moves {
		if move == nil || move.BoardMove == nil || move.IsCovered {
			continue
		}
		targets = append(targets, dispatchconsoleports.CoverageMove{
			MoveID:      move.MoveID,
			MoveVersion: move.MoveVersion,
		})
	}
	if len(targets) == 0 {
		return nil
	}

	proposals, err := r.DispatchAutoAssignService.CoverageProposals(
		ctx,
		&dispatchconsoleports.CoverageProposalsRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			Moves:      targets,
		},
	)
	if err != nil {
		return err
	}

	for i, move := range moves {
		if move == nil || move.BoardMove == nil {
			continue
		}
		coverage, ok := proposals[move.MoveID]
		if !ok {
			continue
		}
		if coverage.Pending != nil {
			mapped[i].PendingProposal = mapDispatchMoveProposal(coverage.Pending)
		}
		if coverage.Failed != nil {
			mapped[i].FailedProposal = mapDispatchMoveFailedProposal(coverage.Failed)
		}
	}

	return nil
}

func mapDispatchMoveProposal(
	coverage *dispatchconsoleports.PendingCoverage,
) *gqlmodel.DispatchMoveProposal {
	var expiresAt *int
	if coverage.ExpiresAt > 0 {
		value := int(coverage.ExpiresAt)
		expiresAt = &value
	}

	return &gqlmodel.DispatchMoveProposal{
		ID:         coverage.ProposalID.String(),
		WorkerID:   coverage.WorkerID.String(),
		WorkerName: coverage.WorkerName,
		TractorID:  coverage.TractorID.String(),
		TrailerID:  gqlid.IDPtr(coverage.TrailerID),
		Confidence: coverage.Confidence.InexactFloat64(),
		Rationale:  coverage.Rationale,
		CreatedAt:  int(coverage.CreatedAt),
		ExpiresAt:  expiresAt,
		Stale:      coverage.Stale,
	}
}

func mapDispatchMoveFailedProposal(
	failed *dispatchconsoleports.FailedCoverage,
) *gqlmodel.DispatchMoveFailedProposal {
	return &gqlmodel.DispatchMoveFailedProposal{
		ID:         failed.ProposalID.String(),
		WorkerID:   failed.WorkerID.String(),
		WorkerName: failed.WorkerName,
		Reason:     failed.Reason,
		FailedAt:   int(failed.FailedAt),
	}
}

func mapDispatchSummary(
	summary *dispatchboardports.BoardSummary,
) *gqlmodel.DispatchBoardSummary {
	if summary == nil {
		return &gqlmodel.DispatchBoardSummary{}
	}

	return &gqlmodel.DispatchBoardSummary{
		UncoveredMoves:       summary.UncoveredMoves,
		CoveredMoves:         summary.CoveredMoves,
		LateMoves:            summary.LateMoves,
		AtRiskMoves:          summary.AtRiskMoves,
		UnseatedDrivers:      summary.UnseatedDrivers,
		AvailableDrivers:     summary.AvailableDrivers,
		AssignedToday:        summary.AssignedToday,
		AverageDeadheadMiles: summary.AverageDeadhead,
		UtilizationPercent:   summary.UtilizationPct,
	}
}

func mapDispatchBoardMove(move *dispatchconsoleservice.BoardMove) *gqlmodel.DispatchBoardMove {
	if move == nil {
		return nil
	}

	return &gqlmodel.DispatchBoardMove{
		MoveID:                move.MoveID.String(),
		ShipmentID:            move.ShipmentID.String(),
		ProNumber:             move.ProNumber,
		BOL:                   move.BOL,
		MoveStatus:            string(move.MoveStatus),
		ShipmentStatus:        string(move.ShipmentStatus),
		Sequence:              int(move.Sequence),
		MoveCount:             move.MoveCount,
		Loaded:                move.Loaded,
		Distance:              move.Distance,
		Revenue:               move.Revenue,
		CustomerID:            gqlid.IDPtr(move.CustomerID),
		CustomerName:          move.CustomerName,
		ServiceTypeID:         gqlid.IDPtr(move.ServiceTypeID),
		ServiceTypeCode:       move.ServiceTypeCode,
		RequiredTractorTypeID: gqlid.IDPtr(move.RequiredTractorTypeID),
		RequiredTrailerTypeID: gqlid.IDPtr(move.RequiredTrailerTypeID),
		TemperatureMin:        int16Ptr(move.TemperatureMin),
		TemperatureMax:        int16Ptr(move.TemperatureMax),
		HasHazmat:             move.HasHazmat,
		HasActiveHold:         move.HasActiveHold,

		Urgency:         string(move.Urgency),
		MinutesToPickup: int(move.MinutesToPickup),
		IsCovered:       move.IsCovered,

		OriginStopID:        gqlid.IDPtr(move.OriginStopID),
		OriginLocationID:    gqlid.IDPtr(move.OriginLocationID),
		OriginName:          move.OriginName,
		OriginCity:          move.OriginCity,
		OriginState:         move.OriginState,
		OriginLatitude:      move.OriginLatitude,
		OriginLongitude:     move.OriginLongitude,
		OriginWindowStart:   int(move.OriginWindowStart),
		OriginWindowEnd:     gqlscalar.IntPtr(move.OriginWindowEnd),
		OriginActualArrival: gqlscalar.IntPtr(move.OriginActualArrive),

		DestinationStopID:      gqlid.IDPtr(move.DestinationStopID),
		DestinationLocationID:  gqlid.IDPtr(move.DestinationLocationID),
		DestinationName:        move.DestinationName,
		DestinationCity:        move.DestinationCity,
		DestinationState:       move.DestinationState,
		DestinationLatitude:    move.DestinationLatitude,
		DestinationLongitude:   move.DestinationLongitude,
		DestinationWindowStart: int(move.DestinationWindowStart),
		DestinationWindowEnd:   gqlscalar.IntPtr(move.DestinationWindowEnd),

		AssignmentID:          gqlid.IDPtr(move.AssignmentID),
		AssignedWorkerID:      gqlid.IDPtr(move.AssignedWorkerID),
		AssignedWorkerName:    move.AssignedWorkerName,
		AssignedTractorID:     gqlid.IDPtr(move.AssignedTractorID),
		AssignedTractorCode:   move.AssignedTractorCode,
		AssignedTrailerID:     gqlid.IDPtr(move.AssignedTrailerID),
		AssignedTrailerCode:   move.AssignedTrailerCode,
		AssignmentAckStatus:   string(move.AssignmentAckStatus),
		PreviousMoveTrailerID: gqlid.IDPtr(move.PreviousMoveTrailerID),

		LiveTender: mapDispatchBoardTenderSummary(move.LiveTender),
		PlanPrice:  mapDispatchMovePlanPrice(move.PlanPrice),
	}
}

func mapDispatchMovePlanPrice(
	price *dispatchplanprice.DispatchPlanPrice,
) *gqlmodel.DispatchMovePlanPrice {
	if price == nil {
		return nil
	}

	var reason *gqlmodel.DispatchUncoveredReason
	if !price.Covered() {
		value := gqlmodel.DispatchUncoveredReason(price.UncoveredReason)
		reason = &value
	}

	return &gqlmodel.DispatchMovePlanPrice{
		WorkerID:           gqlid.IDPtr(price.WorkerID),
		WorkerName:         price.WorkerName,
		CostCents:          int(price.CostCents),
		CoveragePriceCents: int(price.CoveragePriceCents),
		DriverValueCents:   int(price.DriverValueCents),
		UncoveredReason:    reason,
		Alternatives:       mapDispatchPlanAlternatives(price.Alternatives),
		GeneratedAt:        int(price.GeneratedAt),
	}
}

func mapDispatchPlanAlternatives(
	alternatives []dispatchplanprice.Alternative,
) []*gqlmodel.DispatchPlanAlternative {
	mapped := make([]*gqlmodel.DispatchPlanAlternative, 0, len(alternatives))
	for _, alternative := range alternatives {
		mapped = append(mapped, &gqlmodel.DispatchPlanAlternative{
			WorkerID:   alternative.WorkerID.String(),
			WorkerName: alternative.WorkerName,
			ExtraCents: int(alternative.ExtraCents),
		})
	}

	return mapped
}

func mapDispatchDriverPlannedMove(
	planned *dispatchconsoleservice.PlannedMove,
) *gqlmodel.DispatchDriverPlannedMove {
	if planned == nil {
		return nil
	}

	return &gqlmodel.DispatchDriverPlannedMove{
		MoveID:      planned.MoveID.String(),
		ProNumber:   planned.ProNumber,
		CostCents:   int(planned.CostCents),
		ValueCents:  int(planned.ValueCents),
		GeneratedAt: int(planned.GeneratedAt),
	}
}

func mapDispatchBoardTenderSummary(
	summary *dispatchconsoleservice.MoveTenderSummary,
) *gqlmodel.DispatchBoardTenderSummary {
	if summary == nil {
		return nil
	}

	return &gqlmodel.DispatchBoardTenderSummary{
		ID:                    summary.ID.String(),
		Status:                summary.Status,
		Mode:                  summary.Mode,
		CurrentRank:           int(summary.CurrentRank),
		OfferCount:            summary.OfferCount,
		CurrentCarrierName:    summary.CurrentCarrierName,
		CurrentOfferExpiresAt: gqlscalar.IntPtr(summary.CurrentOfferExpiresAt),
	}
}

func mapDispatchBoardDriver(
	driver *dispatchconsoleservice.BoardDriver,
) *gqlmodel.DispatchBoardDriver {
	if driver == nil {
		return nil
	}

	commitments := make([]*gqlmodel.DispatchCommitment, 0, len(driver.Commitments))
	for _, commitment := range driver.Commitments {
		commitments = append(commitments, &gqlmodel.DispatchCommitment{
			MoveID:               commitment.MoveID.String(),
			ShipmentID:           commitment.ShipmentID.String(),
			ProNumber:            commitment.ProNumber,
			MoveStatus:           string(commitment.MoveStatus),
			WindowStart:          int(commitment.WindowStart),
			WindowEnd:            int(commitment.WindowEnd),
			DestinationCity:      commitment.DestinationCty,
			DestinationState:     commitment.DestinationSt,
			DestinationLatitude:  commitment.DestLatitude,
			DestinationLongitude: commitment.DestLongitude,
			TrailerID:            gqlid.IDPtr(commitment.TrailerID),
		})
	}

	timeOff := make([]*gqlmodel.DispatchTimeOff, 0, len(driver.TimeOff))
	for _, pto := range driver.TimeOff {
		timeOff = append(timeOff, &gqlmodel.DispatchTimeOff{
			StartDate: int(pto.StartDate),
			EndDate:   int(pto.EndDate),
			Type:      string(pto.Type),
		})
	}

	return &gqlmodel.DispatchBoardDriver{
		WorkerID:             driver.WorkerID.String(),
		FirstName:            driver.FirstName,
		LastName:             driver.LastName,
		WorkerType:           string(driver.WorkerType),
		DriverType:           string(driver.DriverType),
		FleetCodeID:          gqlid.IDPtr(driver.FleetCodeID),
		FleetCodeName:        driver.FleetCodeName,
		City:                 driver.City,
		StateAbbreviation:    driver.StateAbbr,
		PostalCode:           driver.PostalCode,
		ProfilePicURL:        driver.ProfilePicURL,
		AssignmentBlocked:    driver.AssignmentNote,
		AvailableForDispatch: driver.AvailableForDispatch,

		TractorID:        gqlid.IDPtr(driver.TractorID),
		TractorCode:      driver.TractorCode,
		TractorTypeID:    gqlid.IDPtr(driver.TractorTypeID),
		TractorAvailable: driver.TractorStatusOK,
		OpenAssignments:  driver.OpenAssignments,

		Availability:     string(driver.Availability),
		DutyStatus:       driver.DutyStatus,
		DriveRemainingMs: int(driver.DriveRemainingMs),
		ShiftRemainingMs: int(driver.ShiftRemainingMs),
		CycleRemainingMs: int(driver.CycleRemainingMs),
		BreakRemainingMs: int(driver.BreakRemainingMs),
		HosRecordedAt:    int(driver.HOSRecordedAt),
		HosIsStale:       driver.HOSIsStale,

		Latitude:           driver.Latitude,
		Longitude:          driver.Longitude,
		FormattedLocation:  driver.FormattedLocation,
		PositionRecordedAt: int(driver.PositionAt),

		ProjectedTimeAvailable: int(driver.ProjectedTimeAvailable),
		CommittedMiles:         driver.CommittedMiles,
		CommittedRevenue:       driver.CommittedRevenue,
		Commitments:            commitments,
		TimeOff:                timeOff,
		Findings:               mapDispatchFindings(driver.Findings),
		PlannedMove:            mapDispatchDriverPlannedMove(driver.PlannedMove),
	}
}

func mapDispatchCandidate(
	score *dispatchplanports.CandidateScore,
) *gqlmodel.DispatchCandidate {
	if score == nil {
		return nil
	}

	lines := make([]*gqlmodel.DispatchCostLine, 0, len(score.Cost.Lines))
	for _, line := range score.Cost.Lines {
		lines = append(lines, &gqlmodel.DispatchCostLine{
			Component: string(line.Component),
			Cents:     int(line.Cents),
			Soft:      line.Component.Soft(),
		})
	}

	return &gqlmodel.DispatchCandidate{
		WorkerID:               score.WorkerID.String(),
		WorkerName:             score.WorkerName,
		TractorID:              gqlid.IDPtr(score.TractorID),
		TrailerID:              gqlid.IDPtr(score.TrailerID),
		MoveID:                 score.MoveID.String(),
		CostCents:              int(score.Cost.TotalCents),
		CostLines:              lines,
		Verdict:                score.Verdict,
		Blocked:                score.Blocked(),
		DeadheadMiles:          score.DeadheadMiles,
		OriginSource:           string(score.OriginSource),
		LoadedMiles:            score.LoadedMiles,
		EstimatedDriveMs:       int(score.EstimatedDriveMs),
		RestMs:                 int(score.RestMs),
		ProjectedArrival:       int(score.ProjectedArrival),
		ProjectedDelivery:      int(score.ProjectedDelivery),
		ProjectedFreeAt:        int(score.ProjectedFreeAt),
		MinutesOfSlack:         int(score.MinutesOfSlack),
		DeliveryLateMinutes:    int(score.DeliveryLateMinutes),
		DriveRemainingMs:       int(score.DriveRemainingMs),
		ShiftRemainingMs:       int(score.ShiftRemainingMs),
		CycleRemainingMs:       int(score.CycleRemainingMs),
		ProjectedTimeAvailable: int(score.ProjectedAvailable),
		HosStrategy:            score.HOSStrategy,
		HosRestStartDeadline:   int(score.HOSRestStartDeadline),
		HosProjectedDriveMs:    int(score.HOSProjectedDriveMs),
		HosProjectedShiftMs:    int(score.HOSProjectedShiftMs),
		HosProjectedCycleMs:    int(score.HOSProjectedCycleMs),
		Findings:               mapDispatchFindings(score.Findings),
	}
}

func mapDispatchFindings(
	findings []dispatchfinding.Finding,
) []*gqlmodel.DispatchFinding {
	mapped := make([]*gqlmodel.DispatchFinding, 0, len(findings))
	for i := range findings {
		finding := findings[i]
		mapped = append(mapped, &gqlmodel.DispatchFinding{
			Code:       finding.Code,
			Severity:   string(finding.Severity),
			Field:      finding.Field,
			Message:    finding.Message,
			Regulation: gqlscalar.NonEmptyPtr(finding.Regulation),
		})
	}
	return mapped
}

func mapDispatchPlan(plan *dispatchconsoleports.DispatchPlan) *gqlmodel.DispatchPlan {
	if plan == nil {
		return nil
	}

	assignments := make([]*gqlmodel.DispatchPlannedAssignment, 0, len(plan.Assignments))
	for _, planned := range plan.Assignments {
		confidence, _ := planned.Confidence.Float64()
		candidate := planned.Candidate
		assignments = append(assignments, &gqlmodel.DispatchPlannedAssignment{
			MoveID:              planned.MoveID.String(),
			ProNumber:           planned.ProNumber,
			WorkerID:            candidate.WorkerID.String(),
			WorkerName:          candidate.WorkerName,
			TractorID:           gqlid.IDPtr(candidate.TractorID),
			TrailerID:           gqlid.IDPtr(candidate.TrailerID),
			Confidence:          confidence,
			Rationale:           planned.Rationale,
			AutoExecutable:      planned.AutoExecutable,
			ProposalID:          gqlid.IDPtr(planned.ProposalID),
			Candidate:           mapDispatchCandidate(candidate),
			CoveragePriceCents:  int(planned.CoveragePriceCents),
			DriverValueCents:    int(planned.DriverValueCents),
			Alternatives:        mapDispatchPlanAlternatives(planned.Alternatives),
			TourID:              gqlid.IDPtr(planned.TourID),
			SequenceIndex:       planned.SequenceIndex,
			ProjectedStartAt:    int(planned.ProjectedStartAt),
			ProjectedCompleteAt: int(planned.ProjectedCompleteAt),
		})
	}

	uncovered := make([]*gqlmodel.DispatchUncoveredMove, 0, len(plan.Uncovered))
	for _, item := range plan.Uncovered {
		uncovered = append(uncovered, &gqlmodel.DispatchUncoveredMove{
			MoveID:             item.MoveID.String(),
			ProNumber:          item.ProNumber,
			Reason:             gqlmodel.DispatchUncoveredReason(item.Reason),
			Explanation:        item.Explanation(),
			CoveragePriceCents: int(item.CoveragePriceCents),
			BestCandidate:      mapDispatchCandidate(item.BestCandidate),
		})
	}

	tours := make([]*gqlmodel.DispatchTour, 0, len(plan.Tours))
	for _, tour := range plan.Tours {
		moveIDs := make([]string, 0, len(tour.MoveIDs))
		for _, moveID := range tour.MoveIDs {
			moveIDs = append(moveIDs, moveID.String())
		}
		tours = append(tours, &gqlmodel.DispatchTour{
			TourID:     tour.TourID.String(),
			WorkerID:   tour.WorkerID.String(),
			WorkerName: tour.WorkerName,
			MoveIds:    moveIDs,
			StartsAt:   int(tour.StartsAt),
			EndsAt:     int(tour.EndsAt),
			CostCents:  int(tour.CostCents),
			EmptyMiles: tour.EmptyMiles,
		})
	}

	return &gqlmodel.DispatchPlan{
		RunID:          gqlid.IDPtr(plan.RunID),
		Assignments:    assignments,
		Uncovered:      uncovered,
		Tours:          tours,
		PlanningMode:   plan.Mode.String(),
		ShadowMode:     plan.ShadowMode,
		AutonomyTier:   string(plan.AutonomyTier),
		CostCents:      int(plan.CostCents),
		UncoveredCents: int(plan.UncoveredCents),
		GeneratedAt:    int(plan.GeneratedAt),
	}
}
