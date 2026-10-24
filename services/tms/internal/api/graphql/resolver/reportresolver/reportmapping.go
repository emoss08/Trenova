package reportresolver

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/formula"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func reportDefinitionAsJSON(def *ir.Definition) (map[string]any, error) {
	raw, err := ir.Encode(def)
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
		RowsStored:        entity.RowsKey != "",
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
		model.NotifyUserIds = base.PulidsToStrings(entity.Delivery.NotifyUserIDs)
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
		ColumnID:  base.DerefString(input.ColumnID),
		Value:     input.Value,
	}
	if input.SuppressWhileFiring != nil {
		alert.SuppressWhileFiring = *input.SuppressWhileFiring
	}
	return alert
}

func saveReportScheduleRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.CreateReportScheduleInput,
) (*reportingservice.SaveScheduleRequest, error) {
	definitionID, err := pulid.MustParse(input.DefinitionID)
	if err != nil {
		return nil, err
	}

	notifyUserIDs, err := base.ParsePulids("notifyUserIds", input.NotifyUserIds)
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

	return req, nil
}

func createReportScheduleInputOf(
	input *gqlmodel.UpdateReportScheduleInput,
) *gqlmodel.CreateReportScheduleInput {
	return &gqlmodel.CreateReportScheduleInput{
		DefinitionID:    input.DefinitionID,
		CronExpression:  input.CronExpression,
		Timezone:        input.Timezone,
		Formats:         input.Formats,
		EmailRecipients: input.EmailRecipients,
		EmailAttach:     input.EmailAttach,
		EmailInline:     input.EmailInline,
		NotifyUserIds:   input.NotifyUserIds,
		Alert:           input.Alert,
		Enabled:         input.Enabled,
	}
}

func updateReportScheduleRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.UpdateReportScheduleInput,
) (*reportingservice.SaveScheduleRequest, error) {
	req, err := saveReportScheduleRequest(authCtx, createReportScheduleInputOf(input))
	if err != nil {
		return nil, err
	}

	scheduleID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	req.ScheduleID = scheduleID
	req.Version = int64(input.Version)

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
		layout.Filters = append(layout.Filters, ir.DashboardFilter{
			ID:      filter.ID,
			Label:   base.DerefString(filter.Label),
			Entity:  filter.Entity,
			Field:   filter.Field,
			Op:      ir.Op(filter.Op),
			Default: filter.Default,
		})
	}

	return layout, nil
}

func reportParameterFromGraphQL(input *gqlmodel.ReportParameterInput) ir.Parameter {
	return ir.Parameter{
		Name:     input.Name,
		Label:    base.DerefString(input.Label),
		Type:     formula.Type(input.Type),
		Required: base.BoolValue(input.Required),
		Multi:    base.BoolValue(input.Multi),
		Default:  input.Default,
		Allowed:  input.Allowed,
		Entity:   base.DerefString(input.Entity),
	}
}

func reportDashboardTileFromGraphQL(
	input *gqlmodel.ReportDashboardTileInput,
) (report.DashboardTile, error) {
	tile := report.DashboardTile{
		ID:        input.ID,
		Kind:      report.TileKind(input.Kind),
		Title:     base.DerefString(input.Title),
		CannedKey: base.DerefString(input.CannedKey),
		ChartID:   base.DerefString(input.ChartID),
		ColumnID:  base.DerefString(input.ColumnID),
		Text:      base.DerefString(input.Text),
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
		Description: base.DerefString(input.Description),
		Category:    base.DerefString(input.Category),
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
	page, err := base.EntityCursorConnection(
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
		TotalCount: base.DerefInt(page.TotalCount),
	}, nil
}

func reportRunConnectionToModel(
	result *pagination.CursorListResult[*report.ReportRun],
) (*gqlmodel.ReportRunConnection, error) {
	page, err := base.EntityCursorConnection(
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
		TotalCount: base.DerefInt(page.TotalCount),
	}, nil
}

func reportCatalogToModel(
	ctx context.Context,
	catalog *reportingservice.Catalog,
) (*gqlmodel.ReportCatalog, error) {
	rules, err := reportRulesAsJSON(catalog.Rules)
	if err != nil {
		return nil, err
	}
	entities := make([]*gqlmodel.ReportCatalogEntity, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity := &catalog.Entities[i]
		if !entity.Usable() {
			continue
		}
		entities = append(entities, catalogEntityToModel(ctx, catalog.Kinds, entity))
	}

	return &gqlmodel.ReportCatalog{
		Version:  catalog.Version,
		Rules:    rules,
		Entities: entities,
	}, nil
}

func reportRulesAsJSON(rules analytics.RuleSet) (map[string]any, error) {
	raw, err := sonic.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("serialize report rules: %w", err)
	}
	var out map[string]any
	if err = sonic.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("deserialize report rules: %w", err)
	}
	return out, nil
}

func reportFieldAccessToModel(decision reportaccess.Decision) gqlmodel.ReportFieldAccess {
	switch decision {
	case reportaccess.Available:
		return gqlmodel.ReportFieldAccessAvailable
	case reportaccess.WithheldByCeiling:
		return gqlmodel.ReportFieldAccessWithheldByCeiling
	case reportaccess.NeedsFieldGroup:
		return gqlmodel.ReportFieldAccessNeedsFieldGroup
	case reportaccess.NeedsKindRead,
		reportaccess.NeedsExport,
		reportaccess.OwnScopeUnsupported:
		return gqlmodel.ReportFieldAccessNeedsKindRead
	default:
		return gqlmodel.ReportFieldAccessNeedsKindRead
	}
}

func catalogKindToModel(
	ctx context.Context,
	kinds *reportaccess.Resolver,
	resource permission.Resource,
) *gqlmodel.ReportCatalogKind {
	if resource == "" || kinds == nil {
		return nil
	}
	return &gqlmodel.ReportCatalogKind{
		Resource: resource.String(),
		Label:    kinds.KindLabel(ctx, resource),
	}
}

func catalogEntityToModel(
	ctx context.Context,
	kinds *reportaccess.Resolver,
	decided *reportingservice.CatalogEntity,
) *gqlmodel.ReportCatalogEntity {
	view := &decided.View
	fields := make([]*gqlmodel.ReportCatalogField, 0, len(view.Fields))
	for i := range view.Fields {
		fields = append(fields, catalogFieldToModel(&view.Fields[i], decided.Fields[i]))
	}

	edges := make([]*gqlmodel.ReportCatalogEdge, 0, len(view.Edges))
	for i := range view.Edges {
		edge := &view.Edges[i]
		use := decided.Edges[i]
		edges = append(edges, &gqlmodel.ReportCatalogEdge{
			Name:            edge.Name,
			Label:           edge.Label,
			Description:     nilIfEmpty(edge.Doc),
			Target:          edge.Target,
			ToMany:          edge.ToMany,
			Pick:            edge.Pick,
			Access:          reportFieldAccessToModel(use.Decision),
			InReferenceView: use.Reference,
			NeedsReadOf:     catalogKindToModel(ctx, kinds, use.Kind),
		})
	}

	return &gqlmodel.ReportCatalogEntity{
		Key:               view.Key,
		Resource:          decided.Resource.String(),
		Label:             view.Label,
		PluralLabel:       view.Plural,
		Description:       nilIfEmpty(view.Doc),
		Category:          view.Category,
		OwnScopeSupported: decided.OwnScopeSupported,
		Readable:          decided.Readable,
		Exportable:        decided.Exportable,
		Fields:            fields,
		Edges:             edges,
	}
}

func catalogFieldToModel(
	field *analytics.FieldView,
	use reportaccess.Use,
) *gqlmodel.ReportCatalogField {
	enumValues := make([]*gqlmodel.ReportCatalogEnumValue, 0, len(field.Enum))
	for _, ev := range field.Enum {
		enumValues = append(enumValues, &gqlmodel.ReportCatalogEnumValue{
			Value: ev.Value,
			Label: ev.Label,
		})
	}

	aggregations := make([]string, 0, len(field.Aggs))
	for _, agg := range field.Aggs {
		aggregations = append(aggregations, string(agg))
	}

	operators := make([]string, 0, len(field.Operators))
	for _, op := range field.Operators {
		operators = append(operators, string(op))
	}

	var protection *gqlmodel.ReportFieldProtection
	if use.Group != "" {
		protection = &gqlmodel.ReportFieldProtection{Group: use.Group, Held: use.HoldsGroup}
	}

	return &gqlmodel.ReportCatalogField{
		Key:             field.Key,
		Label:           field.Label,
		Description:     nilIfEmpty(field.Doc),
		Type:            string(field.Type),
		Format:          nilIfEmpty(field.Format),
		Measure:         field.Measure,
		EnumValues:      enumValues,
		Aggregations:    aggregations,
		Operators:       operators,
		Filterable:      field.Filterable,
		Groupable:       field.Groupable,
		Access:          reportFieldAccessToModel(use.Decision),
		InReferenceView: use.Reference,
		Protection:      protection,
		Sensitivity:     use.Sensitivity.String(),
	}
}

func reportPreviewToModel(result *reportingservice.PreviewResult) *gqlmodel.ReportPreview {
	columns := make([]*gqlmodel.ReportPreviewColumn, 0, len(result.Columns))
	for i := range result.Columns {
		col := &result.Columns[i]
		columns = append(columns, &gqlmodel.ReportPreviewColumn{
			ID:          col.ID,
			Label:       col.Label,
			Type:        col.Type.String(),
			Display:     reportDisplayToModel(&col.Display),
			Sensitivity: nilIfEmpty(string(col.Sensitivity)),
		})
	}

	rows := make([]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, row)
	}

	levels := make([]int, 0, len(result.Levels))
	for _, level := range result.Levels {
		levels = append(levels, int(level))
	}

	preview := &gqlmodel.ReportPreview{
		Columns:   columns,
		Rows:      rows,
		Levels:    levels,
		Truncated: result.Truncated,
		Cursor:    nilIfEmpty(result.Cursor),
		Timezone:  result.Timezone,
		Rollup:    nilIfEmpty(result.Rollup),
		Cached:    result.Cached,
	}
	if result.Totals != nil {
		preview.Totals = result.Totals
	}

	return preview
}

func reportDisplayToModel(display *format.Resolved) *gqlmodel.ReportColumnDisplay {
	rules := make([]*gqlmodel.ReportDisplayRule, 0, len(display.Rules))
	for _, rule := range display.Rules {
		rules = append(rules, &gqlmodel.ReportDisplayRule{
			Op:    string(rule.Op),
			Value: rule.Value,
			Upper: rule.Upper,
			Tone:  string(rule.Tone),
		})
	}

	model := &gqlmodel.ReportColumnDisplay{
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
		Rules:         rules,
	}
	if display.Band != nil {
		edges := display.Band.Edges
		if edges == nil {
			edges = []float64{}
		}
		model.Band = &gqlmodel.ReportBand{Width: display.Band.Width, Edges: edges}
	}
	if len(display.Labels) > 0 {
		labels := make(map[string]any, len(display.Labels))
		for value, label := range display.Labels {
			labels[value] = label
		}
		model.Labels = labels
	}
	return model
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

func reportingRequest(authCtx *authctx.AuthContext) reportingservice.Request {
	return reportingservice.Request{
		TenantInfo: base.TenantInfo(authCtx),
		Principal: kernelports.PrincipalInfo{
			Type:     kernelports.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
	}
}

func saveReportDefinitionRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.SaveReportDefinitionInput,
) (*reportingservice.SaveDefinitionRequest, error) {
	definition, err := reportingservice.DecodeDefinition(map[string]any(input.Definition))
	if err != nil {
		return nil, err
	}
	req := &reportingservice.SaveDefinitionRequest{
		Request:    reportingRequest(authCtx),
		Name:       input.Name,
		Definition: definition,
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
	return req, nil
}

func saveReportViewRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.CreateReportViewInput,
	update *gqlmodel.UpdateReportViewInput,
) (*reportingservice.SaveViewRequest, error) {
	req := &reportingservice.SaveViewRequest{
		Request:     reportingRequest(authCtx),
		Name:        input.Name,
		Description: base.DerefString(input.Description),
		Params:      input.Params,
		Shared:      base.BoolValue(input.Shared),
		Pinned:      base.BoolValue(input.Pinned),
		Format:      report.Format(base.DerefString(input.Format)),
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
func (r *MutationResolver) applyReportView(
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
	view, err := r.ReportingService.GetView(ctx, viewRequest)
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
