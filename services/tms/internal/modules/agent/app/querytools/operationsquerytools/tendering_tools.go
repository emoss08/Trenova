package operationsquerytools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShipmentMoveID = "shipmentMoveId"
)

func TenderingToolProviders() []any {
	return []any{
		newListRateConfirmationsTool,
	}
}

// rateConfirmationLister reads a move's rate confirmation revisions, newest
// first.
type rateConfirmationLister interface {
	ListByMoveID(
		ctx context.Context,
		req *rateconfirmationports.ListRateConfirmationsByMoveRequest,
	) ([]*rateconfirmation.RateConfirmation, error)
}

// listRateConfirmationsTool is a move's rate confirmation revisions, which
// is what hands out the ids the rate confirmation tools take.
type listRateConfirmationsTool struct {
	rateCons rateConfirmationLister
}

func newListRateConfirmationsTool(
	rateCons rateconfirmationports.RateConfirmationRepository,
) agenttoolports.AgentQueryTool {
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
	return querytool.ObjectSchema(map[string]any{
		paramShipmentMoveID: agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
			"The move, from shipment_get (its moves) or "+
				"dispatch_get_board."),
	}, paramShipmentMoveID)
}

func (t *listRateConfirmationsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceRateConfirmation,
		Rationale: "Reads a move's rate confirmation revisions; the name a carrier typed " +
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
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	moveID, err := querytool.RequirePulid(params.Params, paramShipmentMoveID)
	if err != nil {
		return nil, err
	}

	entities, err := t.rateCons.ListByMoveID(
		ctx,
		&rateconfirmationports.ListRateConfirmationsByMoveRequest{
			TenantInfo:     querytool.TenantOf(params),
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

	criteria := filtercatalog.NewCriteria("rate confirmations").At(querytool.ClockFor(params))
	criteria.Field(paramShipmentMoveID, moveID.String())

	return querytool.SearchResult(criteria, rows, len(rows)), nil
}

func idString(id *pulid.ID) string {
	if id == nil || id.IsNil() {
		return ""
	}

	return id.String()
}
