package detentionreview

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncharge"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var _ detentionports.DetentionChargeSettler = (*Service)(nil)

type decision struct {
	change   *OccurrenceChange
	userID   pulid.ID
	headline string
	detail   string
}

func (s *Service) settleShipmentCharges(ctx context.Context, d *decision) error {
	saved := d.change.After
	tenantInfo := pagination.TenantInfo{OrgID: saved.OrganizationID, BuID: saved.BusinessUnitID}

	entity, err := s.shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         saved.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	_, err = s.settle(ctx, &settleParams{
		tenantInfo: tenantInfo,
		entity:     entity,
		decided:    saved,
		persist:    true,
		comment:    d,
	})

	return err
}

func (s *Service) SettleShipmentCharges(
	ctx context.Context,
	req *detentionports.SettleDetentionChargesRequest,
) ([]detentioncharge.Change, error) {
	if req == nil || req.Shipment == nil || req.Shipment.ID.IsNil() {
		return nil, nil
	}

	return s.settle(ctx, &settleParams{
		tenantInfo: req.TenantInfo,
		entity:     req.Shipment,
		persist:    req.Persist,
		comment: &decision{
			userID:   req.ActorUserID,
			headline: "Brought in line with its detention decisions before invoicing",
		},
	})
}

type settleParams struct {
	tenantInfo pagination.TenantInfo
	entity     *shipment.Shipment
	decided    *detention.DetentionOccurrence
	persist    bool
	comment    *decision
}

func (s *Service) settle(ctx context.Context, p *settleParams) ([]detentioncharge.Change, error) {
	occurrences, err := s.occurrenceRepo.GetByShipment(
		ctx,
		&detentionports.GetOccurrencesByShipmentRequest{
			ShipmentID: p.entity.ID,
			TenantInfo: p.tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if p.decided != nil {
		occurrences = withDecided(occurrences, p.decided)
	}

	changes := detentioncharge.Settle(p.entity, occurrences)
	if len(changes) == 0 {
		return nil, nil
	}

	if err = rebalanceCharges(ctx, p.entity); err != nil {
		return nil, err
	}
	if !p.persist {
		return changes, nil
	}

	if _, err = s.shipments.UpdateDerivedState(ctx, p.entity); err != nil {
		return nil, err
	}

	if err = s.orders.RecomputeForShipment(ctx, p.tenantInfo, p.entity.ID); err != nil {
		return nil, err
	}

	s.commentChargeChanges(ctx, p.entity, p.comment, changes, currencyOf(occurrences))

	return changes, nil
}

func currencyOf(occurrences []*detention.DetentionOccurrence) string {
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.Currency != "" {
			return occurrence.Currency
		}
	}

	return ""
}

func withDecided(
	occurrences []*detention.DetentionOccurrence,
	decided *detention.DetentionOccurrence,
) []*detention.DetentionOccurrence {
	out := make([]*detention.DetentionOccurrence, 0, len(occurrences)+1)
	found := false
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.ID == decided.ID {
			out = append(out, decided)
			found = true
			continue
		}
		out = append(out, occurrence)
	}
	if !found {
		out = append(out, decided)
	}

	return out
}

func rebalanceCharges(ctx context.Context, entity *shipment.Shipment) error {
	present := make(map[pulid.ID]struct{}, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		if charge != nil {
			present[charge.ID] = struct{}{}
		}
	}

	allocations := make([]*shipment.ChargeAllocation, 0, len(entity.ChargeAllocations))
	for _, allocation := range entity.ChargeAllocations {
		if allocation == nil {
			continue
		}
		if allocation.AdditionalChargeID != nil {
			if _, ok := present[*allocation.AdditionalChargeID]; !ok {
				continue
			}
		}
		allocations = append(allocations, allocation)
	}
	entity.ChargeAllocations = allocations

	freight := entity.FreightChargeAmount.Decimal
	totals, err := chargeallocation.AdditionalChargeTotals(ctx, entity.AdditionalCharges, freight)
	if err != nil {
		return err
	}
	entity.OtherChargeAmount = decimal.NewNullDecimal(totals.Sum)
	entity.TotalChargeAmount = decimal.NewNullDecimal(freight.Add(totals.Sum))

	rebalance, err := chargeallocation.RebalanceAmountSplits(ctx, entity, entity.ChargeAllocations)
	if err != nil {
		return err
	}
	rebalance.ConvertToPercent()

	return nil
}

func (s *Service) commentChargeChanges(
	ctx context.Context,
	entity *shipment.Shipment,
	d *decision,
	changes []detentioncharge.Change,
	currency string,
) {
	for _, change := range changes {
		var b strings.Builder
		if change.Removed() {
			fmt.Fprintf(&b, "Detention charge removed — %s %s",
				change.Before.StringFixed(2), currency)
		} else {
			fmt.Fprintf(&b, "Detention charge lowered from %s %s to %s %s",
				change.Before.StringFixed(2), currency, change.After.StringFixed(2), currency)
		}
		b.WriteString("\n\n")
		b.WriteString(d.headline)
		if detail := strings.TrimSpace(d.detail); detail != "" {
			b.WriteString(": ")
			b.WriteString(detail)
		}

		comment := &shipment.ShipmentComment{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			ShipmentID:     entity.ID,
			UserID:         d.userID,
			Comment:        b.String(),
			Type:           shipmentvocab.CommentTypeBilling,
			Visibility:     shipmentvocab.CommentVisibilityInternal,
			Priority:       shipmentvocab.CommentPriorityNormal,
			Source:         shipmentvocab.CommentSourceSystem,
			Metadata: map[string]any{
				"event":    "DetentionChargeSettled",
				"before":   change.Before.StringFixed(2),
				"after":    change.After.StringFixed(2),
				"currency": currency,
			},
		}
		if _, err := s.comments.Create(ctx, comment); err != nil {
			s.l.Warn("failed to comment a settled detention charge on its shipment")
		}
	}
}
