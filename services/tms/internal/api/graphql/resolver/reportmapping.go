package resolver

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	reportingservice "github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var reportSensitivityRegistry = permission.NewRegistry()

func reportIRFromGraphQL(input *gqlmodel.ReportIRInput) *report.Definition {
	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    input.Entity,
		Filters:   reportFilterGroupFromGraphQL(input.Filters),
		Having:    reportFilterGroupFromGraphQL(input.Having),
	}

	for _, col := range input.Columns {
		spec := report.ColumnSpec{
			ID:   col.ID,
			Ref:  reportFieldRefFromGraphQL(col.Ref),
			Kind: report.ColumnKind(col.Kind),
		}
		if col.Agg != nil {
			spec.Agg = reportcatalog.Aggregation(*col.Agg)
		}
		if col.Bucket != nil {
			spec.Bucket = report.DateBucket(*col.Bucket)
		}
		spec.Band = reportBandFromGraphQL(col.Band)
		if col.Label != nil {
			spec.Label = *col.Label
		}
		if col.Computed != nil {
			computed := &report.ComputedSpec{
				Op:         report.ComputedOp(col.Computed.Op),
				LeftID:     derefString(col.Computed.LeftID),
				RightID:    derefString(col.Computed.RightID),
				LeftValue:  col.Computed.LeftValue,
				RightValue: col.Computed.RightValue,
			}
			if col.Computed.Format != nil {
				computed.Format = reportcatalog.FormatHint(*col.Computed.Format)
			}
			spec.Computed = computed
		}
		spec.Transform = reportTransformFromGraphQL(col.Transform)
		spec.Display = reportDisplayFromGraphQL(col.Display)
		spec.Filter = reportFilterGroupFromGraphQL(col.Filter)
		def.Columns = append(def.Columns, spec)
	}

	for _, sortSpec := range input.Sort {
		def.Sort = append(def.Sort, report.SortSpec{
			ColumnID:  sortSpec.ColumnID,
			Direction: dbtype.SortDirection(sortSpec.Direction),
		})
	}

	if input.Limit != nil {
		def.Limit = *input.Limit
	}

	if input.Pivot != nil {
		pivot := &report.PivotSpec{
			Ref:        reportFieldRefFromGraphQL(input.Pivot.Ref),
			Values:     input.Pivot.Values,
			Labels:     input.Pivot.Labels,
			MeasureIDs: input.Pivot.MeasureIds,
		}
		if input.Pivot.IncludeOther != nil {
			pivot.IncludeOther = *input.Pivot.IncludeOther
		}
		def.Pivot = pivot
	}

	for _, param := range input.Parameters {
		def.Parameters = append(def.Parameters, reportParameterFromGraphQL(param))
	}

	if input.Totals != nil {
		def.Totals = *input.Totals
	}
	for _, chart := range input.Charts {
		def.Charts = append(def.Charts, reportChartFromGraphQL(chart))
	}

	return def
}

func reportParameterFromGraphQL(
	param *gqlmodel.ReportParameterDefInput,
) report.ParameterDef {
	paramDef := report.ParameterDef{
		Name:          param.Name,
		Type:          reportcatalog.FieldType(param.Type),
		Label:         derefString(param.Label),
		AllowedValues: param.AllowedValues,
		RefEntity:     derefString(param.RefEntity),
	}
	if param.Required != nil {
		paramDef.Required = *param.Required
	}
	if param.Default != nil {
		paramDef.Default = param.Default
	}
	if param.Multi != nil {
		paramDef.Multi = *param.Multi
	}
	return paramDef
}

func reportChartFromGraphQL(input *gqlmodel.ReportChartInput) report.ChartSpec {
	spec := report.ChartSpec{
		ID:        input.ID,
		Type:      report.ChartType(input.Type),
		SeriesIDs: input.SeriesIds,
	}
	if input.Title != nil {
		spec.Title = *input.Title
	}
	if input.XColumnID != nil {
		spec.XColumnID = *input.XColumnID
	}
	if input.Stacked != nil {
		spec.Stacked = *input.Stacked
	}
	if input.HideLegend != nil {
		spec.HideLegend = *input.HideLegend
	}
	if input.ShowValues != nil {
		spec.ShowValues = *input.ShowValues
	}
	if input.Curved != nil {
		spec.Curved = *input.Curved
	}
	if input.Limit != nil {
		spec.Limit = *input.Limit
	}
	if input.CompareID != nil {
		spec.CompareID = *input.CompareID
	}
	spec.LatColumnID = derefString(input.LatColumnID)
	spec.LngColumnID = derefString(input.LngColumnID)
	spec.LabelColumnID = derefString(input.LabelColumnID)
	if input.Goal != nil {
		goal := &report.ChartGoal{Label: derefString(input.Goal.Label)}
		if input.Goal.Value != nil {
			value := *input.Goal.Value
			goal.Value = &value
		}
		goal.ColumnID = derefString(input.Goal.ColumnID)
		if !goal.IsEmpty() {
			spec.Goal = goal
		}
	}
	return spec
}

func reportTransformFromGraphQL(input *gqlmodel.ReportTransformInput) *report.TransformSpec {
	if input == nil {
		return nil
	}
	spec := &report.TransformSpec{Op: report.TransformOp(input.Op)}
	if input.Precision != nil {
		precision := *input.Precision
		spec.Precision = &precision
	}
	if input.Factor != nil {
		factor := *input.Factor
		spec.Factor = &factor
	}
	return spec
}

func reportDisplayFromGraphQL(input *gqlmodel.ReportDisplayInput) *reportfmt.Spec {
	if input == nil {
		return nil
	}

	spec := &reportfmt.Spec{
		Style:         reportfmt.Style(derefString(input.Style)),
		Currency:      derefString(input.Currency),
		Negative:      reportfmt.Negative(derefString(input.Negative)),
		Notation:      reportfmt.Notation(derefString(input.Notation)),
		Prefix:        derefString(input.Prefix),
		Suffix:        derefString(input.Suffix),
		DateStyle:     reportfmt.DateStyle(derefString(input.DateStyle)),
		BoolStyle:     reportfmt.BoolStyle(derefString(input.BoolStyle)),
		DurationUnit:  reportfmt.DurationUnit(derefString(input.DurationUnit)),
		DurationStyle: reportfmt.DurationStyle(derefString(input.DurationStyle)),
		NullText:      derefString(input.NullText),
	}
	if input.Decimals != nil {
		decimals := *input.Decimals
		spec.Decimals = &decimals
	}
	if input.Grouping != nil {
		grouping := *input.Grouping
		spec.Grouping = &grouping
	}
	for _, rule := range input.Rules {
		converted := reportfmt.Rule{
			Op:    reportfmt.RuleOp(rule.Op),
			Value: rule.Value,
			Tone:  reportfmt.Tone(rule.Tone),
		}
		if rule.Upper != nil {
			converted.Upper = *rule.Upper
		}
		spec.Rules = append(spec.Rules, converted)
	}

	if spec.IsEmpty() {
		return nil
	}
	return spec
}

func reportBandFromGraphQL(input *gqlmodel.ReportBandInput) *reportfmt.Band {
	if input == nil {
		return nil
	}

	band := &reportfmt.Band{Edges: input.Edges}
	if input.Width != nil {
		band.Width = *input.Width
	}
	if band.IsEmpty() {
		return nil
	}
	return band
}

func reportBandToModel(band *reportfmt.Band) *gqlmodel.ReportBand {
	if band.IsEmpty() {
		return nil
	}
	return &gqlmodel.ReportBand{Width: band.Width, Edges: band.Edges}
}

func reportDisplayRulesToModel(rules []reportfmt.Rule) []*gqlmodel.ReportDisplayRule {
	models := make([]*gqlmodel.ReportDisplayRule, 0, len(rules))
	for i := range rules {
		models = append(models, &gqlmodel.ReportDisplayRule{
			Op:    string(rules[i].Op),
			Value: rules[i].Value,
			Upper: rules[i].Upper,
			Tone:  string(rules[i].Tone),
		})
	}
	return models
}

func reportDisplayToModel(display *reportfmt.Resolved) *gqlmodel.ReportColumnDisplay {
	return &gqlmodel.ReportColumnDisplay{
		Style:         string(display.Style),
		Decimals:      display.Decimals,
		Grouping:      display.Grouping,
		Currency:      display.Currency,
		Negative:      string(display.Negative),
		Notation:      string(display.Notation),
		Prefix:        display.Prefix,
		Suffix:        display.Suffix,
		DateStyle:     string(display.DateStyle),
		BoolStyle:     string(display.BoolStyle),
		DurationUnit:  string(display.DurationUnit),
		DurationStyle: string(display.DurationStyle),
		NullText:      display.NullText,
		Rules:         reportDisplayRulesToModel(display.Rules),
		Band:          reportBandToModel(display.Band),
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func reportFieldRefFromGraphQL(input *gqlmodel.ReportFieldRefInput) report.FieldRef {
	if input == nil {
		return report.FieldRef{}
	}
	return report.FieldRef{Path: input.Path, Field: input.Field}
}

func reportFilterGroupFromGraphQL(input *gqlmodel.ReportFilterGroupInput) *report.FilterGroup {
	if input == nil {
		return nil
	}

	group := &report.FilterGroup{Op: report.BoolOp(input.Op)}
	for _, filter := range input.Filters {
		fieldFilter := report.FieldFilter{
			Ref:      reportFieldRefFromGraphQL(filter.Ref),
			Operator: dbtype.Operator(filter.Operator),
			Value:    filter.Value,
		}
		if filter.Param != nil {
			fieldFilter.Param = *filter.Param
		}
		if filter.Agg != nil {
			fieldFilter.Agg = reportcatalog.Aggregation(*filter.Agg)
		}
		fieldFilter.Transform = reportTransformFromGraphQL(filter.Transform)
		group.Filters = append(group.Filters, fieldFilter)
	}
	for _, nested := range input.Groups {
		if child := reportFilterGroupFromGraphQL(nested); child != nil {
			group.Groups = append(group.Groups, *child)
		}
	}

	return group
}

func reportDefinitionAsJSON(def *report.Definition) (map[string]any, error) {
	raw, err := sonic.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("serialize report definition: %w", err)
	}
	var out map[string]any
	if err = sonic.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("deserialize report definition: %w", err)
	}
	return out, nil
}

func reportDefinitionToModel(
	entity *report.ReportDefinition,
) (*gqlmodel.ReportDefinition, error) {
	definitionJSON, err := reportDefinitionAsJSON(entity.Definition)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ReportDefinition{
		ID:              entity.ID.String(),
		Name:            entity.Name,
		Description:     entity.Description,
		Category:        entity.Category,
		Tags:            emptyIfNil(entity.Tags),
		Kind:            string(entity.Kind),
		CannedKey:       nilIfEmpty(entity.CannedKey),
		CannedVersion:   nilIfEmpty(entity.CannedVersion),
		OwnerID:         entity.OwnerID.String(),
		Visibility:      string(entity.Visibility),
		Status:          string(entity.Status),
		Diagnostics:     emptyIfNil(entity.Diagnostics),
		CatalogVersion:  entity.CatalogVersion,
		Definition:      definitionJSON,
		DefaultFormat:   string(entity.DefaultFormat),
		CurrentRevision: int(entity.CurrentRevision),
		LastRunAt:       nilIfZero(entity.LastRunAt),
		Version:         int(entity.Version),
		CreatedAt:       int(entity.CreatedAt),
		UpdatedAt:       int(entity.UpdatedAt),
	}, nil
}

func reportRevisionToModel(
	entity *report.ReportDefinitionRevision,
) (*gqlmodel.ReportDefinitionRevision, error) {
	definitionJSON, err := reportDefinitionAsJSON(entity.Definition)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ReportDefinitionRevision{
		ID:             entity.ID.String(),
		DefinitionID:   entity.DefinitionID.String(),
		RevisionNumber: int(entity.RevisionNumber),
		CatalogVersion: entity.CatalogVersion,
		Definition:     definitionJSON,
		CreatedByID:    entity.CreatedByID.String(),
		CreatedAt:      int(entity.CreatedAt),
	}, nil
}

func reportRunToModel(entity *report.ReportRun) *gqlmodel.ReportRun {
	model := &gqlmodel.ReportRun{
		ID:                entity.ID.String(),
		CannedKey:         nilIfEmpty(entity.CannedKey),
		CannedVersion:     nilIfEmpty(entity.CannedVersion),
		RequestedByID:     entity.RequestedByID.String(),
		Trigger:           string(entity.Trigger),
		Params:            entity.Params,
		Format:            string(entity.Format),
		Status:            string(entity.Status),
		RowCount:          int(entity.RowCount),
		ByteSize:          int(entity.ByteSize),
		DurationMs:        int(entity.DurationMs),
		Truncated:         entity.Truncated,
		ArtifactExpiresAt: nilIfZero(entity.ArtifactExpiresAt),
		CacheHit:          entity.CacheHit,
		QueuedAt:          nilIfZero(entity.QueuedAt),
		StartedAt:         nilIfZero(entity.StartedAt),
		CompletedAt:       nilIfZero(entity.CompletedAt),
		Version:           int(entity.Version),
		CreatedAt:         int(entity.CreatedAt),
	}

	if !entity.DefinitionID.IsNil() {
		definitionID := entity.DefinitionID.String()
		model.DefinitionID = &definitionID
	}
	if !entity.RevisionID.IsNil() {
		revisionID := entity.RevisionID.String()
		model.RevisionID = &revisionID
	}
	if entity.Error != nil {
		model.Error = &gqlmodel.ReportRunError{
			Code:    entity.Error.Code,
			Message: entity.Error.Message,
			Detail:  nilIfEmpty(entity.Error.Detail),
		}
	}

	return model
}

func reportScheduleToModel(entity *report.ReportSchedule) *gqlmodel.ReportSchedule {
	model := &gqlmodel.ReportSchedule{
		ID:                  entity.ID.String(),
		DefinitionID:        entity.DefinitionID.String(),
		CronExpression:      entity.CronExpression,
		Timezone:            entity.Timezone,
		Formats:             emptyIfNil(entity.Formats),
		EmailRecipients:     []string{},
		NotifyUserIds:       []string{},
		Enabled:             entity.Enabled,
		RunAsID:             entity.RunAsID.String(),
		NextRunAt:           nilIfZero(entity.NextRunAt),
		ConsecutiveFailures: entity.ConsecutiveFailures,
		AlertFiring:         entity.AlertFiring,
		Alert:               reportScheduleAlertToModel(entity.Alert),
		Version:             int(entity.Version),
		CreatedAt:           int(entity.CreatedAt),
		UpdatedAt:           int(entity.UpdatedAt),
	}

	if !entity.LastRunID.IsNil() {
		lastRunID := entity.LastRunID.String()
		model.LastRunID = &lastRunID
	}
	if entity.Delivery != nil {
		model.EmailRecipients = emptyIfNil(entity.Delivery.EmailRecipients)
		model.EmailAttach = entity.Delivery.EmailAttach
		model.EmailInline = entity.Delivery.EmailInline
		model.NotifyUserIds = pulidsToStrings(entity.Delivery.NotifyUserIDs)
	}

	return model
}

func reportScheduleAlertToModel(alert *report.ScheduleAlert) *gqlmodel.ReportScheduleAlert {
	if alert == nil {
		return nil
	}
	return &gqlmodel.ReportScheduleAlert{
		Operator:            string(alert.Operator),
		Threshold:           int(alert.Threshold),
		ColumnID:            nilIfEmpty(alert.ColumnID),
		Value:               alert.Value,
		SuppressWhileFiring: alert.SuppressWhileFiring,
	}
}

func reportScheduleAlertFromGraphQL(
	input *gqlmodel.ReportScheduleAlertInput,
) *report.ScheduleAlert {
	if input == nil {
		return nil
	}
	alert := &report.ScheduleAlert{
		Operator:  dbtype.Operator(input.Operator),
		Threshold: int64(input.Threshold),
		ColumnID:  derefString(input.ColumnID),
		Value:     input.Value,
	}
	if input.SuppressWhileFiring != nil {
		alert.SuppressWhileFiring = *input.SuppressWhileFiring
	}
	return alert
}

func pulidsToStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func parsePulids(field string, raw []string) ([]pulid.ID, error) {
	out := make([]pulid.ID, 0, len(raw))
	for i, value := range raw {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("%s[%d]", field, i), errortypes.ErrInvalid, "Invalid identifier",
			)
		}
		out = append(out, id)
	}
	return out, nil
}

func saveReportScheduleRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.CreateReportScheduleInput,
	update *gqlmodel.UpdateReportScheduleInput,
) (*reportingservice.SaveScheduleRequest, error) {
	definitionID, err := pulid.MustParse(input.DefinitionID)
	if err != nil {
		return nil, err
	}

	notifyUserIDs, err := parsePulids("notifyUserIds", input.NotifyUserIds)
	if err != nil {
		return nil, err
	}

	req := &reportingservice.SaveScheduleRequest{
		Request:         reportingRequest(authCtx),
		DefinitionID:    definitionID,
		CronExpression:  input.CronExpression,
		Formats:         input.Formats,
		EmailRecipients: input.EmailRecipients,
		NotifyUserIDs:   notifyUserIDs,
		Alert:           reportScheduleAlertFromGraphQL(input.Alert),
		Enabled:         input.Enabled,
	}
	if input.EmailAttach != nil {
		req.EmailAttach = *input.EmailAttach
	}
	if input.EmailInline != nil {
		req.EmailInline = *input.EmailInline
	}
	if input.Timezone != nil {
		req.Timezone = *input.Timezone
	}

	if update != nil {
		scheduleID, parseErr := pulid.MustParse(update.ID)
		if parseErr != nil {
			return nil, parseErr
		}
		req.ScheduleID = scheduleID
		req.Version = int64(update.Version)
	}

	return req, nil
}

func reportDashboardToModel(entity *report.Dashboard) (*gqlmodel.ReportDashboard, error) {
	raw, err := sonic.Marshal(entity.Layout)
	if err != nil {
		return nil, fmt.Errorf("serialize dashboard layout: %w", err)
	}
	var layout map[string]any
	if err = sonic.Unmarshal(raw, &layout); err != nil {
		return nil, fmt.Errorf("deserialize dashboard layout: %w", err)
	}

	return &gqlmodel.ReportDashboard{
		ID:          entity.ID.String(),
		Name:        entity.Name,
		Description: entity.Description,
		Category:    entity.Category,
		Tags:        emptyIfNil(entity.Tags),
		OwnerID:     entity.OwnerID.String(),
		Visibility:  string(entity.Visibility),
		Layout:      layout,
		Version:     int(entity.Version),
		CreatedAt:   int(entity.CreatedAt),
		UpdatedAt:   int(entity.UpdatedAt),
	}, nil
}

func reportDashboardLayoutFromGraphQL(
	input *gqlmodel.ReportDashboardLayoutInput,
) (*report.DashboardLayout, error) {
	layout := &report.DashboardLayout{
		Tiles: make([]report.DashboardTile, 0, len(input.Tiles)),
	}

	for _, tile := range input.Tiles {
		converted, err := reportDashboardTileFromGraphQL(tile)
		if err != nil {
			return nil, err
		}
		layout.Tiles = append(layout.Tiles, converted)
	}

	for _, param := range input.Parameters {
		layout.Parameters = append(layout.Parameters, reportParameterFromGraphQL(param))
	}

	for _, filter := range input.Filters {
		layout.Filters = append(layout.Filters, report.DashboardFilter{
			ID:       filter.ID,
			Label:    derefString(filter.Label),
			Entity:   filter.Entity,
			Ref:      reportFieldRefFromGraphQL(filter.Ref),
			Operator: dbtype.Operator(filter.Operator),
			Default:  filter.Default,
		})
	}

	return layout, nil
}

func reportDashboardTileFromGraphQL(
	input *gqlmodel.ReportDashboardTileInput,
) (report.DashboardTile, error) {
	tile := report.DashboardTile{
		ID:        input.ID,
		Kind:      report.TileKind(input.Kind),
		Title:     derefString(input.Title),
		CannedKey: derefString(input.CannedKey),
		ChartID:   derefString(input.ChartID),
		ColumnID:  derefString(input.ColumnID),
		Text:      derefString(input.Text),
		X:         input.X,
		Y:         input.Y,
		W:         input.W,
		H:         input.H,
	}
	if input.Limit != nil {
		tile.Limit = *input.Limit
	}
	if input.DefinitionID != nil && *input.DefinitionID != "" {
		definitionID, err := pulid.MustParse(*input.DefinitionID)
		if err != nil {
			return report.DashboardTile{}, errortypes.NewValidationError(
				"layout.tiles.definitionId", errortypes.ErrInvalid, "Invalid report identifier",
			)
		}
		tile.DefinitionID = definitionID
	}
	if len(input.ParamBindings) > 0 {
		tile.ParamBindings = make(map[string]string, len(input.ParamBindings))
		for name, bound := range input.ParamBindings {
			if text, ok := bound.(string); ok {
				tile.ParamBindings[name] = text
			}
		}
	}

	return tile, nil
}

func saveReportDashboardRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.SaveReportDashboardInput,
) (*reportingservice.SaveDashboardRequest, error) {
	layout, err := reportDashboardLayoutFromGraphQL(input.Layout)
	if err != nil {
		return nil, err
	}

	req := &reportingservice.SaveDashboardRequest{
		Request:     reportingRequest(authCtx),
		Name:        input.Name,
		Description: derefString(input.Description),
		Category:    derefString(input.Category),
		Tags:        input.Tags,
		Layout:      layout,
	}
	if input.Visibility != nil {
		req.Visibility = report.Visibility(*input.Visibility)
	}

	return req, nil
}

func cannedReportToModel(entry *canned.Entry) (*gqlmodel.CannedReport, error) {
	definitionJSON, err := reportDefinitionAsJSON(entry.Definition)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CannedReport{
		Key:           entry.Key,
		Version:       entry.Version,
		Name:          entry.Name,
		Description:   entry.Description,
		Category:      entry.Category,
		Tags:          emptyIfNil(entry.Tags),
		DefaultFormat: string(entry.DefaultFormat),
		Definition:    definitionJSON,
	}, nil
}

func reportDefinitionConnectionToModel(
	result *pagination.CursorListResult[*report.ReportDefinition],
) (*gqlmodel.ReportDefinitionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *report.ReportDefinition, cursor string) *gqlmodel.ReportDefinitionEdge {
			model, mapErr := reportDefinitionToModel(node)
			if mapErr != nil {
				model = &gqlmodel.ReportDefinition{ID: node.ID.String(), Name: node.Name}
			}
			return &gqlmodel.ReportDefinitionEdge{Node: model, Cursor: cursor}
		},
		func(edge *gqlmodel.ReportDefinitionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ReportDefinitionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: derefInt(page.TotalCount),
	}, nil
}

func reportRunConnectionToModel(
	result *pagination.CursorListResult[*report.ReportRun],
) (*gqlmodel.ReportRunConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *report.ReportRun, cursor string) *gqlmodel.ReportRunEdge {
			return &gqlmodel.ReportRunEdge{Node: reportRunToModel(node), Cursor: cursor}
		},
		func(edge *gqlmodel.ReportRunEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ReportRunConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: derefInt(page.TotalCount),
	}, nil
}

func reportCatalogToModel(
	ctx context.Context,
	engine services.PermissionEngine,
	tenant pagination.TenantInfo,
) (*gqlmodel.ReportCatalog, error) {
	catalog := &reportcatalog.Default

	entities := make([]*gqlmodel.ReportCatalogEntity, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity := &catalog.Entities[i]

		detail, err := engine.GetResourcePermissions(
			ctx, tenant.UserID, tenant.OrgID, entity.Resource.String(),
		)
		if err != nil {
			return nil, fmt.Errorf("resolve permissions for %q: %w", entity.Resource, err)
		}
		if !hasReportReadOperation(detail) {
			continue
		}

		entities = append(entities, catalogEntityToModel(entity, detail))
	}

	return &gqlmodel.ReportCatalog{
		Version:  catalog.Version,
		Entities: entities,
	}, nil
}

func hasReportReadOperation(detail *services.ResourcePermissionDetail) bool {
	for _, op := range detail.Operations {
		if op == permission.OpRead {
			return true
		}
	}
	return false
}

func catalogEntityToModel(
	entity *reportcatalog.Entity,
	detail *services.ResourcePermissionDetail,
) *gqlmodel.ReportCatalogEntity {
	fields := make([]*gqlmodel.ReportCatalogField, 0, len(entity.Fields))
	for i := range entity.Fields {
		field := &entity.Fields[i]
		sensitivity := reportSensitivityRegistry.GetFieldSensitivity(
			entity.Resource.String(), field.Key,
		)

		accessible := detail.MaxSensitivity.CanAccess(sensitivity)
		if accessible && len(detail.AccessibleFields) > 0 {
			accessible = containsString(detail.AccessibleFields, field.Key)
		}

		enumValues := make([]*gqlmodel.ReportCatalogEnumValue, 0, len(field.EnumValues))
		for _, ev := range field.EnumValues {
			enumValues = append(enumValues, &gqlmodel.ReportCatalogEnumValue{
				Value: ev.Value,
				Label: ev.Label,
			})
		}

		aggregations := make([]string, 0, len(field.Aggregations))
		for _, agg := range field.Aggregations {
			aggregations = append(aggregations, string(agg))
		}

		fields = append(fields, &gqlmodel.ReportCatalogField{
			Key:          field.Key,
			Label:        field.Label,
			Description:  nilIfEmpty(field.Description),
			Type:         string(field.Type),
			Format:       nilIfEmpty(string(field.Format)),
			Nullable:     field.Nullable,
			EnumValues:   enumValues,
			Aggregations: aggregations,
			Filterable:   field.Filterable,
			Groupable:    field.Groupable,
			Accessible:   accessible,
			Sensitivity:  sensitivity.String(),
		})
	}

	edges := make([]*gqlmodel.ReportCatalogEdge, 0, len(entity.Edges))
	for i := range entity.Edges {
		edge := &entity.Edges[i]
		edges = append(edges, &gqlmodel.ReportCatalogEdge{
			Name:        edge.Name,
			Label:       edge.Label,
			Target:      edge.Target,
			Cardinality: string(edge.Cardinality),
			Traversable: edge.Traversable,
		})
	}

	return &gqlmodel.ReportCatalogEntity{
		Key:               entity.Key,
		Resource:          entity.Resource.String(),
		Label:             entity.Label,
		PluralLabel:       entity.PluralLabel,
		Description:       nilIfEmpty(entity.Description),
		Category:          entity.Category,
		OwnScopeSupported: entity.OwnershipColumn != "",
		Fields:            fields,
		Edges:             edges,
	}
}

func reportPreviewToModel(result *reportingservice.PreviewResult) *gqlmodel.ReportPreview {
	columns := make([]*gqlmodel.ReportPreviewColumn, 0, len(result.Columns))
	for i := range result.Columns {
		col := &result.Columns[i]
		columns = append(columns, &gqlmodel.ReportPreviewColumn{
			ID:      col.ID,
			Label:   col.Label,
			Type:    string(col.Type),
			Format:  nilIfEmpty(string(col.Format)),
			Display: reportDisplayToModel(&col.Display),
		})
	}

	rows := make([]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, encodeReportRow(row))
	}

	preview := &gqlmodel.ReportPreview{
		Columns:   columns,
		Rows:      rows,
		Truncated: result.Truncated,
	}
	if result.Totals != nil {
		preview.Totals = encodeReportRow(result.Totals)
	}

	return preview
}

// encodeReportRow keeps decimals lossless across the JSON boundary: a float64
// would quietly round money the formatter is about to render.
func encodeReportRow(row services.ReportRow) []any {
	encoded := make([]any, len(row))
	for i, value := range row {
		if dec, ok := value.(decimal.Decimal); ok {
			encoded[i] = dec.String()
			continue
		}
		encoded[i] = value
	}
	return encoded
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfZero(v int64) *int {
	if v == 0 {
		return nil
	}
	out := int(v)
	return &out
}

func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func reportingRequest(authCtx *authctx.AuthContext) reportingservice.Request {
	return reportingservice.Request{
		TenantInfo: tenantInfo(authCtx),
		Principal: services.PrincipalInfo{
			Type:     services.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
	}
}

func saveReportDefinitionRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.SaveReportDefinitionInput,
) *reportingservice.SaveDefinitionRequest {
	req := &reportingservice.SaveDefinitionRequest{
		Request:    reportingRequest(authCtx),
		Name:       input.Name,
		Definition: reportIRFromGraphQL(input.Definition),
	}
	if input.Description != nil {
		req.Description = *input.Description
	}
	if input.Category != nil {
		req.Category = *input.Category
	}
	req.Tags = input.Tags
	if input.Visibility != nil {
		req.Visibility = report.Visibility(*input.Visibility)
	}
	if input.Status != nil {
		req.Status = report.DefinitionStatus(*input.Status)
	}
	if input.DefaultFormat != nil {
		req.DefaultFormat = report.Format(*input.DefaultFormat)
	}
	return req
}

func saveReportViewRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.CreateReportViewInput,
	update *gqlmodel.UpdateReportViewInput,
) (*reportingservice.SaveViewRequest, error) {
	req := &reportingservice.SaveViewRequest{
		Request:     reportingRequest(authCtx),
		Name:        input.Name,
		Description: derefString(input.Description),
		Params:      input.Params,
		Shared:      boolValue(input.Shared),
		Pinned:      boolValue(input.Pinned),
		Format:      report.Format(derefString(input.Format)),
	}

	// A create carries the report it answers; an update never does, because the
	// service pins it to what the stored view already points at.
	if update == nil {
		definitionID, err := pulid.MustParse(input.DefinitionID)
		if err != nil {
			return nil, err
		}
		req.DefinitionID = definitionID
		return req, nil
	}

	viewID, err := pulid.MustParse(update.ID)
	if err != nil {
		return nil, err
	}
	req.ViewID = viewID
	req.Version = int64(update.Version)

	return req, nil
}

func reportViewToModel(entity *report.ReportView) *gqlmodel.ReportView {
	return &gqlmodel.ReportView{
		ID:           entity.ID.String(),
		DefinitionID: entity.DefinitionID.String(),
		OwnerID:      entity.OwnerID.String(),
		Name:         entity.Name,
		Description:  nilIfEmpty(entity.Description),
		Params:       entity.Params,
		Shared:       entity.Shared,
		Pinned:       entity.Pinned,
		Format:       nilIfEmpty(string(entity.Format)),
		LastRunAt:    nilIfZero(entity.LastRunAt),
		RunCount:     int(entity.RunCount),
		Version:      int(entity.Version),
		CreatedAt:    int(entity.CreatedAt),
		UpdatedAt:    int(entity.UpdatedAt),
	}
}

// applyReportView resolves a run against a saved view. GetView enforces both
// the view's own visibility and read access to the report it points at, so a
// caller cannot reach a report through a view they could not open directly.
func (r *mutationResolver) applyReportView(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	viewID *string,
	req *reportingservice.RunReportRequest,
) (*reportingservice.GetViewRequest, error) {
	if viewID == nil {
		return nil, nil //nolint:nilnil // no view is a valid, unremarkable case
	}

	parsedID, err := pulid.MustParse(*viewID)
	if err != nil {
		return nil, err
	}

	viewRequest := &reportingservice.GetViewRequest{
		Request: reportingRequest(authCtx),
		ViewID:  parsedID,
	}
	view, err := r.reportingService.GetView(ctx, viewRequest)
	if err != nil {
		return nil, err
	}

	req.DefinitionID = view.DefinitionID
	req.CannedKey = ""
	if len(req.Params) == 0 {
		req.Params = view.Params
	}
	if req.Format == "" && view.Format != "" {
		req.Format = view.Format
	}

	return viewRequest, nil
}
