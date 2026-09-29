package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransfercriteria"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	paramMarkReady    = "markCompletedReadyToInvoice"
	maxTransferIssues = 3
)

type billingTransferCandidateLister interface {
	ListBillingTransferCandidates(
		ctx context.Context,
		req *serviceports.ListBillingTransferCandidatesRequest,
	) (*serviceports.BillingTransferCandidates, error)
}

type listBillingTransferCandidatesTool struct {
	shipments billingTransferCandidateLister
}

func newListBillingTransferCandidatesTool(
	shipments serviceports.ShipmentService,
) serviceports.AgentQueryTool {
	return &listBillingTransferCandidatesTool{shipments: shipments}
}

func (t *listBillingTransferCandidatesTool) Name() string {
	return "list_billing_transfer_candidates"
}

func (t *listBillingTransferCandidatesTool) Description() string {
	return "List the delivered shipments not yet in the billing queue, oldest first: the same " +
		"list the transfer-to-billing dialog offers. Each row says what a transfer would do " +
		"with it (Transfer, MarkReadyAndTransfer, Refused with a failure code, or " +
		"ReturnToOperations), the documents it still lacks and its rate or validation " +
		"issues, decided by the checks the transfer itself makes. totals counts and sums " +
		"the charges of each outcome per currency, so quote them rather than adding rows " +
		"yourself. To transfer, propose one transfer_to_billing: allTransferable with these " +
		"same filters when every row that can go should go, or shipmentIds for a hand-picked " +
		"set, rather than one call per shipment."
}

func (t *listBillingTransferCandidatesTool) ParamSchema() map[string]any {
	properties := billingtransfercriteria.Properties()
	properties[paramMarkReady] = boolParam("Decide as a transfer that first marks completed " +
		"shipments ready to invoice would. Defaults to false.")

	return objectSchema(withPaging(properties, defaultListLimit, maxListLimit))
}

func (t *listBillingTransferCandidatesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceShipment})
}

type billingTransferCandidateRow struct {
	ID            string       `json:"id"`
	ProNumber     string       `json:"proNumber"`
	Status        string       `json:"status"`
	Customer      string       `json:"customer,omitempty"`
	TotalCharge   string       `json:"totalCharge,omitempty"`
	DeliveredAt   optionalDate `json:"deliveredAt"`
	WouldDo       string       `json:"wouldDo"`
	CanTransfer   bool         `json:"canTransfer"`
	FailureCode   string       `json:"failureCode,omitempty"`
	MissingDocs   string       `json:"missingDocuments,omitempty"`
	Issues        string       `json:"issues,omitempty"`
	Reason        string       `json:"reason,omitempty"`
	AutoApproves  bool         `json:"autoApproves"`
	MultiplePayer bool         `json:"splitBilled,omitempty"`
}

type billingTransferCandidatesResult struct {
	searchOutcome

	TotalCandidates int `json:"totalCandidates"`
	PageTransfer    int `json:"pageTransfer"`
	PageRefused     int `json:"pageRefused"`
	PageReturned    int `json:"pageReturnedToOperations"`

	Totals candidateTotals `json:"totals"`
}

func (t *listBillingTransferCandidatesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	clk := clockFor(params)
	selection, err := billingtransfercriteria.Read(params.Params, clk)
	if err != nil {
		return nil, err
	}
	window := readPage(params.Params, defaultListLimit, maxListLimit)

	criteria := filtercatalog.NewCriteria("shipments ready to transfer to billing").At(clk)
	selection.Describe(criteria)

	candidates, err := t.shipments.ListBillingTransferCandidates(
		ctx,
		&serviceports.ListBillingTransferCandidatesRequest{
			Filter: selection.QueryOptions(
				tenantOf(params),
				pagination.Info{Limit: window.limit, Offset: window.offset},
			),
			Status:                      selection.Status,
			MarkCompletedReadyToInvoice: optionalBool(params.Params, paramMarkReady),
		},
	)
	if err != nil {
		return nil, err
	}

	result := billingTransferCandidatesResult{TotalCandidates: candidates.TotalCount}
	rows := make([]billingTransferCandidateRow, 0, len(candidates.Decisions))
	for idx := range candidates.Decisions {
		decision := &candidates.Decisions[idx]
		rows = append(rows, candidateRow(decision))
		switch {
		case decision.Outcome.Transfers():
			result.PageTransfer++
		case decision.Outcome == serviceports.BillingTransferOutcomeReturnToOperations:
			result.PageReturned++
		default:
			result.PageRefused++
		}
	}
	result.searchOutcome = searchResult(criteria, rows, len(rows)).paged(window, candidates.HasMore)
	result.Totals = candidateTotalsOf(
		candidates.Decisions,
		window.offset == 0 && !candidates.HasMore,
	)

	return result, nil
}

func candidateRow(decision *serviceports.BillingTransferDecision) billingTransferCandidateRow {
	row := billingTransferCandidateRow{
		ID:            decision.ShipmentID.String(),
		ProNumber:     decision.ProNumber,
		Status:        string(decision.Status),
		Customer:      decision.CustomerName,
		DeliveredAt:   expectedDate(derefInt64(decision.DeliveredAt), "not delivered yet"),
		WouldDo:       string(decision.Outcome),
		CanTransfer:   decision.Outcome.Transfers(),
		FailureCode:   string(decision.FailureCode),
		AutoApproves:  decision.AutoApprove,
		MultiplePayer: decision.PayerCount > 1,
	}
	if decision.TotalCharge.Valid {
		row.TotalCharge = decision.TotalCharge.Decimal.StringFixed(2)
	}
	if !row.CanTransfer {
		row.Reason = decision.Reason
	}

	missing := make([]string, 0, len(decision.MissingRequirements))
	for _, requirement := range decision.MissingRequirements {
		missing = append(missing, requirement.DocumentTypeName)
	}
	row.MissingDocs = strings.Join(missing, ", ")

	issues := make([]string, 0, maxTransferIssues)
	for _, failure := range decision.ValidationFailures {
		if len(issues) == maxTransferIssues {
			break
		}
		issues = append(issues, failure.Message)
	}
	row.Issues = strings.Join(issues, "; ")

	return row
}
