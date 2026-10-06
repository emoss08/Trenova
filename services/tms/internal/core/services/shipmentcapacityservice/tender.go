package shipmentcapacityservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const genericTenderFailure = "The shipment could not be tendered"

var (
	errAlreadyCovered = errors.New("every move on this shipment already has coverage")
	errNoPricedRate   = errors.New(
		"no carrier could be priced for this lane; tender it from the shipment with a rate",
	)
)

func (s *Service) TenderShipments(
	ctx context.Context,
	req *services.TenderShipmentsRequest,
) (*services.TenderShipmentsResult, error) {
	if len(req.Items) == 0 {
		return nil, errortypes.NewValidationError(
			"items",
			errortypes.ErrRequired,
			"Select a shipment to tender",
		)
	}
	if len(req.Items) > maxTenderItems {
		return nil, errortypes.NewValidationError(
			"items",
			errortypes.ErrInvalid,
			fmt.Sprintf("Tender at most %d shipments at once", maxTenderItems),
		)
	}

	result := &services.TenderShipmentsResult{
		Tendered: make([]*services.TenderShipmentSuccess, 0, len(req.Items)),
		Failed:   make([]*services.TenderShipmentFailure, 0),
	}
	for _, item := range req.Items {
		success, err := s.tenderOne(ctx, req.TenantInfo, item)
		if err != nil {
			message := failureMessage(err)
			if message == genericTenderFailure {
				s.l.Error("failed to tender shipment",
					zap.String("shipmentId", item.ShipmentID.String()),
					zap.Error(err),
				)
			}
			result.Failed = append(result.Failed, &services.TenderShipmentFailure{
				ShipmentID: item.ShipmentID,
				Message:    message,
			})
			continue
		}
		result.Tendered = append(result.Tendered, success)
	}

	return result, nil
}

func (s *Service) tenderOne(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item services.TenderShipmentItem,
) (*services.TenderShipmentSuccess, error) {
	sp, err := s.loadShipment(ctx, tenantInfo, item.ShipmentID)
	if err != nil {
		return nil, err
	}
	move := firstUncoveredMove(sp)
	if move == nil {
		return nil, errAlreadyCovered
	}

	if item.CarrierID.IsNotNil() {
		return s.tenderSpot(ctx, tenantInfo, sp.ID, move.ID, []pulid.ID{item.CarrierID})
	}

	created, err := s.tenderer.CreateWaterfall(ctx, &tenderservice.CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})
	if err == nil {
		return waterfallSuccess(sp.ID, created), nil
	}
	if !errors.Is(err, tenderservice.ErrNoRoutingGuideMatch) {
		return nil, err
	}

	return s.tenderSpot(ctx, tenantInfo, sp.ID, move.ID, nil)
}

func (s *Service) tenderSpot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID, moveID pulid.ID,
	carrierIDs []pulid.ID,
) (*services.TenderShipmentSuccess, error) {
	shopped, err := s.shopper.Shop(ctx, &ratequoteservice.ShopRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
		CarrierIDs: carrierIDs,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	option := firstPriced(shopped)
	if option == nil {
		return nil, errNoPricedRate
	}

	created, err := s.tenderer.CreateSpot(ctx, &tenderservice.CreateSpotTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
		Mode:           tender.ModeSpotBroadcast,
		Lines: []tenderservice.SpotTenderLine{{
			CarrierID:  option.CarrierID,
			RateMethod: shipment.CarrierRateMethodFlat,
			Rate:       option.Cost,
		}},
	})
	if err != nil {
		return nil, err
	}

	return &services.TenderShipmentSuccess{
		ShipmentID:  shipmentID,
		TenderID:    created.ID,
		CarrierID:   option.CarrierID,
		CarrierName: option.CarrierName,
	}, nil
}

func firstPriced(result *services.ShopResult) *services.ShopOption {
	if result == nil {
		return nil
	}
	for _, option := range result.Options {
		if option != nil && option.Priced() {
			return option
		}
	}

	return nil
}

func waterfallSuccess(
	shipmentID pulid.ID,
	created *tenderservice.CreateWaterfallResult,
) *services.TenderShipmentSuccess {
	success := &services.TenderShipmentSuccess{ShipmentID: shipmentID}
	if created == nil || created.Tender == nil {
		return success
	}
	success.TenderID = created.ID
	for _, offer := range created.Offers {
		if offer == nil {
			continue
		}
		success.CarrierID = offer.CarrierID
		if offer.Carrier != nil {
			success.CarrierName = offer.Carrier.Name
		}
		break
	}

	return success
}

func failureMessage(err error) string {
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return business.Message
	}
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		return multi.Error()
	}
	if errors.Is(err, errAlreadyCovered) || errors.Is(err, errNoPricedRate) {
		return err.Error()
	}
	if errortypes.IsNotFoundError(err) {
		return "Shipment not found"
	}

	return genericTenderFailure
}
