package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ListAutoTransferCandidateIDs finds the shipments an automatic transfer may
// move into billing now: ready to invoice already, or completed and eligible to
// be marked ready without a person, where the payer's transfer mode is
// automatic and nothing in billing policy would refuse the transfer.
//
// Shipments that fail those checks are left out rather than attempted. A
// manual transfer reports why a shipment could not go; an automatic one runs
// every few minutes, and reporting the same unbillable shipment each time would
// bury the failures worth reading.
func (s *service) ListAutoTransferCandidateIDs(
	ctx context.Context,
	req *shipmentports.ListAutoTransferCandidateIDsRequest,
) ([]pulid.ID, error) {
	if req.Limit <= 0 {
		return []pulid.ID{}, nil
	}

	scanned, err := s.repo.ListBillingTransferCandidateIDs(
		ctx,
		&shipmentports.ListBillingTransferCandidateIDsQuery{
			Filter:      &pagination.QueryOptions{TenantInfo: req.TenantInfo},
			Limit:       max(req.ScanLimit, req.Limit),
			RecentFirst: true,
		},
	)
	if err != nil {
		return nil, err
	}

	cache := new(billingReadinessCache)
	eligible := make([]pulid.ID, 0, min(req.Limit, len(scanned.IDs)))
	for _, id := range scanned.IDs {
		if len(eligible) == req.Limit {
			break
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		ok, evalErr := s.autoTransferEligible(ctx, req.TenantInfo, id, cache)
		if evalErr != nil {
			s.l.Warn("skipped a shipment the automatic transfer could not evaluate",
				zap.String("shipmentID", id.String()), zap.Error(evalErr))
			continue
		}
		if ok {
			eligible = append(eligible, id)
		}
	}

	return eligible, nil
}

func (s *service) autoTransferEligible(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	cache *billingReadinessCache,
) (bool, error) {
	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if errortypes.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, _, gateErr := transferGate(entity, true); gateErr != nil {
		return false, nil //nolint:nilerr // a shipment the gate refuses is not ready to transfer
	}

	readiness, err := s.evaluateBillingReadinessCached(ctx, entity, cache, true)
	if err != nil {
		return false, err
	}

	return readinessAllowsAutoTransfer(entity.Status, readiness), nil
}

func readinessAllowsAutoTransfer(
	status shipment.Status,
	readiness *shipmentports.ShipmentBillingReadiness,
) bool {
	if readiness == nil ||
		readiness.Policy.BillingQueueTransferMode != tenant.BillingQueueTransferModeAutomaticWhenReady {
		return false
	}
	if _, violation := billingTransferPolicyViolation(readiness); violation != nil {
		return false
	}

	switch status { //nolint:exhaustive // only these two may be transferred
	case shipment.StatusReadyToInvoice:
		return true
	case shipment.StatusCompleted:
		return readiness.ShouldAutoMarkReadyToInvoice
	default:
		return false
	}
}
