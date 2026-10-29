package billingquerytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransfercriteria"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	paramMarkReady    = "markCompletedReadyToInvoice"
	maxTransferIssues = 3
)

type billingTransferCandidateLister interface {
	ListBillingTransferCandidates(
		ctx context.Context,
		req *shipmentbillingports.ListBillingTransferCandidatesRequest,
	) (*shipmentbillingports.BillingTransferCandidates, error)
}

type listBillingTransferCandidatesTool struct {
	shipments billingTransferCandidateLister
}

func NewListBillingTransferCandidatesTool(
	shipments shipmentserviceports.ShipmentBillingTransferService,
) agenttoolports.AgentQueryTool {
	return &listBillingTransferCandidatesTool{shipments: shipments}
}

func (t *listBillingTransferCandidatesTool) Name() string {
	return "billing_list_transfer_candidates"
}

func (t *listBillingTransferCandidatesTool) Description() string {
	return "List the delivered shipments not yet in the billing queue, oldest first: the same " +
		"list the transfer-to-billing dialog offers. Each row says what a transfer would do " +
		"with it (Transfer, MarkReadyAndTransfer, Refused with a failure code, or " +
		"ReturnToOperations), the documents it still lacks and its rate or validation " +
		"issues, decided by the checks the transfer itself makes. totals counts and sums " +
		"the charges of each outcome per currency, so quote them rather than adding rows " +
		"yourself. A question about what is waiting is answered from this list; propose a " +
		"transfer only when asked to send them. To transfer, propose one billing_transfer: " +
		"allTransferable with these " +
		"same filters when every row that can go should go, or shipmentIds for a hand-picked " +
		"set, rather than one call per shipment."
}

func (t *listBillingTransferCandidatesTool) ParamSchema() map[string]any {
	properties := billingtransfercriteria.Properties()
	properties[paramMarkReady] = querytool.BoolParam(
		"Decide as a transfer that first marks completed " +
			"shipments ready to invoice would. Defaults to false.",
	)

	return querytool.ObjectSchema(
		querytool.WithPaging(properties, listtool.DefaultLimit, listtool.MaxLimit),
	)
}

func (t *listBillingTransferCandidatesTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{Resource: permission.ResourceShipment})
}

type billingTransferCandidateRow struct {
	ID            string                 `json:"id"`
	ProNumber     string                 `json:"proNumber"`
	Status        string                 `json:"status"`
	Customer      string                 `json:"customer,omitempty"`
	TotalCharge   string                 `json:"totalCharge,omitempty"`
	DeliveredAt   querytool.OptionalDate `json:"deliveredAt"`
	WouldDo       string                 `json:"wouldDo"`
	CanTransfer   bool                   `json:"canTransfer"`
	FailureCode   string                 `json:"failureCode,omitempty"`
	MissingDocs   string                 `json:"missingDocuments,omitempty"`
	Issues        string                 `json:"issues,omitempty"`
	Reason        string                 `json:"reason,omitempty"`
	AutoApproves  bool                   `json:"autoApproves"`
	MultiplePayer bool                   `json:"splitBilled,omitempty"`
}

type billingTransferCandidatesResult struct {
	querytool.SearchOutcome

	TotalCandidates int `json:"totalCandidates"`
	PageTransfer    int `json:"pageTransfer"`
	PageRefused     int `json:"pageRefused"`
	PageReturned    int `json:"pageReturnedToOperations"`

	Totals candidateTotals `json:"totals"`
}

func (t *listBillingTransferCandidatesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	clk := querytool.ClockFor(params)
	selection, err := billingtransfercriteria.Read(params.Params, clk)
	if err != nil {
		return nil, err
	}
	window := querytool.ReadPage(params.Params, listtool.DefaultLimit, listtool.MaxLimit)

	criteria := filtercatalog.NewCriteria("shipments ready to transfer to billing").At(clk)
	selection.Describe(criteria)

	candidates, err := t.shipments.ListBillingTransferCandidates(
		ctx,
		&shipmentbillingports.ListBillingTransferCandidatesRequest{
			Filter: selection.QueryOptions(
				querytool.TenantOf(params),
				pagination.Info{Limit: window.Limit, Offset: window.Offset},
			),
			Status:                      selection.Status,
			MarkCompletedReadyToInvoice: querytool.OptionalBool(params.Params, paramMarkReady),
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
		case decision.Outcome == shipmentbillingports.BillingTransferOutcomeReturnToOperations:
			result.PageReturned++
		default:
			result.PageRefused++
		}
	}
	result.SearchOutcome = querytool.SearchResult(criteria, rows, len(rows)).
		Paged(window, candidates.HasMore)
	result.Totals = candidateTotalsOf(
		candidates.Decisions,
		window.Offset == 0 && !candidates.HasMore,
	)
	if selection.Status == shipmentvocab.StatusReadyToInvoice {
		if note := t.completedAlsoWaiting(ctx, params, selection); note != "" {
			result.Note = strings.TrimSpace(result.Note + " " + note)
		}
	}

	return result, nil
}

// completedAlsoWaiting counts what a ReadyToInvoice-only listing left out:
// delivered loads still Completed, which a transfer takes once it marks them
// ready. Asked what was ready to bill, gpt-6-luna filtered to ReadyToInvoice
// and reported one load while eighteen more were waiting.
func (t *listBillingTransferCandidatesTool) completedAlsoWaiting(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	selection billingtransfercriteria.Criteria,
) string {
	selection.Status = shipmentvocab.StatusCompleted
	completed, err := t.shipments.ListBillingTransferCandidates(
		ctx,
		&shipmentbillingports.ListBillingTransferCandidatesRequest{
			Filter: selection.QueryOptions(
				querytool.TenantOf(params),
				pagination.Info{Limit: 1},
			),
			Status:                      selection.Status,
			MarkCompletedReadyToInvoice: true,
		},
	)
	if err != nil || completed.TotalCount == 0 {
		return ""
	}

	return fmt.Sprintf("%d more delivered shipments are still Completed: a transfer with "+
		"markCompletedReadyToInvoice takes them too. Call again without status to list them, "+
		"and count them when asked what is ready to bill.", completed.TotalCount)
}

func candidateRow(
	decision *shipmentbillingports.BillingTransferDecision,
) billingTransferCandidateRow {
	row := billingTransferCandidateRow{
		ID:        decision.ShipmentID.String(),
		ProNumber: decision.ProNumber,
		Status:    string(decision.Status),
		Customer:  decision.CustomerName,
		DeliveredAt: querytool.ExpectedDate(
			typeutils.ValueOrZero(decision.DeliveredAt),
			"not delivered yet",
		),
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
