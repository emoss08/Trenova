package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type FieldCeilingsFactoryParams struct {
	fx.In

	PermissionEngine permissionports.PermissionEngine
}

type FieldCeilingsFactory struct {
	permissionEngine permissionports.PermissionEngine
}

func NewFieldCeilingsFactory(p FieldCeilingsFactoryParams) *FieldCeilingsFactory {
	return &FieldCeilingsFactory{permissionEngine: p.PermissionEngine}
}

func newFieldCeilings(
	f *FieldCeilingsFactory,
	tenantInfo pagination.TenantInfo,
) aiauditports.FieldCeilings {
	return fieldsensitivity.NewCeilings(f.permissionEngine, tenantInfo.UserID, tenantInfo.OrgID)
}

var fieldCeilingsKey = loadercore.NewKey(newFieldCeilings)

func FieldCeilings(ctx context.Context) aiauditports.FieldCeilings {
	ceilings, _ := loadercore.Get(ctx, fieldCeilingsKey)
	return ceilings
}

func (f *FieldCeilingsFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
