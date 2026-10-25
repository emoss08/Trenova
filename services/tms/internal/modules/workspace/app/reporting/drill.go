package reporting

import (
	"context"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
)

type DrillValue struct {
	ColumnID string
	Value    any
}

type DrillThroughRequest struct {
	Request

	Definition  *ir.Definition
	Params      map[string]any
	Dimensions  []DrillValue
	ColumnID    string
	PivotValues []string
	Limit       int
}

func (s *Service) DrillThrough(
	ctx context.Context,
	req *DrillThroughRequest,
) (*PreviewResult, error) {
	prepared, err := s.Prepare(ctx, &PrepareParams{
		Tenant:     req.TenantInfo,
		Definition: req.Definition,
		Params:     req.Params,
		Mode:       reportaccess.ModeRead,
	})
	if err != nil {
		return nil, err
	}

	row := make([]ir.DrillValue, 0, len(req.Dimensions))
	for _, d := range req.Dimensions {
		row = append(row, ir.DrillValue{Column: d.ColumnID, Value: d.Value})
	}

	limit := req.Limit
	if limit < 0 || limit >= s.cfg.GetPreviewRowLimit() {
		limit = 0
	}

	result, err := s.engine.Drill(prepared.Context, &prepared.Request, row, req.ColumnID,
		&analytics.DrillOptions{
			Policy:      prepared.Access,
			Location:    prepared.Location,
			Params:      req.Params,
			PivotValues: req.PivotValues,
			Limit:       limit,
		})
	if err != nil {
		return nil, EngineError(prepared.Context, prepared.Access, err)
	}

	out, err := collectPreview(prepared.Context, result)
	if err != nil {
		return nil, err
	}
	out.Timezone = prepared.Location.String()
	return out, nil
}
