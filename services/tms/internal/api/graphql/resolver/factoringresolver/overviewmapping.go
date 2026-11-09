package factoringresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/accountingloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringoverviewservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/shared/pulid"
)

func overviewToModel(o *factoringoverviewservice.Overview) *gqlmodel.FactoringCompanyOverview {
	attention := make([]*gqlmodel.FactoringAttention, len(o.Attention))
	for i := range o.Attention {
		attention[i] = attentionToModel(&o.Attention[i])
	}
	summary := make([]*gqlmodel.FactoringOverviewSegment, len(o.Summary))
	for i, segment := range o.Summary {
		summary[i] = &gqlmodel.FactoringOverviewSegment{
			Text:   segment.Text,
			Strong: segment.Strong,
			Tone:   stringOrNil(string(segment.Tone)),
		}
		if segment.Target != "" {
			target := segment.Target
			summary[i].Target = &target
		}
	}
	pace := make([]*factoringoverviewservice.PacePoint, len(o.Pace))
	for i := range o.Pace {
		pace[i] = &o.Pace[i]
	}
	delivery := &gqlmodel.FactoringDeliveryHealth{
		Configured:    o.Delivery.Configured,
		LastTestedAt:  timestampOrNil(o.Delivery.LastTestedAt),
		LastTestError: stringOrNil(o.Delivery.LastTestError),
	}
	if o.Delivery.Configured {
		method := o.Delivery.Method
		delivery.Method = &method
	}
	setup := o.Setup
	notices := o.Notices

	return &gqlmodel.FactoringCompanyOverview{
		FactoringCompanyID: o.FactoringCompanyID.String(),
		CurrencyCode:       o.CurrencyCode,
		AmountsIncluded:    o.AmountsIncluded,
		MovedWindowDays:    o.MovedWindowDays,
		Ready:              totals(o.Ready),
		ReadyLate:          totals(o.ReadyLate),
		Submitted:          totals(o.Submitted),
		Funded:             totals(o.Funded),
		ReserveHeld:        totals(o.ReserveHeld),
		Released:           totals(o.Released),
		ChargebacksDue:     totals(o.ChargebacksDue),
		ChargedBack:        totals(o.ChargedBack),
		OldestFundedOn:     timestampOrNil(o.OldestFundedOn),
		ChargebackDeadline: timestampOrNil(o.ChargebackDeadline),
		Awaiting:           o.Awaiting,
		AwaitingCount:      o.AwaitingCount,
		Pace:               pace,
		PaceMedianDays:     o.PaceMedianDays,
		PaceLate:           o.PaceLate,
		Agreement:          o.Agreement,
		Delivery:           delivery,
		Notices:            &notices,
		Attention:          attention,
		Setup:              &setup,
		Summary:            summary,
	}
}

func attentionToModel(a *factoringoverviewservice.Attention) *gqlmodel.FactoringAttention {
	return &gqlmodel.FactoringAttention{
		Kind:           a.Kind,
		Tone:           string(a.Tone),
		Count:          a.Count,
		AmountMinor:    int(a.AmountMinor),
		At:             timestampOrNil(a.At),
		SubmissionID:   idOrNil(a.SubmissionID),
		Number:         stringOrNil(a.Number),
		InvoiceID:      idOrNil(a.InvoiceID),
		InvoiceNumber:  stringOrNil(a.InvoiceNumber),
		CustomerName:   stringOrNil(a.CustomerName),
		RemittanceID:   idOrNil(a.RemittanceID),
		FileName:       stringOrNil(a.FileName),
		LineCount:      a.LineCount,
		ExceptionCount: a.ExceptionCount,
		Detail:         stringOrNil(a.Detail),
	}
}

func customerPageToModel(
	page *factoringports.FactorCustomerPage,
	notices bool,
	amounts bool,
) (*gqlmodel.FactoringCompanyCustomerConnection, error) {
	edges := make([]*gqlmodel.FactoringCompanyCustomerEdge, len(page.Items))
	for i, item := range page.Items {
		cursor, err := gqlconnection.EdgeCursor(page.EdgeCursors, i)
		if err != nil {
			return nil, err
		}
		node := &gqlmodel.FactoringCompanyCustomer{
			ID:          item.CustomerID.String(),
			Code:        item.Code,
			Name:        item.Name,
			Status:      item.Status,
			FundedCount: item.FundedCount,
		}
		if notices {
			node.Notice = item.Notice
			node.LastReleasedAt = timestampOrNil(item.LastReleasedAt)
		}
		if amounts {
			node.FundedMinor = int(item.FundedMinor)
		}
		edges[i] = &gqlmodel.FactoringCompanyCustomerEdge{Node: node, Cursor: cursor}
	}

	return &gqlmodel.FactoringCompanyCustomerConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			page.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.FactoringCompanyCustomerEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func totals(t factoringports.VerdictTotals) *factoringports.VerdictTotals {
	return &t
}

func timestampOrNil(at *int64) *int {
	if at == nil {
		return nil
	}
	value := int(*at)
	return &value
}

func stringOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (r *FactoringAgreementResolver) accountRef(
	ctx context.Context,
	id pulid.ID,
) (*gqlmodel.JournalEntryLineAccount, error) {
	if id.IsNil() || !loadercore.Available(ctx) {
		return nil, nil
	}
	if _, err := r.RequireAuth(ctx); err != nil {
		return nil, err
	}
	account, err := accountingloaders.GLAccountByID(ctx, id.String())
	if err != nil {
		return nil, nil //nolint:nilerr,nilnil // the page falls back to the organization default line
	}
	return &gqlmodel.JournalEntryLineAccount{
		ID:          account.ID.String(),
		AccountCode: account.AccountCode,
		Name:        account.Name,
	}, nil
}
