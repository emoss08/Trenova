package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

// LeaveReader is the slice of the leave service the portal needs. A driver
// reads their own standing and nothing else, so nothing that writes is here.
type LeaveReader interface {
	Entitlement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		asOf int64,
	) (worker.LeaveEntitlement, error)
	ListCases(
		ctx context.Context,
		req *repositories.ListLeaveCasesRequest,
	) ([]*worker.WorkerLeaveCase, error)
	CountCases(
		ctx context.Context,
		req *repositories.ListLeaveCasesRequest,
	) (int, error)
}

// PortalLeaveCase is a driver's own view of one leave case: what was decided,
// what is owed, and how much of it they have taken. The office's notes and the
// decision-maker stay in the office.
type PortalLeaveCase struct {
	ID                  string                          `json:"id"`
	LeaveType           worker.LeaveType                `json:"leaveType"`
	Status              worker.LeaveCaseStatus          `json:"status"`
	Frequency           worker.LeaveFrequency           `json:"frequency"`
	FMLADesignated      bool                            `json:"fmlaDesignated"`
	Reason              string                          `json:"reason"`
	StartsAt            int64                           `json:"startsAt"`
	EndsAt              *int64                          `json:"endsAt"`
	DecidedAt           *int64                          `json:"decidedAt"`
	CertificationStatus worker.LeaveCertificationStatus `json:"certificationStatus"`
	CertificationDueAt  *int64                          `json:"certificationDueAt"`
	CertificationLate   bool                            `json:"certificationLate"`
	// HoursUsed is what has been recorded against this case, whether or not it
	// was charged to the entitlement. A driver counts the days they were away.
	HoursUsed string `json:"hoursUsed"`
	// HoursCharged is the part of that which drew the entitlement down.
	HoursCharged string `json:"hoursCharged"`
}

// PortalLeave is the driver's leave standing: the balance and the cases it was
// drawn down by, which are only meaningful beside each other.
type PortalLeave struct {
	Entitlement worker.LeaveEntitlement `json:"entitlement"`
	Cases       []*PortalLeaveCase      `json:"cases"`
}

// MyLeave reads the signed-in driver's own leave standing. The worker is
// resolved from the session, never accepted as an argument.
func (s *Service) MyLeave(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalLeave, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if s.leave == nil {
		return &PortalLeave{Cases: []*PortalLeaveCase{}}, nil
	}

	entitlement, err := s.leave.Entitlement(ctx, tenantInfo, wrk.ID, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	cases, err := s.leave.ListCases(ctx, &repositories.ListLeaveCasesRequest{
		TenantInfo:     tenantInfo,
		WorkerID:       wrk.ID,
		IncludeEntries: true,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	views := make([]*PortalLeaveCase, 0, len(cases))
	for _, entity := range cases {
		if entity == nil {
			continue
		}
		views = append(views, portalLeaveCase(entity, now))
	}

	return &PortalLeave{Entitlement: entitlement, Cases: views}, nil
}

func portalLeaveCase(entity *worker.WorkerLeaveCase, now int64) *PortalLeaveCase {
	used := decimal.Zero
	charged := decimal.Zero
	for _, entry := range entity.Entries {
		if entry == nil {
			continue
		}
		used = used.Add(entry.Hours)
		if entry.CountsAgainstEntitlement {
			charged = charged.Add(entry.Hours)
		}
	}

	return &PortalLeaveCase{
		ID:                  entity.ID.String(),
		LeaveType:           entity.LeaveType,
		Status:              entity.Status,
		Frequency:           entity.Frequency,
		FMLADesignated:      entity.FMLADesignated,
		Reason:              entity.Reason,
		StartsAt:            entity.StartsAt,
		EndsAt:              entity.EndsAt,
		DecidedAt:           entity.DecidedAt,
		CertificationStatus: entity.CertificationStatus,
		CertificationDueAt:  entity.CertificationDueAt,
		CertificationLate:   entity.CertificationLate(now),
		HoursUsed:           used.String(),
		HoursCharged:        charged.String(),
	}
}

// leaveVisible says whether the portal should offer a leave section at all. A
// driver who has never taken leave has no balance worth showing, and an empty
// FMLA card reads as an entitlement they are owed rather than one they have
// not started.
func (s *Service) leaveVisible(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (bool, error) {
	if s.leave == nil {
		return false, nil
	}
	count, err := s.leave.CountCases(ctx, &repositories.ListLeaveCasesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
