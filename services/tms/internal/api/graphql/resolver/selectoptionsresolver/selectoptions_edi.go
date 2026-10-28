package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
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
				edipartnerports.GetEDICommunicationProfileByIDRequest{
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
		&edipartnerports.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
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
			&edipartnerports.EDIDocumentTypeSelectOptionsRequest{
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
		&edipartnerports.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
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
		&edipartnerports.EDITransactionSetSelectOptionsRequest{
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
		req.SelectQuery.Pagination,
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
				edipartnerports.GetMappingProfileByIDRequest{
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
		&edipartnerports.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
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
				edipartnerports.GetEDIPartnerByIDRequest{
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
		&edipartnerports.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		ediPartnerSelectOptionItem,
	)
}

func (r *Deps) resolveEDIConnectionSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EdiService.GetConnectionsByIDs(
			ctx,
			edipartnerports.GetEDIConnectionsByIDsRequest{
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
		&edipartnerports.EDIConnectionSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
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
			editenderports.GetEDITransfersByIDsRequest{
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
			ediTransferSelectOptionItemFor(req.TenantInfo.OrgID),
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EdiService.TransferSelectOptions(
		ctx,
		&editenderports.EDITransferSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		ediTransferSelectOptionItemFor(req.TenantInfo.OrgID),
	)
}
