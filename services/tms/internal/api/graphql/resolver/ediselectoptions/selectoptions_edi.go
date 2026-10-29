package ediselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	EdiSetupService    *edisetup.Service
	EdiTransferService *editransferservice.Service
}

func (r *Sources) EDICommunicationProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiSetupService.GetCommunicationProfile(
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

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiSetupService.SelectCommunicationProfileOptions(
		ctx,
		&edipartnerports.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediCommunicationProfileSelectOptionItem,
	)
}

func (r *Sources) EDIDocumentType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		result, err := r.EdiSetupService.SelectDocumentTypeOptions(
			ctx,
			&edipartnerports.EDIDocumentTypeSelectOptionsRequest{
				SelectQueryRequest: req.SelectQuery,
			},
		)
		if err != nil {
			return nil, err
		}
		items := selectoption.OrderedItems(
			req.IDs,
			result.Items,
			func(e *edidocument.EDIDocumentType) pulid.ID { return e.ID },
			ediDocumentTypeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiSetupService.SelectDocumentTypeOptions(
		ctx,
		&edipartnerports.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediDocumentTypeSelectOptionItem,
	)
}

func (r *Sources) EDITransactionSet(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	selectQuery := req.SelectQuery
	if len(req.IDs) > 0 {
		byIDs := *req.SelectQuery
		byIDs.Pagination = pagination.Info{Limit: pagination.ClampLimit(len(req.IDs))}
		selectQuery = &byIDs
	}

	result, err := r.EdiSetupService.SelectTransactionSetOptions(
		ctx,
		&edipartnerports.EDITransactionSetSelectOptionsRequest{
			SelectQueryRequest: selectQuery,
			IDs:                req.IDs,
			Standard: edidocument.EDIStandard(
				selectoption.StringFilter(req.Filters, "standard"),
			),
			Status: edidocument.DocumentStatus(
				selectoption.StringFilter(req.Filters, "status"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := selectoption.OrderedItems(
			req.IDs,
			result.Items,
			func(e *edidocument.EDITransactionSet) pulid.ID { return e.ID },
			ediTransactionSetSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediTransactionSetSelectOptionItem,
	)
}

func (r *Sources) EDIMappingProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiSetupService.GetMappingProfileByID(
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

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiSetupService.SelectMappingProfileOptions(
		ctx,
		&edipartnerports.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediMappingProfileSelectOptionItem,
	)
}

func (r *Sources) EDIPartner(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EdiSetupService.GetPartner(
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

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiSetupService.SelectPartnerOptions(
		ctx,
		&edipartnerports.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediPartnerSelectOptionItem,
	)
}

func (r *Sources) EDIConnection(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EdiSetupService.GetConnectionsByIDs(
			ctx,
			edipartnerports.GetEDIConnectionsByIDsRequest{
				TenantInfo:    req.TenantInfo,
				ConnectionIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			ediConnectionID,
			ediConnectionSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiSetupService.ConnectionSelectOptions(
		ctx,
		&edipartnerports.EDIConnectionSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediConnectionSelectOptionItem,
	)
}

func (r *Sources) EDITransfer(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EdiTransferService.GetTransfersByIDs(
			ctx,
			editenderports.GetEDITransfersByIDsRequest{
				TenantInfo:  req.TenantInfo,
				TransferIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			ediTransferID,
			ediTransferSelectOptionItemFor(req.TenantInfo.OrgID),
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EdiTransferService.TransferSelectOptions(
		ctx,
		&editenderports.EDITransferSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ediTransferSelectOptionItemFor(req.TenantInfo.OrgID),
	)
}

func ediCommunicationProfileSelectOptionItem(
	entity *edi.EDICommunicationProfile,
) selectoption.Item {
	return selectoption.ItemFor(
		ediCommunicationProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediCommunicationProfileSelectOption(
	entity *edi.EDICommunicationProfile,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(string(entity.Method)),
		Meta: map[string]any{
			"method":                   string(entity.Method),
			selectoption.MetaKeyStatus: string(entity.Status),
		},
	}
}

func ediDocumentTypeSelectOptionItem(
	entity *edidocument.EDIDocumentType,
) selectoption.Item {
	return selectoption.ItemFor(
		ediDocumentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediDocumentTypeSelectOption(entity *edidocument.EDIDocumentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: entity.Code + " - " + entity.Name,
		Description: stringutils.Ptr(
			string(entity.TransactionSet) + " / " + string(entity.Direction),
		),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
			"transactionSet":         string(entity.TransactionSet),
			"direction":              string(entity.Direction),
			"defaultVersion":         entity.DefaultVersion,
		},
	}
}

func ediTransactionSetSelectOptionItem(
	entity *edidocument.EDITransactionSet,
) selectoption.Item {
	return selectoption.ItemFor(
		ediTransactionSetSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediTransactionSetSelectOption(entity *edidocument.EDITransactionSet) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       string(entity.Code) + " - " + entity.Name,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			selectoption.MetaKeyCode:   string(entity.Code),
			"standard":                 string(entity.Standard),
			"defaultVersion":           entity.DefaultVersion,
			selectoption.MetaKeyStatus: string(entity.Status),
		},
	}
}

func ediMappingProfileSelectOptionItem(entity *edi.EDIMappingProfile) selectoption.Item {
	return selectoption.ItemFor(
		ediMappingProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediMappingProfileSelectOption(entity *edi.EDIMappingProfile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
	}
}

func ediPartnerSelectOptionItem(entity *edi.EDIPartner) selectoption.Item {
	return selectoption.ItemFor(
		ediPartnerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediPartnerSelectOption(entity *edi.EDIPartner) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code + " - " + entity.Name,
		Description: stringutils.Ptr(string(entity.Kind)),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
			"kind":                   string(entity.Kind),
		},
	}
}

func ediTransferSelectOptionItemFor(
	orgID pulid.ID,
) func(*editransfer.EDITransfer) selectoption.Item {
	return func(entity *editransfer.EDITransfer) selectoption.Item {
		return selectoption.ItemFor(
			ediTransferSelectOption(entity, orgID),
			entity.CreatedAt,
			entity.ID,
		)
	}
}

func ediTransferSelectOption(
	entity *editransfer.EDITransfer,
	orgID pulid.ID,
) *gqlmodel.SelectOption {
	label := entity.TenderPayload.BOL
	if label == "" {
		label = "Load tender " + entity.ID.String()
	}

	meta := map[string]any{
		selectoption.MetaKeyStatus: string(entity.Status),
		"bol":                      entity.TenderPayload.BOL,
		"customerLabel":            entity.TenderPayload.CustomerLabel,
	}
	if partner := entity.Counterparty(orgID); partner != nil {
		meta["partner"] = partner.Name
	}

	description := entity.TenderPayload.CustomerLabel
	if description == "" {
		description = entity.TenderPayload.ServiceTypeLabel
	}

	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringutils.Ptr(description),
		Meta:        meta,
	}
}

func ediConnectionSelectOptionItem(entity *edi.EDIConnection) selectoption.Item {
	return selectoption.ItemFor(
		ediConnectionSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediConnectionSelectOption(entity *edi.EDIConnection) *gqlmodel.SelectOption {
	source := ""
	if entity.SourceOrganization != nil {
		source = entity.SourceOrganization.Name
	}
	if source == "" {
		source = entity.SourceOrganizationID.String()
	}
	target := ""
	if entity.TargetOrganization != nil {
		target = entity.TargetOrganization.Name
	}
	if target == "" {
		target = entity.TargetOrganizationID.String()
	}
	label := source + " → " + target

	description := string(entity.Method) + " \u00b7 " + string(entity.Status)

	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringutils.Ptr(description),
		Meta: map[string]any{
			"method":                   string(entity.Method),
			selectoption.MetaKeyStatus: string(entity.Status),
			"sourceOrganizationId":     entity.SourceOrganizationID.String(),
			"targetOrganizationId":     entity.TargetOrganizationID.String(),
			"sourceOrganizationName":   source,
			"targetOrganizationName":   target,
			"sourcePartnerId":          selectoption.OptionalIDMeta(entity.SourcePartnerID),
			"targetPartnerId":          selectoption.OptionalIDMeta(entity.TargetPartnerID),
		},
	}
}

func ediConnectionID(entity *edi.EDIConnection) pulid.ID {
	return entity.ID
}

func ediTransferID(entity *editransfer.EDITransfer) pulid.ID {
	return entity.ID
}
