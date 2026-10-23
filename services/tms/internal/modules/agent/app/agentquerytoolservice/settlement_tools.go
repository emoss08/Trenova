package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	carrierInvoiceEntity = "edi_carrier_invoice"
)

var invoiceMatchStatuses = agenttoolschema.Source("carrierSettlement.invoiceMatchStatus", []string{
	string(carriersettlement.InvoiceMatchStatusSuggested),
	string(carriersettlement.InvoiceMatchStatusMatched),
	string(carriersettlement.InvoiceMatchStatusVariance),
	string(carriersettlement.InvoiceMatchStatusResolved),
	string(carriersettlement.InvoiceMatchStatusRejected),
})

func settlementToolProviders() []any {
	return []any{
		provideListCarrierInvoiceMatchesTool,
	}
}

type carrierSettlementReader interface {
	ListInvoiceMatches(
		ctx context.Context,
		req *carriersettlementports.ListCarrierInvoiceMatchesRequest,
	) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error)
}

func provideListCarrierInvoiceMatchesTool(
	settlements *carriersettlementservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListCarrierInvoiceMatchesTool(settlements, permissions)
}

type listCarrierInvoiceMatchesTool struct {
	settlements carrierSettlementReader
	access      fieldAccess
}

func newListCarrierInvoiceMatchesTool(
	settlements carrierSettlementReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listCarrierInvoiceMatchesTool{
		settlements: settlements,
		access:      newFieldAccess(permissions),
	}
}

func (t *listCarrierInvoiceMatchesTool) Name() string { return "carrier_list_invoice_matches" }

func (t *listCarrierInvoiceMatchesTool) Description() string {
	return "List carrier freight invoices matched against what each load was expected " +
		"to cost, with the variance. Each has the carrier's invoice number, the match " +
		"status and how it was matched. Narrow to one carrier or a status such as " +
		"Variance. An invoice a carrier sent over EDI is the carrier's own text."
}

func (t *listCarrierInvoiceMatchesTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramCarrierID: agenttoolschema.RecordIDText(permission.ResourceCarrier,
			"Only this carrier's invoices, by id from carrier_list."),
		paramStatus: enumParam("Only matches in this status.", invoiceMatchStatuses),
	}, defaultListLimit, maxListLimit))
}

func (t *listCarrierInvoiceMatchesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceCarrierSettlement,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceEDI,
		rationale: "Lists carrier invoices, some of which a carrier sent over EDI with " +
			"its own invoice text; nothing changes and nothing is sent.",
	})
}

type invoiceMatchRow struct {
	ID             string       `json:"id"`
	CarrierID      string       `json:"carrierId"`
	Carrier        string       `json:"carrier,omitempty"`
	InvoiceNumber  string       `json:"invoiceNumber,omitempty"`
	Status         string       `json:"status"`
	MatchedVia     string       `json:"matchedVia"`
	Source         string       `json:"source"`
	SettlementID   string       `json:"carrierSettlementId,omitempty"`
	Currency       string       `json:"currency"`
	InvoiceTotal   string       `json:"invoiceTotal,omitempty"`
	ExpectedTotal  string       `json:"expectedTotal,omitempty"`
	Variance       string       `json:"variance,omitempty"`
	ResolutionNote string       `json:"resolutionNote,omitempty"`
	ResolvedAt     optionalDate `json:"resolvedAt"`
	CreatedAt      optionalDate `json:"createdAt"`
}

func matchSource(match *carriersettlement.InvoiceMatch) string {
	switch {
	case match.HasEDISource():
		return "EDI"
	case match.HasDocumentAISource():
		return "Document"
	default:
		return "Manual"
	}
}

func (t *listCarrierInvoiceMatchesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	carrierID, err := optionalID(params.Params, paramCarrierID)
	if err != nil {
		return nil, err
	}
	status, err := validEnum(params.Params, paramStatus, invoiceMatchStatuses)
	if err != nil {
		return nil, err
	}
	window := readPage(params.Params, defaultListLimit, maxListLimit)

	criteria := filtercatalog.NewCriteria("carrier invoice matches").At(clockFor(params))
	if carrierID.IsNotNil() {
		criteria.Field(labelCarrier, carrierID.String())
	}
	if status != "" {
		criteria.Field(paramStatus, status)
	}

	result, err := t.settlements.ListInvoiceMatches(ctx,
		&carriersettlementports.ListCarrierInvoiceMatchesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantOf(params),
				Pagination: pagination.Info{Limit: window.fetch(), Offset: window.offset},
			},
			CarrierID: carrierID,
			Status:    carriersettlement.InvoiceMatchStatus(status),
		})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceCarrierSettlement)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)
	matches, more := trim(window, result.Items)
	rows := make([]invoiceMatchRow, 0, len(matches))
	tainted := make([]agent.RecordRef, 0, len(matches))
	for _, match := range matches {
		if match == nil {
			continue
		}
		row := invoiceMatchRow{
			ID:             match.ID.String(),
			CarrierID:      match.CarrierID.String(),
			InvoiceNumber:  match.InvoiceNumber,
			Status:         string(match.Status),
			MatchedVia:     string(match.MatchedVia),
			Source:         matchSource(match),
			SettlementID:   pointerIDString(match.CarrierSettlementID),
			Currency:       match.CurrencyCode,
			ResolutionNote: match.ResolutionNote,
			ResolvedAt: expectedDate(
				typeutils.ValueOrZero(match.ResolvedAt),
				absentNotResolved,
			),
			CreatedAt: recordedDate(match.CreatedAt),
		}
		if match.Carrier != nil {
			row.Carrier = match.Carrier.Name
		}
		if showAmounts {
			row.InvoiceTotal = minorText(match.InvoiceTotalMinor)
			row.ExpectedTotal = minorText(match.ExpectedTotalMinor)
			row.Variance = minorText(match.VarianceMinor)
		}
		rows = append(rows, row)
		if match.HasEDISource() {
			tainted = append(tainted, agent.RecordRef{
				EntityType: carrierInvoiceEntity,
				ID:         match.EDICarrierInvoiceID.String(),
			})
		}
	}

	found := searchResult(criteria, rows, len(rows)).paged(window, more)
	outcome := gatedResult(&found, gate)

	return outcome.withTaint(tainted), nil
}
