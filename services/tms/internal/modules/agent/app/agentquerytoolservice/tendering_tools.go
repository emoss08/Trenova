package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShipmentID     = "shipmentId"
	paramShipmentMoveID = "shipmentMoveId"
)

func tenderingToolProviders() []any {
	return []any{
		newListShipmentTendersTool,
		newListRateConfirmationsTool,
	}
}

// tenderLister reads a shipment's tenders with their offers in rank order.
type tenderLister interface {
	ListByShipment(
		ctx context.Context,
		req tenderports.ListTendersByShipmentRequest,
	) ([]*tender.Tender, error)
}

// rateConfirmationLister reads a move's rate confirmation revisions, newest
// first.
type rateConfirmationLister interface {
	ListByMoveID(
		ctx context.Context,
		req *rateconfirmationports.ListRateConfirmationsByMoveRequest,
	) ([]*rateconfirmation.RateConfirmation, error)
}

// listShipmentTendersTool is the tenders panel of a shipment: every tender
// with the offer each carrier holds. It is what hands out the tender and
// offer ids dispatch_cancel_tender and dispatch_record_tender_response take.
type listShipmentTendersTool struct {
	tenders tenderLister
	access  fieldAccess
}

func newListShipmentTendersTool(
	tenders tenderports.TenderRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listShipmentTendersTool{tenders: tenders, access: newFieldAccess(permissions)}
}

func (t *listShipmentTendersTool) Name() string { return "dispatch_list_tenders" }

func (t *listShipmentTendersTool) Description() string {
	return "List a shipment's tenders, newest first, with every offer in rank order. Each " +
		"tender shows its mode and status; each offer the carrier, the rate, how it was sent, " +
		"its status, any counter the carrier made and when it lapses or was answered. Use " +
		"it to see who holds an offer before recording a carrier's answer, a decision on its " +
		"counter or the carrier backing out with dispatch_record_tender_response, or to find a live " +
		"tender to withdraw with dispatch_cancel_tender. Narrow to one move with shipmentMoveId."
}

func (t *listShipmentTendersTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
			"The shipment, from shipment_list, shipment_get or "+
				"the page you are on."),
		paramShipmentMoveID: agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
			"Only this move's tenders, by id from shipment_get "+
				"(its moves) or dispatch_get_board."),
	}, paramShipmentID)
}

func (t *listShipmentTendersTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceTender,
		rationale: "Reads a shipment's tenders and their offers; a carrier's own words, " +
			"such as a decline reason, are left out, so nothing written outside the " +
			"organization is read.",
	})
}

type tenderOfferRow struct {
	OfferID        string `json:"offerId"`
	Rank           int16  `json:"rank"`
	CarrierID      string `json:"carrierId"`
	CarrierName    string `json:"carrierName,omitempty"`
	RateMethod     string `json:"rateMethod"`
	Rate           string `json:"rate,omitempty"`
	Channel        string `json:"channel"`
	Status         string `json:"status"`
	SentAt         *int64 `json:"sentAt,omitempty"`
	ExpiresAt      *int64 `json:"expiresAt,omitempty"`
	RespondedAt    *int64 `json:"respondedAt,omitempty"`
	ResponseSource string `json:"responseSource,omitempty"`
	// Answerable is whether dispatch_record_tender_response can still record an
	// answer on this offer: the carrier's own, or a decision on its counter.
	Answerable bool `json:"answerable"`
	// CounterRate is the rate the carrier countered with, when it made one.
	CounterRate string `json:"counterRate,omitempty"`
	// CounterNeedsApproval is whether taking the counter needs the tender
	// approve permission, because it is below the margin floor.
	CounterNeedsApproval bool `json:"counterNeedsApproval,omitempty"`
	// BackOutAllowed is whether dispatch_record_tender_response can record the
	// carrier backing out of this offer, which it accepted.
	BackOutAllowed bool `json:"backOutAllowed,omitempty"`
}

type tenderRow struct {
	TenderID       string           `json:"tenderId"`
	ShipmentMoveID string           `json:"shipmentMoveId"`
	Mode           string           `json:"mode"`
	Status         string           `json:"status"`
	Live           bool             `json:"live"`
	AcceptedOffer  string           `json:"acceptedOfferId,omitempty"`
	Bounces        int16            `json:"bounces,omitempty"`
	Escalation     int16            `json:"escalationRound,omitempty"`
	CreatedAt      int64            `json:"createdAt"`
	Offers         []tenderOfferRow `json:"offers"`
}

func (t *listShipmentTendersTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}
	moveID, err := optionalID(params.Params, paramShipmentMoveID)
	if err != nil {
		return nil, err
	}

	tenders, err := t.tenders.ListByShipment(ctx, tenderports.ListTendersByShipmentRequest{
		TenantInfo: tenantOf(params),
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, err
	}

	criteria := filtercatalog.NewCriteria("tenders").At(clockFor(params))
	criteria.Field(paramShipmentID, shipmentID.String())
	if moveID.IsNotNil() {
		criteria.Field(paramShipmentMoveID, moveID.String())
	}

	gate := t.access.gate(ctx, params, permission.ResourceTender)
	rows := make([]tenderRow, 0, len(tenders))
	for _, entity := range tenders {
		if entity == nil || (moveID.IsNotNil() && entity.ShipmentMoveID != moveID) {
			continue
		}
		rows = append(rows, tenderRowFrom(entity, gate))
	}

	found := searchResult(criteria, rows, len(rows))

	return gatedResult(&found, gate), nil
}

func tenderRowFrom(entity *tender.Tender, gate *fieldGate) tenderRow {
	row := tenderRow{
		TenderID:       entity.ID.String(),
		ShipmentMoveID: entity.ShipmentMoveID.String(),
		Mode:           entity.Mode.String(),
		Status:         entity.Status.String(),
		Live:           entity.IsLive(),
		AcceptedOffer:  idString(entity.AcceptedOfferID),
		Bounces:        entity.BounceCount,
		Escalation:     entity.EscalationRound,
		CreatedAt:      entity.CreatedAt,
		Offers:         make([]tenderOfferRow, 0, len(entity.Offers)),
	}
	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		offerRow := tenderOfferRow{
			OfferID:        offer.ID.String(),
			Rank:           offer.Rank,
			CarrierID:      offer.CarrierID.String(),
			RateMethod:     string(offer.RateMethod),
			Channel:        string(offer.Channel),
			Status:         string(offer.Status),
			SentAt:         offer.SentAt,
			ExpiresAt:      offer.ExpiresAt,
			RespondedAt:    offer.RespondedAt,
			ResponseSource: string(offer.ResponseSource),
			Answerable: entity.Status == tender.StatusActive &&
				offer.Status.IsAwaitingAnswer(),
			CounterNeedsApproval: offer.Status == tender.OfferStatusCountered &&
				offer.CounterNeedsApproval,
		}
		if accepted, ok := entity.CanBounce(); ok && accepted.ID == offer.ID {
			offerRow.BackOutAllowed = true
		}
		if offer.Carrier != nil {
			offerRow.CarrierName = strings.TrimSpace(offer.Carrier.Name)
		}
		if gate.show("rate", "offer rate") {
			offerRow.Rate = decimalPointerText(&offer.Rate)
			if offer.CounterRate.Valid {
				offerRow.CounterRate = decimalPointerText(&offer.CounterRate.Decimal)
			}
		}
		row.Offers = append(row.Offers, offerRow)
	}

	return row
}

// listRateConfirmationsTool is a move's rate confirmation revisions, which
// is what hands out the ids the rate confirmation tools take.
type listRateConfirmationsTool struct {
	rateCons rateConfirmationLister
}

func newListRateConfirmationsTool(
	rateCons rateconfirmationports.RateConfirmationRepository,
) agentports.AgentQueryTool {
	return &listRateConfirmationsTool{rateCons: rateCons}
}

func (t *listRateConfirmationsTool) Name() string { return "rate_list_confirmations" }

func (t *listRateConfirmationsTool) Description() string {
	return "List a move's rate confirmations, newest revision first. Each shows the carrier, " +
		"the revision, its status (Generated, Sent, Confirmed or Voided), who it was sent to " +
		"and when, and whether and how the carrier confirmed it. At most one revision " +
		"stands at a time; a new one voids the last. Use it before sending, voiding or " +
		"recording a confirmation, and to find the revision rate_generate_confirmation made."
}

func (t *listRateConfirmationsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramShipmentMoveID: agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
			"The move, from shipment_get (its moves) or "+
				"dispatch_get_board."),
	}, paramShipmentMoveID)
}

func (t *listRateConfirmationsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceRateConfirmation,
		rationale: "Reads a move's rate confirmation revisions; the name a carrier typed " +
			"when signing is left out, so nothing written outside the organization is read.",
	})
}

type rateConfirmationRow struct {
	RateConfirmationID string `json:"rateConfirmationId"`
	Revision           int64  `json:"revision"`
	Status             string `json:"status"`
	CarrierID          string `json:"carrierId"`
	CarrierName        string `json:"carrierName,omitempty"`
	CarrierAssignment  string `json:"carrierAssignmentId"`
	GeneratedVia       string `json:"generatedVia"`
	SentAt             *int64 `json:"sentAt,omitempty"`
	SentTo             string `json:"sentTo,omitempty"`
	ConfirmedAt        *int64 `json:"confirmedAt,omitempty"`
	ConfirmedVia       string `json:"confirmedVia,omitempty"`
	VoidedAt           *int64 `json:"voidedAt,omitempty"`
	CreatedAt          int64  `json:"createdAt"`
}

func (t *listRateConfirmationsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	moveID, err := requirePulid(params.Params, paramShipmentMoveID)
	if err != nil {
		return nil, err
	}

	entities, err := t.rateCons.ListByMoveID(
		ctx,
		&rateconfirmationports.ListRateConfirmationsByMoveRequest{
			TenantInfo:     tenantOf(params),
			ShipmentMoveID: moveID,
		},
	)
	if err != nil {
		return nil, err
	}

	rows := make([]rateConfirmationRow, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		row := rateConfirmationRow{
			RateConfirmationID: entity.ID.String(),
			Revision:           entity.Revision,
			Status:             entity.Status.String(),
			CarrierID:          entity.CarrierID.String(),
			CarrierAssignment:  entity.CarrierAssignmentID.String(),
			GeneratedVia:       entity.GeneratedVia.String(),
			SentAt:             entity.SentAt,
			SentTo:             entity.SentToEmails,
			ConfirmedAt:        entity.ConfirmedAt,
			ConfirmedVia:       string(entity.ConfirmedVia),
			VoidedAt:           entity.VoidedAt,
			CreatedAt:          entity.CreatedAt,
		}
		if entity.Carrier != nil {
			row.CarrierName = strings.TrimSpace(entity.Carrier.Name)
		}
		rows = append(rows, row)
	}

	criteria := filtercatalog.NewCriteria("rate confirmations").At(clockFor(params))
	criteria.Field(paramShipmentMoveID, moveID.String())

	return searchResult(criteria, rows, len(rows)), nil
}

func idString(id *pulid.ID) string {
	if id == nil || id.IsNil() {
		return ""
	}

	return id.String()
}
