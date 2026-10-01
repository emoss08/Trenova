package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *Deps) resolveEDICommunicationProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiService.GetCommunicationProfile(
				ctx,
				repositories.GetEDICommunicationProfileByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediCommunicationProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectCommunicationProfileOptions(
		ctx,
		&repositories.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediCommunicationProfileSelectOptionItem,
	)
}

func (r *Deps) resolveEDIDocumentTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		result, err := r.EdiService.SelectDocumentTypeOptions(
			ctx,
			&repositories.EDIDocumentTypeSelectOptionsRequest{
				SelectQueryRequest: req.SelectQuery,
			},
		)
		if err != nil {
			return nil, err
		}
		items := orderedSelectOptionItems(
			req.IDs,
			result.Items,
			func(e *edi.EDIDocumentType) pulid.ID { return e.ID },
			ediDocumentTypeSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectDocumentTypeOptions(
		ctx,
		&repositories.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediDocumentTypeSelectOptionItem,
	)
}

func (r *Deps) resolveEDITransactionSetSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	selectQuery := req.SelectQuery
	if len(req.IDs) > 0 {
		byIDs := *req.SelectQuery
		byIDs.Pagination = pagination.Info{Limit: pagination.ClampLimit(len(req.IDs))}
		selectQuery = &byIDs
	}

	result, err := r.EdiService.SelectTransactionSetOptions(
		ctx,
		&repositories.EDITransactionSetSelectOptionsRequest{
			SelectQueryRequest: selectQuery,
			IDs:                req.IDs,
			Standard:           edi.EDIStandard(selectOptionStringFilter(req.Filters, "standard")),
			Status:             edi.DocumentStatus(selectOptionStringFilter(req.Filters, "status")),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := orderedSelectOptionItems(
			req.IDs,
			result.Items,
			func(e *edi.EDITransactionSet) pulid.ID { return e.ID },
			ediTransactionSetSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediTransactionSetSelectOptionItem,
	)
}

func (r *Deps) resolveEDIMappingProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiService.GetMappingProfileByID(
				ctx,
				repositories.GetMappingProfileByIDRequest{
					ProfileID:  id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediMappingProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectMappingProfileOptions(
		ctx,
		&repositories.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediMappingProfileSelectOptionItem,
	)
}

func (r *Deps) resolveEDIPartnerSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiService.GetPartner(
				ctx,
				repositories.GetEDIPartnerByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectPartnerOptions(
		ctx,
		&repositories.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediPartnerSelectOptionItem,
	)
}

func (r *Deps) resolveEDIPartnerDocumentProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiService.GetPartnerDocumentProfile(
				ctx,
				repositories.GetEDIPartnerDocumentProfileByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerDocumentProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectPartnerDocumentProfileOptions(
		ctx,
		&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			PartnerID:          selectOptionIDFilter(req.Filters, "partnerId"),
			TransactionSet: edi.TransactionSet(
				selectOptionStringFilter(req.Filters, "transactionSet"),
			),
			Direction: edi.DocumentDirection(
				selectOptionStringFilter(req.Filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediPartnerDocumentProfileSelectOptionItem,
	)
}

func (r *Deps) resolveEDITemplateSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiService.GetTemplate(
				ctx,
				repositories.GetEDITemplateByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.SelectTemplateOptions(
		ctx,
		&repositories.EDITemplateSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			TransactionSet: edi.TransactionSet(
				selectOptionStringFilter(req.Filters, "transactionSet"),
			),
			Direction: edi.DocumentDirection(
				selectOptionStringFilter(req.Filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediTemplateSelectOptionItem,
	)
}

func (r *Deps) resolveEDIConnectionSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EdiService.GetConnectionsByIDs(
			ctx,
			repositories.GetEDIConnectionsByIDsRequest{
				TenantInfo:    req.TenantInfo,
				ConnectionIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.IDs,
			entities,
			ediConnectionID,
			ediConnectionSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.ConnectionSelectOptions(
		ctx,
		&repositories.EDIConnectionSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediConnectionSelectOptionItem,
	)
}

func (r *Deps) resolveEDITransferSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EdiService.GetTransfersByIDs(
			ctx,
			repositories.GetEDITransfersByIDsRequest{
				TenantInfo:  req.TenantInfo,
				TransferIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.IDs,
			entities,
			ediTransferID,
			ediTransferSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.TransferSelectOptions(
		ctx,
		&repositories.EDITransferSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ediTransferSelectOptionItem,
	)
}
