package tablelayoutresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/tablelayout"
	"github.com/emoss08/trenova/internal/core/services/tablelayoutservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
)

func tableLayoutRequest(authCtx *authctx.AuthContext, resource string) tablelayoutservice.Request {
	return tablelayoutservice.Request{
		TenantInfo: base.TenantInfo(authCtx),
		Resource:   resource,
	}
}

func layoutFromInput(raw map[string]any) (*tablelayout.Layout, error) {
	layout := new(tablelayout.Layout)
	if len(raw) == 0 {
		return layout, nil
	}
	if err := jsonutils.Convert(raw, layout); err != nil {
		return nil, errortypes.NewValidationError(
			"layout",
			errortypes.ErrInvalid,
			"The layout is not in a shape this table can keep",
		)
	}

	return layout, nil
}

func tableLayoutToModel(entity *tablelayout.TableLayout) (*gqlmodel.TableLayout, error) {
	if entity == nil {
		return nil, nil //nolint:nilnil // no layout saved is a null field, not an error
	}

	layout, err := jsonutils.ToJSON(entity.Layout)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.TableLayout{
		Resource:  entity.Resource,
		Layout:    layout,
		Version:   int(entity.Version),
		UpdatedAt: int(entity.UpdatedAt),
	}, nil
}
