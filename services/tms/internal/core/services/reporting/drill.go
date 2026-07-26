package reporting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

const (
	// maxDrillColumns caps the width of a drill-through list so a click on an
	// aggregate returns something readable rather than the whole entity.
	maxDrillColumns = 14
	maxDrillRoot    = 8
	pivotIDSep      = ":"
	pivotOtherValue = "__other__"
)

// DrillValue pins one grouping column of the row that was clicked.
type DrillValue struct {
	ColumnID string
	Value    any
}

type DrillThroughRequest struct {
	Request

	Definition *report.Definition
	Params     map[string]any
	// Dimensions identify the row: one entry per grouping column.
	Dimensions []DrillValue
	// ColumnID is the output column that was clicked. When it names a measure
	// (or one of its pivot cells) the drill inherits that measure's own scope.
	ColumnID string
	Limit    int
}

// DrillThrough answers "which records produced this number" by rewriting the
// aggregate definition into a record list constrained to the clicked row: the
// report's own filters, plus the row's grouping values, plus whatever narrowed
// the specific measure that was clicked.
func (s *Service) DrillThrough(
	ctx context.Context,
	req *DrillThroughRequest,
) (*PreviewResult, error) {
	if req.Definition == nil {
		return nil, errortypes.NewValidationError(
			"definition", errortypes.ErrRequired, "Definition is required",
		)
	}

	entity, ok := reportcatalog.Default.Entity(req.Definition.Entity)
	if !ok {
		return nil, errortypes.NewValidationError(
			"definition.entity", errortypes.ErrInvalid,
			fmt.Sprintf("Unknown entity %q", req.Definition.Entity),
		)
	}

	loc, err := time.LoadLocation(s.orgTimezone(ctx, req.TenantInfo))
	if err != nil {
		loc = time.UTC
	}

	detail, err := s.detailDefinition(ctx, req, entity, loc)
	if err != nil {
		return nil, err
	}

	return s.runPreview(ctx, &PreviewRequest{
		Request:    req.Request,
		Definition: detail,
		Params:     req.Params,
	})
}

func (s *Service) detailDefinition(
	ctx context.Context,
	req *DrillThroughRequest,
	entity *reportcatalog.Entity,
	loc *time.Location,
) (*report.Definition, error) {
	filters := &report.FilterGroup{Op: report.BoolOpAnd}
	if !req.Definition.Filters.IsEmpty() {
		filters.Groups = append(filters.Groups, *req.Definition.Filters)
	}

	for i := range req.Dimensions {
		leaves, err := drillDimensionFilters(req.Definition, &req.Dimensions[i], loc)
		if err != nil {
			return nil, err
		}
		filters.Filters = append(filters.Filters, leaves...)
	}

	if err := appendMeasureScope(req.Definition, req.ColumnID, filters); err != nil {
		return nil, err
	}

	columns, err := s.detailColumns(ctx, req, entity)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, errortypes.NewAuthorizationError(
			"You do not have access to any fields of " + entity.PluralLabel,
		)
	}

	limit := req.Limit
	previewLimit := s.cfg.GetPreviewRowLimit()
	if limit <= 0 || limit > previewLimit {
		limit = previewLimit
	}

	return &report.Definition{
		IRVersion:  report.CurrentIRVersion,
		Entity:     req.Definition.Entity,
		Columns:    columns,
		Filters:    filters,
		Parameters: req.Definition.Parameters,
		Limit:      limit,
	}, nil
}

// drillDimensionFilters turns one grouping value back into row predicates. A
// bucketed date becomes the half-open range the bucket covered, and a
// transformed dimension carries its transform onto the filter so the comparison
// happens on the same normalized value the row displayed.
func drillDimensionFilters(
	def *report.Definition,
	value *DrillValue,
	loc *time.Location,
) ([]report.FieldFilter, error) {
	col, ok := def.ColumnByID(value.ColumnID)
	if !ok || col.Kind != report.ColumnKindDimension {
		return nil, errortypes.NewValidationError(
			"dimensions", errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a grouping column of this report", value.ColumnID),
		)
	}

	if value.Value == nil {
		return []report.FieldFilter{{Ref: col.Ref, Operator: dbtype.OpIsNull}}, nil
	}

	if col.Bucket != report.DateBucketNone {
		start, end, err := bucketRange(col.Bucket, value.Value, loc)
		if err != nil {
			return nil, err
		}
		return []report.FieldFilter{
			{Ref: col.Ref, Operator: dbtype.OpGreaterThanOrEqual, Value: start},
			{Ref: col.Ref, Operator: dbtype.OpLessThan, Value: end},
		}, nil
	}

	if !col.Band.IsEmpty() {
		return bandFilters(col, value.Value)
	}

	return []report.FieldFilter{{
		Ref:       col.Ref,
		Operator:  dbtype.OpEqual,
		Value:     value.Value,
		Transform: col.Transform,
	}}, nil
}

// bucketRange inverts the compiler's date_trunc bucketing. The emitted bucket
// value is the truncated local wall clock read as a UTC epoch, so it is read
// back as UTC and reinterpreted in the organization's zone.
func bucketRange(
	bucket report.DateBucket,
	value any,
	loc *time.Location,
) (start, end int64, err error) {
	seconds, err := drillEpoch(value)
	if err != nil {
		return 0, 0, err
	}

	wall := time.Unix(seconds, 0).UTC()
	local := time.Date(
		wall.Year(), wall.Month(), wall.Day(),
		wall.Hour(), wall.Minute(), wall.Second(), 0, loc,
	)

	switch bucket {
	case report.DateBucketDay:
		return local.Unix(), local.AddDate(0, 0, 1).Unix(), nil
	case report.DateBucketWeek:
		return local.Unix(), local.AddDate(0, 0, 7).Unix(), nil
	case report.DateBucketMonth:
		return local.Unix(), local.AddDate(0, 1, 0).Unix(), nil
	case report.DateBucketQuarter:
		return local.Unix(), local.AddDate(0, 3, 0).Unix(), nil
	case report.DateBucketYear:
		return local.Unix(), local.AddDate(1, 0, 0).Unix(), nil
	case report.DateBucketNone:
		return local.Unix(), local.Unix() + 1, nil
	default:
		return 0, 0, errortypes.NewValidationError(
			"dimensions", errortypes.ErrInvalid,
			fmt.Sprintf("Unknown date bucket %q", bucket),
		)
	}
}

// bandFilters inverts a numeric band back into the half-open range it grouped.
// The open ends are deliberate: the lowest band also swept up anything beneath
// its first boundary and the highest band everything above its last, so
// re-applying those bounds would hide rows the band itself counted.
func bandFilters(col *report.ColumnSpec, value any) ([]report.FieldFilter, error) {
	number, err := drillNumber(value)
	if err != nil {
		return nil, err
	}

	lower, upper, bounded := col.Band.Bounds(number)

	filters := make([]report.FieldFilter, 0, 2)
	if !col.Band.IsLowest(number) {
		filters = append(filters, report.FieldFilter{
			Ref: col.Ref, Operator: dbtype.OpGreaterThanOrEqual, Value: lower,
		})
	}
	if bounded {
		filters = append(filters, report.FieldFilter{
			Ref: col.Ref, Operator: dbtype.OpLessThan, Value: upper,
		})
	}
	if len(filters) == 0 {
		return []report.FieldFilter{{Ref: col.Ref, Operator: dbtype.OpIsNotNull}}, nil
	}
	return filters, nil
}

func drillNumber(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, errortypes.NewValidationError(
				"dimensions", errortypes.ErrInvalid,
				fmt.Sprintf("Expected a number for a ranged column, got %q", v),
			)
		}
		return parsed, nil
	default:
		return 0, errortypes.NewValidationError(
			"dimensions", errortypes.ErrInvalid,
			fmt.Sprintf("Expected a number for a ranged column, got %T", value),
		)
	}
}

func drillEpoch(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, errortypes.NewValidationError(
				"dimensions", errortypes.ErrInvalid,
				fmt.Sprintf("Expected a timestamp for a bucketed column, got %q", v),
			)
		}
		return parsed, nil
	default:
		return 0, errortypes.NewValidationError(
			"dimensions", errortypes.ErrInvalid,
			fmt.Sprintf("Expected a timestamp for a bucketed column, got %T", value),
		)
	}
}

// appendMeasureScope narrows the drill to the specific cell that was clicked:
// the measure's own filter, and — for a pivoted measure — the bucket the cell
// belongs to.
func appendMeasureScope(
	def *report.Definition,
	columnID string,
	filters *report.FilterGroup,
) error {
	if columnID == "" {
		return nil
	}

	baseID, pivotValue, isPivotCell := strings.Cut(columnID, pivotIDSep)
	col, ok := def.ColumnByID(baseID)
	if !ok {
		return errortypes.NewValidationError(
			"columnId", errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a column of this report", columnID),
		)
	}
	if col.Kind == report.ColumnKindDimension {
		return nil
	}

	if !col.Filter.IsEmpty() {
		filters.Groups = append(filters.Groups, *col.Filter)
	}

	if !isPivotCell || def.Pivot == nil {
		return nil
	}
	return appendPivotScope(def.Pivot, pivotValue, filters)
}

func appendPivotScope(
	pivot *report.PivotSpec,
	pivotValue string,
	filters *report.FilterGroup,
) error {
	if pivotValue != pivotOtherValue {
		filters.Filters = append(filters.Filters, report.FieldFilter{
			Ref:      pivot.Ref,
			Operator: dbtype.OpEqual,
			Value:    pivotValue,
		})
		return nil
	}

	// The "Other" bucket is everything the named values did not claim, nulls
	// included, so it inverts to the same OR the compiler emits.
	values := make([]any, 0, len(pivot.Values))
	for _, value := range pivot.Values {
		values = append(values, value)
	}
	filters.Groups = append(filters.Groups, report.FilterGroup{
		Op: report.BoolOpOr,
		Filters: []report.FieldFilter{
			{Ref: pivot.Ref, Operator: dbtype.OpIsNull},
			{Ref: pivot.Ref, Operator: dbtype.OpNotIn, Value: values},
		},
	})
	return nil
}

// detailColumns picks the record view: the root entity's own fields, in the
// order the catalog declares them, followed by the grouping columns the report
// reached for through relationships — skipping anything this user cannot see.
func (s *Service) detailColumns(
	ctx context.Context,
	req *DrillThroughRequest,
	entity *reportcatalog.Entity,
) ([]report.ColumnSpec, error) {
	detail, err := s.perms.GetResourcePermissions(
		ctx, req.TenantInfo.UserID, req.TenantInfo.OrgID, entity.Resource.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("resolve permissions for %q: %w", entity.Resource, err)
	}

	columns := make([]report.ColumnSpec, 0, maxDrillColumns)
	seen := make(map[string]bool, maxDrillColumns)

	for i := range entity.Fields {
		field := &entity.Fields[i]
		if len(columns) >= maxDrillRoot {
			break
		}
		if field.Key == "id" || field.Type == reportcatalog.FieldJSON {
			continue
		}
		if !s.fieldVisible(detail, entity, field) {
			continue
		}
		ref := report.FieldRef{Field: field.Key}
		seen[ref.String()] = true
		columns = append(columns, report.ColumnSpec{
			ID:   fmt.Sprintf("d%d", len(columns)),
			Ref:  ref,
			Kind: report.ColumnKindDimension,
		})
	}

	for i := range req.Definition.Columns {
		col := &req.Definition.Columns[i]
		if col.Kind != report.ColumnKindDimension || len(col.Ref.Path) == 0 {
			continue
		}
		if len(columns) >= maxDrillColumns || seen[col.Ref.String()] {
			continue
		}
		seen[col.Ref.String()] = true
		columns = append(columns, report.ColumnSpec{
			ID:   fmt.Sprintf("d%d", len(columns)),
			Ref:  col.Ref,
			Kind: report.ColumnKindDimension,
		})
	}

	return columns, nil
}

func (s *Service) fieldVisible(
	detail *services.ResourcePermissionDetail,
	entity *reportcatalog.Entity,
	field *reportcatalog.Field,
) bool {
	sensitivity := s.permRegistry.GetFieldSensitivity(entity.Resource.String(), field.Key)
	if !detail.MaxSensitivity.CanAccess(sensitivity) {
		return false
	}
	if len(detail.AccessibleFields) == 0 {
		return true
	}
	for _, allowed := range detail.AccessibleFields {
		if allowed == field.Key {
			return true
		}
	}
	return false
}
