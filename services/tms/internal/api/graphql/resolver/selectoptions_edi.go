package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *Resolver) resolveEDICommunicationProfileSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetCommunicationProfile(
				ctx,
				repositories.GetEDICommunicationProfileByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediCommunicationProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectCommunicationProfileOptions(
		ctx,
		&repositories.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediCommunicationProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIDocumentTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		result, err := r.ediService.SelectDocumentTypeOptions(
			ctx,
			&repositories.EDIDocumentTypeSelectOptionsRequest{
				SelectQueryRequest: req.selectQuery,
			},
		)
		if err != nil {
			return nil, err
		}
		items := orderedSelectOptionItems(
			req.ids,
			result.Items,
			func(e *edi.EDIDocumentType) pulid.ID { return e.ID },
			ediDocumentTypeSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectDocumentTypeOptions(
		ctx,
		&repositories.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediDocumentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveEDITransactionSetSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	selectQuery := req.selectQuery
	if len(req.ids) > 0 {
		byIDs := *req.selectQuery
		byIDs.Pagination = pagination.Info{Limit: pagination.ClampLimit(len(req.ids))}
		selectQuery = &byIDs
	}

	result, err := r.ediService.SelectTransactionSetOptions(
		ctx,
		&repositories.EDITransactionSetSelectOptionsRequest{
			SelectQueryRequest: selectQuery,
			IDs:                req.ids,
			Standard:           edi.EDIStandard(selectOptionStringFilter(req.filters, "standard")),
			Status:             edi.DocumentStatus(selectOptionStringFilter(req.filters, "status")),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(req.ids) > 0 {
		items := orderedSelectOptionItems(
			req.ids,
			result.Items,
			func(e *edi.EDITransactionSet) pulid.ID { return e.ID },
			ediTransactionSetSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediTransactionSetSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIMappingProfileSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetMappingProfileByID(
				ctx,
				repositories.GetMappingProfileByIDRequest{
					ProfileID:  id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediMappingProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectMappingProfileOptions(
		ctx,
		&repositories.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediMappingProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIPartnerSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetPartner(
				ctx,
				repositories.GetEDIPartnerByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectPartnerOptions(
		ctx,
		&repositories.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediPartnerSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIPartnerDocumentProfileSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetPartnerDocumentProfile(
				ctx,
				repositories.GetEDIPartnerDocumentProfileByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerDocumentProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectPartnerDocumentProfileOptions(
		ctx,
		&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			PartnerID:          selectOptionIDFilter(req.filters, "partnerId"),
			TransactionSet: edi.TransactionSet(
				selectOptionStringFilter(req.filters, "transactionSet"),
			),
			Direction: edi.DocumentDirection(
				selectOptionStringFilter(req.filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediPartnerDocumentProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDITemplateSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetTemplate(
				ctx,
				repositories.GetEDITemplateByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectTemplateOptions(
		ctx,
		&repositories.EDITemplateSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			TransactionSet: edi.TransactionSet(
				selectOptionStringFilter(req.filters, "transactionSet"),
			),
			Direction: edi.DocumentDirection(
				selectOptionStringFilter(req.filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediTemplateSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIConnectionSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.ediService.GetConnectionsByIDs(
			ctx,
			repositories.GetEDIConnectionsByIDsRequest{
				TenantInfo:    req.tenantInfo,
				ConnectionIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			ediConnectionID,
			ediConnectionSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.ConnectionSelectOptions(
		ctx,
		&repositories.EDIConnectionSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediConnectionSelectOptionItem,
	)
}

func (r *Resolver) resolveEDITransferSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.ediService.GetTransfersByIDs(
			ctx,
			repositories.GetEDITransfersByIDsRequest{
				TenantInfo:  req.tenantInfo,
				TransferIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			ediTransferID,
			ediTransferSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.TransferSelectOptions(
		ctx,
		&repositories.EDITransferSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediTransferSelectOptionItem,
	)
}
