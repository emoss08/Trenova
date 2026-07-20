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
		if col.Label != nil {
			spec.Label = *col.Label
		}
		if col.Computed != nil {
			computed := &report.ComputedSpec{
				Op:      report.ComputedOp(col.Computed.Op),
				LeftID:  col.Computed.LeftID,
				RightID: col.Computed.RightID,
			}
			if col.Computed.Format != nil {
				computed.Format = reportcatalog.FormatHint(*col.Computed.Format)
			}
			spec.Computed = computed
		}
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
			MeasureIDs: input.Pivot.MeasureIds,
		}
		if input.Pivot.IncludeOther != nil {
			pivot.IncludeOther = *input.Pivot.IncludeOther
		}
		def.Pivot = pivot
	}

	for _, param := range input.Parameters {
		paramDef := report.ParameterDef{
			Name: param.Name,
			Type: reportcatalog.FieldType(param.Type),
		}
		if param.Label != nil {
			paramDef.Label = *param.Label
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
		paramDef.AllowedValues = param.AllowedValues
		if param.RefEntity != nil {
			paramDef.RefEntity = *param.RefEntity
		}
		def.Parameters = append(def.Parameters, paramDef)
	}

	return def
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
		model.NotifyUserIds = pulidsToStrings(entity.Delivery.NotifyUserIDs)
	}

	return model
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
		Enabled:         input.Enabled,
	}
	if input.EmailAttach != nil {
		req.EmailAttach = *input.EmailAttach
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
			ID:     col.ID,
			Label:  col.Label,
			Type:   string(col.Type),
			Format: nilIfEmpty(string(col.Format)),
		})
	}

	rows := make([]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		encoded := make([]any, len(row))
		for i, value := range row {
			if dec, ok := value.(decimal.Decimal); ok {
				encoded[i] = dec.String()
				continue
			}
			encoded[i] = value
		}
		rows = append(rows, encoded)
	}

	return &gqlmodel.ReportPreview{
		Columns:   columns,
		Rows:      rows,
		Truncated: result.Truncated,
	}
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
