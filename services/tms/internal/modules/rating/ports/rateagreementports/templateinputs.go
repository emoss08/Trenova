package rateagreementports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type TemplateInputsKind string

const (
	TemplateInputsRule     = TemplateInputsKind("rule")
	TemplateInputsShipment = TemplateInputsKind("shipment")
)

type ListInputsByTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Limit      int
}

type TemplateInputs struct {
	Kind        TemplateInputsKind
	ExampleID   pulid.ID
	ExampleName string
	Count       int
	Rule        tarifftypes.Inputs
	Agreement   tarifftypes.Inputs
}

type TemplateInputsLister interface {
	ListInputsByTemplate(
		ctx context.Context,
		req *ListInputsByTemplateRequest,
	) ([]*TemplateInputs, error)
}
