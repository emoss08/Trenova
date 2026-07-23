package driverportalservice

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type LoadScope string

const (
	LoadScopeActive  = LoadScope("Active")
	LoadScopeHistory = LoadScope("History")
)

type PortalProfile struct {
	WorkerID         pulid.ID `json:"workerId"`
	FirstName        string   `json:"firstName"`
	LastName         string   `json:"lastName"`
	Email            string   `json:"email"`
	PhoneNumber      string   `json:"phoneNumber"`
	WorkerType       string   `json:"workerType"`
	DriverType       string   `json:"driverType"`
	FleetCodeName    string   `json:"fleetCodeName"`
	OrganizationName string   `json:"organizationName"`
}

type PortalStop struct {
	ID                   pulid.ID `json:"id"`
	Type                 string   `json:"type"`
	Status               string   `json:"status"`
	Sequence             int64    `json:"sequence"`
	LocationName         string   `json:"locationName"`
	AddressLine          string   `json:"addressLine"`
	ScheduledWindowStart int64    `json:"scheduledWindowStart"`
	ScheduledWindowEnd   *int64   `json:"scheduledWindowEnd"`
	ActualArrival        *int64   `json:"actualArrival"`
	ActualDeparture      *int64   `json:"actualDeparture"`
}

type PortalLoad struct {
	AssignmentID  pulid.ID      `json:"assignmentId"`
	MoveID        pulid.ID      `json:"moveId"`
	ShipmentID    pulid.ID      `json:"shipmentId"`
	ProNumber     string        `json:"proNumber"`
	BOL           string        `json:"bol"`
	Status        string        `json:"status"`
	IsPrimary     bool          `json:"isPrimary"`
	TractorCode   string        `json:"tractorCode"`
	TrailerCode   string        `json:"trailerCode"`
	Pieces        *int64        `json:"pieces"`
	Weight        *int64        `json:"weight"`
	DistanceMiles *float64      `json:"distanceMiles"`
	PayGrossMinor *int64        `json:"payGrossMinor"`
	PayStatus     string        `json:"payStatus"`
	PayOnHold     bool          `json:"payOnHold"`
	AckStatus     string        `json:"ackStatus"`
	Stops         []*PortalStop `json:"stops"`
}

type PortalComment struct {
	ID         pulid.ID `json:"id"`
	Type       string   `json:"type"`
	Priority   string   `json:"priority"`
	Comment    string   `json:"comment"`
	AuthorName string   `json:"authorName"`
	CreatedAt  int64    `json:"createdAt"`
}

type PortalPeriodSummary struct {
	PeriodStart       int64 `json:"periodStart"`
	PeriodEnd         int64 `json:"periodEnd"`
	PayDate           int64 `json:"payDate"`
	AccruedGrossMinor int64 `json:"accruedGrossMinor"`
	EventCount        int   `json:"eventCount"`
}

type PortalEscrowSummary struct {
	Account      *driverpay.EscrowAccount       `json:"account"`
	Transactions []*driverpay.EscrowTransaction `json:"transactions"`
}

func (s *Service) ResolveWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.Worker, error) {
	wrk, err := s.portalRepo.GetWorkerByUserID(ctx, tenantInfo)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewAuthorizationError(
				"Your account is not linked to a driver profile. Contact your carrier.",
			)
		}
		return nil, err
	}
	return wrk, nil
}

func (s *Service) MyProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalProfile, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	profile := &PortalProfile{
		WorkerID:    wrk.ID,
		FirstName:   wrk.FirstName,
		LastName:    wrk.LastName,
		Email:       wrk.Email,
		PhoneNumber: wrk.PhoneNumber,
		WorkerType:  wrk.Type.String(),
		DriverType:  wrk.DriverType.String(),
	}
	if wrk.FleetCode != nil {
		profile.FleetCodeName = wrk.FleetCode.Code
	}
	if wrk.Organization != nil {
		profile.OrganizationName = wrk.Organization.Name
	}
	return profile, nil
}

func (s *Service) MyLoads(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	scope LoadScope,
	limit int,
) ([]*PortalLoad, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	statuses := []shipment.MoveStatus{
		shipment.MoveStatusNew,
		shipment.MoveStatusAssigned,
		shipment.MoveStatusInTransit,
	}
	if scope == LoadScopeHistory {
		statuses = []shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		}
	}

	assignments, err := s.portalRepo.ListWorkerLoads(ctx, &repositories.ListWorkerLoadsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
		Statuses:   statuses,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	loads := make([]*PortalLoad, 0, len(assignments))
	moveIDs := make([]pulid.ID, 0, len(assignments))
	for _, assignment := range assignments {
		load := buildPortalLoad(assignment, wrk.ID)
		if load != nil {
			loads = append(loads, load)
			moveIDs = append(moveIDs, load.MoveID)
		}
	}
	control, err := s.dashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if control.ShowLoadPay {
		if err = s.attachLoadPay(ctx, tenantInfo, wrk.ID, moveIDs, loads); err != nil {
			return nil, err
		}
	}
	if scope == LoadScopeActive {
		sort.SliceStable(loads, func(i, j int) bool {
			return firstStopWindow(loads[i]) < firstStopWindow(loads[j])
		})
	}
	return loads, nil
}

func (s *Service) attachLoadPay(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	moveIDs []pulid.ID,
	loads []*PortalLoad,
) error {
	if len(moveIDs) == 0 {
		return nil
	}
	events, err := s.payEventRepo.ListByMovesForWorker(ctx, tenantInfo, workerID, moveIDs)
	if err != nil {
		return err
	}

	type payInfo struct {
		grossMinor int64
		status     driversettlement.PayEventStatus
		onHold     bool
	}
	byMove := make(map[pulid.ID]*payInfo, len(events))
	for _, event := range events {
		if event.MoveID == nil {
			continue
		}
		info, ok := byMove[*event.MoveID]
		if !ok {
			info = &payInfo{status: event.Status}
			byMove[*event.MoveID] = info
		}
		info.grossMinor += event.GrossAmountMinor
		info.onHold = info.onHold || event.OnHold
		if event.Status == driversettlement.PayEventStatusAccrued {
			info.status = driversettlement.PayEventStatusAccrued
		}
	}

	for _, load := range loads {
		info, ok := byMove[load.MoveID]
		if !ok {
			continue
		}
		gross := info.grossMinor
		load.PayGrossMinor = &gross
		load.PayStatus = string(info.status)
		load.PayOnHold = info.onHold
	}
	return nil
}

func buildPortalLoad(assignment *shipment.Assignment, workerID pulid.ID) *PortalLoad {
	move := assignment.ShipmentMove
	if move == nil {
		return nil
	}

	load := &PortalLoad{
		AssignmentID: assignment.ID,
		MoveID:       move.ID,
		Status:       string(move.Status),
		AckStatus:    string(assignment.AckStatus),
		IsPrimary: assignment.PrimaryWorkerID != nil &&
			*assignment.PrimaryWorkerID == workerID,
		Stops: make([]*PortalStop, 0, len(move.Stops)),
	}
	if move.Shipment != nil {
		load.ShipmentID = move.Shipment.ID
		load.ProNumber = move.Shipment.ProNumber
		load.BOL = move.Shipment.BOL
		load.Pieces = move.Shipment.Pieces
		load.Weight = move.Shipment.Weight
	}
	load.DistanceMiles = move.Distance
	if assignment.Tractor != nil {
		load.TractorCode = assignment.Tractor.Code
	}
	if assignment.Trailer != nil {
		load.TrailerCode = assignment.Trailer.Code
	}
	for _, stop := range move.Stops {
		view := &PortalStop{
			ID:                   stop.ID,
			Type:                 string(stop.Type),
			Status:               string(stop.Status),
			Sequence:             stop.Sequence,
			AddressLine:          stop.AddressLine,
			ScheduledWindowStart: stop.ScheduledWindowStart,
			ScheduledWindowEnd:   stop.ScheduledWindowEnd,
			ActualArrival:        stop.ActualArrival,
			ActualDeparture:      stop.ActualDeparture,
		}
		if stop.Location != nil {
			view.LocationName = stop.Location.Name
		}
		load.Stops = append(load.Stops, view)
	}
	return load
}

func firstStopWindow(load *PortalLoad) int64 {
	if len(load.Stops) == 0 {
		return int64(1) << 62
	}
	return load.Stops[0].ScheduledWindowStart
}

func (s *Service) MyLoadComments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*PortalComment, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	assigned, err := s.portalRepo.WorkerAssignedToShipment(ctx, tenantInfo, wrk.ID, shipmentID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, errortypes.NewNotFoundError("Load not found")
	}

	comments, err := s.portalRepo.ListDriverShipmentComments(ctx, tenantInfo, shipmentID)
	if err != nil {
		return nil, err
	}

	views := make([]*PortalComment, 0, len(comments))
	for _, comment := range comments {
		view := &PortalComment{
			ID:        comment.ID,
			Type:      string(comment.Type),
			Priority:  string(comment.Priority),
			Comment:   comment.Comment,
			CreatedAt: comment.CreatedAt,
		}
		switch {
		case comment.User != nil:
			view.AuthorName = comment.User.Name
		case comment.Source == shipment.CommentSourceSystem:
			view.AuthorName = "System"
		default:
			view.AuthorName = "Dispatch"
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) MyStopAction(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	stopID pulid.ID,
	action repositories.StopActualAction,
) error {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if _, err = s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowStopActions },
		"Your carrier records arrivals and departures from dispatch — call in your status instead.",
	); err != nil {
		return err
	}

	assigned, err := s.portalRepo.WorkerAssignedToMove(ctx, tenantInfo, wrk.ID, moveID)
	if err != nil {
		return err
	}
	if !assigned {
		return errortypes.NewNotFoundError("Load not found")
	}

	_, err = s.moveService.RecordStopActual(ctx, &repositories.RecordStopActualRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		StopID:     stopID,
		Action:     action,
	})
	return err
}

func (s *Service) CreateMyLoadComment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	body string,
	actor *serviceports.RequestActor,
) (*PortalComment, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowLoadComments },
		"Your carrier has turned off load messaging in Dash — call your dispatcher instead.",
	); err != nil {
		return nil, err
	}

	assigned, err := s.portalRepo.WorkerAssignedToShipment(ctx, tenantInfo, wrk.ID, shipmentID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, errortypes.NewNotFoundError("Load not found")
	}

	entity := &shipment.ShipmentComment{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		ShipmentID:     shipmentID,
		Comment:        strings.TrimSpace(body),
		Type:           shipment.CommentTypeDriverUpdate,
		Visibility:     shipment.CommentVisibilityDriver,
		Priority:       shipment.CommentPriorityNormal,
		Source:         shipment.CommentSourceUser,
	}
	created, err := s.commentService.Create(ctx, entity, actor)
	if err != nil {
		return nil, err
	}

	view := &PortalComment{
		ID:         created.ID,
		Type:       string(created.Type),
		Priority:   string(created.Priority),
		Comment:    created.Comment,
		AuthorName: strings.TrimSpace(wrk.FirstName + " " + wrk.LastName),
		CreatedAt:  created.CreatedAt,
	}
	return view, nil
}

func (s *Service) MyPeriodSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalPeriodSummary, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	bounds := driversettlementservice.ResolveCurrentPeriod(control, timeutils.NowUnix())

	totals, err := s.payEventRepo.GetAccruedTotalsForWorker(
		ctx,
		repositories.GetAccruedTotalsForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   wrk.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &PortalPeriodSummary{
		PeriodStart:       bounds.PeriodStart,
		PeriodEnd:         bounds.PeriodEnd,
		PayDate:           bounds.PayDate,
		AccruedGrossMinor: totals.GrossAmountMinor,
		EventCount:        totals.EventCount,
	}, nil
}

func (s *Service) MyRecentPayEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*driversettlement.PayEvent, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	control, err := s.dashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !control.ShowLoadPay {
		return []*driversettlement.PayEvent{}, nil
	}
	result, err := s.payEventRepo.List(ctx, &repositories.ListPayEventsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: limit},
		},
		WorkerID: wrk.ID,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func portalVisibleStatuses() []driversettlement.Status {
	return []driversettlement.Status{
		driversettlement.StatusPosted,
		driversettlement.StatusPaid,
	}
}

func (s *Service) MySettlements(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit, offset int,
) (*pagination.ListResult[*driversettlement.Settlement], error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.settlementRepo.List(ctx, &repositories.ListDriverSettlementsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: limit, Offset: offset},
		},
		WorkerID: wrk.ID,
		Statuses: portalVisibleStatuses(),
	})
}

func (s *Service) MySettlement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) (*driversettlement.Settlement, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	settlement, err := s.settlementRepo.GetByID(ctx, repositories.GetDriverSettlementByIDRequest{
		ID:           settlementID,
		TenantInfo:   tenantInfo,
		IncludeLines: true,
	})
	if err != nil {
		return nil, err
	}
	if settlement.WorkerID != wrk.ID {
		return nil, errortypes.NewNotFoundError("Settlement not found")
	}
	visible := false
	for _, status := range portalVisibleStatuses() {
		if settlement.Status == status {
			visible = true
			break
		}
	}
	if !visible {
		return nil, errortypes.NewNotFoundError("Settlement not found")
	}
	return settlement, nil
}

func (s *Service) MyEscrow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalEscrowSummary, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	account, err := s.escrowRepo.GetActiveForWorker(
		ctx,
		repositories.GetActiveEscrowAccountForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   wrk.ID,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &PortalEscrowSummary{}, nil
		}
		return nil, err
	}
	transactions, err := s.escrowRepo.ListTransactions(
		ctx,
		repositories.GetEscrowAccountByIDRequest{
			ID:         account.ID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	return &PortalEscrowSummary{Account: account, Transactions: transactions}, nil
}

func (s *Service) MyAdvances(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*driverpay.PayAdvance, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.advanceRepo.ListOutstandingForWorker(
		ctx,
		repositories.ListOutstandingAdvancesForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   wrk.ID,
		},
	)
}
