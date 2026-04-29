package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type LocationCodeInput struct {
	Name              string
	City              string
	StateAbbreviation string
	PostalCode        string
}

type LocationCodeGenerateRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Input          LocationCodeInput
}

type LocationCodeGenerator interface {
	Generate(ctx context.Context, req LocationCodeGenerateRequest) (string, error)
	BuildPrefix(input LocationCodeInput, strategy *tenant.LocationCodeStrategy) (string, error)
}
