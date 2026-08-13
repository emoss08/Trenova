package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
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
		Moves:       moves,
		Drivers:     drivers,
		Summary:     mapDispatchSummary(board.Summary),
		WindowStart: int(board.WindowStart),
		WindowEnd:   int(board.WindowEnd),
		GeneratedAt: int(board.GeneratedAt),
	}
}

func mapDispatchSummary(
	summary *repositories.BoardSummary,
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
		Bol:                   move.BOL,
		MoveStatus:            string(move.MoveStatus),
		ShipmentStatus:        string(move.ShipmentStatus),
		Sequence:              int(move.Sequence),
		MoveCount:             move.MoveCount,
		Loaded:                move.Loaded,
		Distance:              move.Distance,
		Revenue:               move.Revenue,
		CustomerID:            idPtr(move.CustomerID),
		CustomerName:          move.CustomerName,
		ServiceTypeID:         idPtr(move.ServiceTypeID),
		ServiceTypeCode:       move.ServiceTypeCode,
		RequiredTractorTypeID: idPtr(move.RequiredTractorTypeID),
		RequiredTrailerTypeID: idPtr(move.RequiredTrailerTypeID),
		TemperatureMin:        int16Ptr(move.TemperatureMin),
		TemperatureMax:        int16Ptr(move.TemperatureMax),
		HasHazmat:             move.HasHazmat,
		HasActiveHold:         move.HasActiveHold,

		Urgency:         string(move.Urgency),
		MinutesToPickup: int(move.MinutesToPickup),
		IsCovered:       move.IsCovered,

		OriginStopID:        idPtr(move.OriginStopID),
		OriginLocationID:    idPtr(move.OriginLocationID),
		OriginName:          move.OriginName,
		OriginCity:          move.OriginCity,
		OriginState:         move.OriginState,
		OriginLatitude:      move.OriginLatitude,
		OriginLongitude:     move.OriginLongitude,
		OriginWindowStart:   int(move.OriginWindowStart),
		OriginWindowEnd:     intPtr(move.OriginWindowEnd),
		OriginActualArrival: intPtr(move.OriginActualArrive),

		DestinationStopID:      idPtr(move.DestinationStopID),
		DestinationLocationID:  idPtr(move.DestinationLocationID),
		DestinationName:        move.DestinationName,
		DestinationCity:        move.DestinationCity,
		DestinationState:       move.DestinationState,
		DestinationLatitude:    move.DestinationLatitude,
		DestinationLongitude:   move.DestinationLongitude,
		DestinationWindowStart: int(move.DestinationWindowStart),
		DestinationWindowEnd:   intPtr(move.DestinationWindowEnd),

		AssignmentID:          idPtr(move.AssignmentID),
		AssignedWorkerID:      idPtr(move.AssignedWorkerID),
		AssignedWorkerName:    move.AssignedWorkerName,
		AssignedTractorID:     idPtr(move.AssignedTractorID),
		AssignedTractorCode:   move.AssignedTractorCode,
		AssignedTrailerID:     idPtr(move.AssignedTrailerID),
		AssignedTrailerCode:   move.AssignedTrailerCode,
		AssignmentAckStatus:   string(move.AssignmentAckStatus),
		PreviousMoveTrailerID: idPtr(move.PreviousMoveTrailerID),

		LiveTender: mapDispatchBoardTenderSummary(move.LiveTender),
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
		CurrentOfferExpiresAt: intPtr(summary.CurrentOfferExpiresAt),
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
			TrailerID:            idPtr(commitment.TrailerID),
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
		FleetCodeID:          idPtr(driver.FleetCodeID),
		FleetCodeName:        driver.FleetCodeName,
		City:                 driver.City,
		StateAbbreviation:    driver.StateAbbr,
		PostalCode:           driver.PostalCode,
		ProfilePicURL:        driver.ProfilePicURL,
		AssignmentBlocked:    driver.AssignmentNote,
		AvailableForDispatch: driver.AvailableForDispatch,

		TractorID:        idPtr(driver.TractorID),
		TractorCode:      driver.TractorCode,
		TractorTypeID:    idPtr(driver.TractorTypeID),
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
	}
}

func mapDispatchCandidate(
	score *dispatchcandidateservice.CandidateScore,
) *gqlmodel.DispatchCandidate {
	if score == nil {
		return nil
	}

	factors := make([]*gqlmodel.DispatchScoreFactor, 0, len(score.Factors))
	for _, factor := range score.Factors {
		factors = append(factors, &gqlmodel.DispatchScoreFactor{
			Key:          string(factor.Key),
			Label:        factor.Label,
			Raw:          factor.Raw,
			Weight:       factor.Weight,
			Contribution: factor.Contribution,
			Detail:       factor.Detail,
		})
	}

	return &gqlmodel.DispatchCandidate{
		WorkerID:               score.WorkerID.String(),
		WorkerName:             score.WorkerName,
		TractorID:              idPtr(score.TractorID),
		TrailerID:              idPtr(score.TrailerID),
		MoveID:                 score.MoveID.String(),
		Score:                  score.Score,
		Verdict:                score.Verdict,
		Blocked:                score.Blocked(),
		DeadheadMiles:          score.DeadheadMiles,
		EstimatedDriveMs:       int(score.EstimatedDriveMs),
		ProjectedArrival:       int(score.ProjectedArrival),
		MinutesOfSlack:         int(score.MinutesOfSlack),
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
		Factors:                factors,
	}
}

func mapDispatchFindings(
	findings []dispatcheligibility.Finding,
) []*gqlmodel.DispatchFinding {
	mapped := make([]*gqlmodel.DispatchFinding, 0, len(findings))
	for i := range findings {
		finding := findings[i]
		mapped = append(mapped, &gqlmodel.DispatchFinding{
			Code:       finding.Code,
			Severity:   string(finding.Severity),
			Field:      finding.Field,
			Message:    finding.Message,
			Regulation: nonEmptyPtr(finding.Regulation),
		})
	}
	return mapped
}

func mapDispatchPlan(plan *services.DispatchPlan) *gqlmodel.DispatchPlan {
	if plan == nil {
		return nil
	}

	assignments := make([]*gqlmodel.DispatchPlannedAssignment, 0, len(plan.Assignments))
	for _, planned := range plan.Assignments {
		confidence, _ := planned.Confidence.Float64()
		assignments = append(assignments, &gqlmodel.DispatchPlannedAssignment{
			MoveID:         planned.MoveID.String(),
			ProNumber:      planned.ProNumber,
			WorkerID:       planned.WorkerID.String(),
			WorkerName:     planned.WorkerName,
			TractorID:      idPtr(planned.TractorID),
			TrailerID:      idPtr(planned.TrailerID),
			Confidence:     confidence,
			Rationale:      planned.Rationale,
			AutoExecutable: planned.AutoExecutable,
			ProposalID:     idPtr(planned.ProposalID),
			Score:          mapDispatchCandidate(planned.Score),

			TourID:                    idPtr(planned.TourID),
			SequenceIndex:             planned.SequenceIndex,
			ProjectedStartAt:          int(planned.ProjectedStartAt),
			ProjectedCompleteAt:       int(planned.ProjectedCompleteAt),
			ProjectedDriveRemainingMs: int(planned.ProjectedDriveRemains),
		})
	}

	uncovered := make([]*gqlmodel.DispatchUncoveredMove, 0, len(plan.Uncovered))
	for _, item := range plan.Uncovered {
		uncovered = append(uncovered, &gqlmodel.DispatchUncoveredMove{
			MoveID:              item.MoveID.String(),
			ProNumber:           item.ProNumber,
			Reason:              item.Reason,
			BestBlockedFindings: mapDispatchFindings(item.BestBlockedFindings),
		})
	}

	tours := make([]*gqlmodel.DispatchTour, 0, len(plan.Tours))
	for _, tour := range plan.Tours {
		moveIDs := make([]string, 0, len(tour.MoveIDs))
		for _, moveID := range tour.MoveIDs {
			moveIDs = append(moveIDs, moveID.String())
		}

		tours = append(tours, &gqlmodel.DispatchTour{
			TourID:             tour.TourID.String(),
			WorkerID:           tour.WorkerID.String(),
			WorkerName:         tour.WorkerName,
			MoveIds:            moveIDs,
			StartsAt:           int(tour.StartsAt),
			EndsAt:             int(tour.EndsAt),
			TotalScore:         tour.TotalScore,
			TotalDeadheadMiles: tour.TotalDeadheadMiles,
		})
	}

	return &gqlmodel.DispatchPlan{
		RunID:        idPtr(plan.RunID),
		Assignments:  assignments,
		Uncovered:    uncovered,
		Tours:        tours,
		PlanningMode: plan.PlanningMode,
		ShadowMode:   plan.ShadowMode,
		AutonomyTier: string(plan.AutonomyTier),
		TotalScore:   plan.TotalScore,
		GeneratedAt:  int(plan.GeneratedAt),
	}
}
