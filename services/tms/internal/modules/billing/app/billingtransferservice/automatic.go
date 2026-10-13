package billingtransferservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// automaticScanFactor is how many recently changed candidates are weighed
	// for each shipment a run may take, so a backlog of shipments that are not
	// yet billable cannot crowd out the ones that are.
	automaticScanFactor = 5
	maxAutomaticScan    = 2000
)

// AutomaticRunRequest is one organization's turn in the automatic transfer
// sweep.
type AutomaticRunRequest struct {
	Control      *tenant.BillingControl
	SystemUserID pulid.ID
	Now          int64
}

// StartAutomatic moves an organization's ready freight into billing when its
// billing control asks for that and its schedule says it is time. It starts
// nothing, and reports no error, when there is nothing to move, the previous
// automatic transfer is still working, or the interval has not passed.
//
// The run is an ordinary transfer recorded under the Automatic scope and
// requested by the system user, so it shows in the transfer report, can be
// stopped and retried like any other, and the unique index on a requester's
// active run keeps two automatic transfers from overlapping.
func (s *Service) StartAutomatic(
	ctx context.Context,
	req *AutomaticRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	control := req.Control
	if control == nil ||
		control.BillingQueueTransferMode != tenant.BillingQueueTransferModeAutomaticWhenReady ||
		s.shipments == nil {
		return nil, nil //nolint:nilnil // automatic transfer is off for this organization
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  control.OrganizationID,
		BuID:   control.BusinessUnitID,
		UserID: req.SystemUserID,
	}

	due, err := s.automaticRunDue(ctx, tenantInfo, control.BillingQueueTransferSchedule, req.Now)
	if err != nil || !due {
		return nil, err
	}

	active, err := s.runRepo.GetActive(
		ctx,
		&billingtransferports.GetActiveBillingTransferRunRequest{
			TenantInfo:    tenantInfo,
			RequestedByID: req.SystemUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, nil //nolint:nilnil // a transfer for this shipment is already running
	}

	limit := AutomaticBatchSize(control)
	shipmentIDs, err := s.shipments.ListAutoTransferCandidateIDs(
		ctx,
		&shipmentports.ListAutoTransferCandidateIDsRequest{
			TenantInfo: tenantInfo,
			Limit:      limit,
			ScanLimit:  min(limit*automaticScanFactor, maxAutomaticScan),
		},
	)
	if err != nil || len(shipmentIDs) == 0 {
		return nil, err
	}

	run, err := s.start(ctx, &StartRunRequest{
		TenantInfo:                  tenantInfo,
		Scope:                       billingtransfer.RunScopeAutomatic,
		ShipmentIDs:                 shipmentIDs,
		BillType:                    billingqueue.BillTypeInvoice,
		MarkCompletedReadyToInvoice: true,
	})
	if errortypes.IsConflictError(err) {
		return nil, nil //nolint:nilnil // another transfer claimed the shipment first
	}

	return run, err
}

// AutomaticBatchSize is how many shipments one automatic transfer may take: the
// transfer batch size, and no more invoice drafts than the auto-invoice batch
// size allows where transferring creates them.
func AutomaticBatchSize(control *tenant.BillingControl) int {
	limit := control.BillingQueueTransferBatchSize
	if control.InvoiceDraftCreationMode == tenant.InvoiceDraftCreationModeAutomaticWhenTransferred &&
		control.AutoInvoiceBatchSize > 0 {
		limit = min(limit, control.AutoInvoiceBatchSize)
	}

	return max(min(limit, billingtransfer.MaxRunShipments), 1)
}

func (s *Service) automaticRunDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	schedule tenant.TransferSchedule,
	now int64,
) (bool, error) {
	interval := automaticInterval(schedule)
	if interval == 0 {
		return true, nil
	}

	last, err := s.runRepo.LastQueuedAt(
		ctx,
		&billingtransferports.LastBillingTransferRunQueuedAtRequest{
			TenantInfo: tenantInfo,
			Scope:      billingtransfer.RunScopeAutomatic,
		},
	)
	if err != nil {
		return false, err
	}

	return now-last >= interval, nil
}

func automaticInterval(schedule tenant.TransferSchedule) int64 {
	switch schedule {
	case tenant.TransferScheduleHourly:
		return int64(time.Hour.Seconds())
	case tenant.TransferScheduleDaily:
		return int64((24 * time.Hour).Seconds())
	case tenant.TransferScheduleWeekly:
		return int64((7 * 24 * time.Hour).Seconds())
	case tenant.TransferScheduleContinuous:
		return 0
	default:
		return 0
	}
}
