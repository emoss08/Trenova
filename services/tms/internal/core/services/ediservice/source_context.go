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
	return s.documentRepo.ListSourceContextSchemas(ctx, req)
}

func (s *Service) ListPartnerSettingSchemas(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingSchemasRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
	return s.documentRepo.ListPartnerSettingSchemas(ctx, req)
}

func (s *Service) GetPartnerSettingSchema(
	ctx context.Context,
	req repositories.GetEDIPartnerSettingSchemaRequest,
) (*edi.EDIPartnerSettingSchema, error) {
	return s.documentRepo.GetPartnerSettingSchema(ctx, req)
}

func (s *Service) ListPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return s.documentRepo.ListPartnerSettingFields(ctx, req)
}

func (s *Service) SearchPartnerSettingFields(
	ctx context.Context,
	req *repositories.ListEDIPartnerSettingFieldsRequest,
) (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
	return s.documentRepo.SearchPartnerSettingFields(ctx, req)
}

func (s *Service) GetSourceContextSchema(
	ctx context.Context,
	req repositories.GetEDISourceContextSchemaRequest,
) (*edi.EDISourceContextSchema, error) {
	return s.documentRepo.GetSourceContextSchema(ctx, req)
}

func (s *Service) ListSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return s.documentRepo.ListSourceContextFields(ctx, req)
}

func (s *Service) SearchSourceContextFields(
	ctx context.Context,
	req *repositories.ListEDISourceContextFieldsRequest,
) (*pagination.ListResult[*edi.EDISourceContextField], error) {
	return s.documentRepo.SearchSourceContextFields(ctx, req)
}
