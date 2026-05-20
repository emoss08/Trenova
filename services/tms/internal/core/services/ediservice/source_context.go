package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListSourceContextSchemas(
	ctx context.Context,
	req *repositories.ListEDISourceContextSchemasRequest,
) (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
	return s.sourceContextRepo.ListSourceContextSchemas(ctx, req)
}

func (s *Service) ListPartnerSettingSchemas(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingSchemasRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
	return s.partnerSettingRepo.ListPartnerSettingSchemas(ctx, req)
}

func (s *Service) GetPartnerSettingSchema(
	ctx context.Context,
	req repositories.GetEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	return s.partnerSettingRepo.GetPartnerSettingSchema(ctx, req)
}

func (s *Service) ListPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return s.partnerSettingRepo.ListPartnerSettingFields(ctx, req)
}

func (s *Service) SearchPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return s.partnerSettingRepo.SearchPartnerSettingFields(ctx, req)
}

func (s *Service) SelectPartnerSettingFieldOptions(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return s.partnerSettingRepo.SelectPartnerSettingFieldOptions(ctx, req)
}

func (s *Service) GetSourceContextSchema(
	ctx context.Context,
	req repositories.GetEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	return s.sourceContextRepo.GetSourceContextSchema(ctx, req)
}

func (s *Service) ListSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return s.sourceContextRepo.ListSourceContextFields(ctx, req)
}

func (s *Service) SearchSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return s.sourceContextRepo.SearchSourceContextFields(ctx, req)
}

func (s *Service) SelectSourceContextFieldOptions(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return s.sourceContextRepo.SelectSourceContextFieldOptions(ctx, req)
}
