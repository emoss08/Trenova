package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func homeLayoutRequest(authCtx *authctx.AuthContext) homelayoutservice.Request {
	return homelayoutservice.Request{
		TenantInfo: tenantInfo(authCtx),
		Principal: services.PrincipalInfo{
			Type:     services.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
	}
}

var homeLayoutSources = map[homelayoutservice.Source]gqlmodel.HomeLayoutSource{
	homelayoutservice.SourceUser:       gqlmodel.HomeLayoutSourceUser,
	homelayoutservice.SourceRolePreset: gqlmodel.HomeLayoutSourceRolePreset,
	homelayoutservice.SourceOrgDefault: gqlmodel.HomeLayoutSourceOrgDefault,
	homelayoutservice.SourceBuiltIn:    gqlmodel.HomeLayoutSourceBuiltIn,
}

func homeLayoutToModel(effective *homelayoutservice.EffectiveLayout) *gqlmodel.HomeLayout {
	source, ok := homeLayoutSources[effective.Source]
	if !ok {
		source = gqlmodel.HomeLayoutSourceBuiltIn
	}

	return &gqlmodel.HomeLayout{
		SchemaVersion: homelayout.DocumentSchemaVersion,
		Version:       int(effective.Version),
		Source:        source,
		PresetID:      homeOptionalID(effective.PresetID),
		PresetName:    homeOptionalString(effective.PresetName),
		Locked:        effective.Locked,
		CanCustomize:  effective.CanCustomize,
		Density:       effective.Density,
		Widgets:       homeWidgetsToModel(effective.Layout),
	}
}

func homeWidgetsToModel(layout *homelayout.Layout) []*gqlmodel.HomeWidget {
	if layout == nil {
		return []*gqlmodel.HomeWidget{}
	}

	widgets := make([]*gqlmodel.HomeWidget, 0, len(layout.Widgets))
	for i := range layout.Widgets {
		widget := &layout.Widgets[i]
		widgets = append(widgets, &gqlmodel.HomeWidget{
			ID:     widget.ID,
			Key:    widget.Key,
			Title:  homeOptionalString(widget.Title),
			W:      widget.W,
			H:      widget.H,
			Config: homeWidgetConfigToModel(widget.Config),
		})
	}

	return widgets
}

func homeWidgetConfigToModel(config homelayout.WidgetConfig) *gqlmodel.HomeWidgetConfig {
	return &gqlmodel.HomeWidgetConfig{
		Metric:       homeOptionalString(config.Metric),
		Metrics:      config.Metrics,
		DefinitionID: homeOptionalID(config.DefinitionID),
		CannedKey:    homeOptionalString(config.CannedKey),
		ChartID:      homeOptionalString(config.ChartID),
		ColumnID:     homeOptionalString(config.ColumnID),
		DashboardID:  homeOptionalID(config.DashboardID),
		Text:         homeOptionalString(config.Text),
		Limit:        homeOptionalInt(config.Limit),
		WindowDays:   homeOptionalInt(config.WindowDays),
	}
}

func homeWidgetCatalogToModel(
	catalog *homelayoutservice.WidgetCatalog,
) *gqlmodel.HomeWidgetCatalog {
	widgets := make([]*gqlmodel.HomeWidgetOption, 0, len(catalog.Widgets))
	for _, widget := range catalog.Widgets {
		widgets = append(widgets, &gqlmodel.HomeWidgetOption{
			Key:              widget.Key,
			Label:            widget.Label,
			Description:      widget.Description,
			Category:         widget.Category,
			ConfigKind:       string(widget.ConfigKind),
			AnalyticsInclude: homeOptionalString(widget.AnalyticsInclude),
			DefaultW:         widget.DefaultW,
			DefaultH:         widget.DefaultH,
			MinW:             widget.MinW,
			MinH:             widget.MinH,
			MaxW:             widget.MaxW,
			MaxH:             widget.MaxH,
		})
	}

	metrics := make([]*gqlmodel.HomeMetricOption, 0, len(catalog.Metrics))
	for _, metric := range catalog.Metrics {
		metrics = append(metrics, &gqlmodel.HomeMetricOption{
			Key:   metric.Key,
			Label: metric.Label,
		})
	}

	categories := make([]*gqlmodel.HomeWidgetCategory, 0, len(catalog.Categories))
	for _, category := range catalog.Categories {
		categories = append(categories, &gqlmodel.HomeWidgetCategory{
			Key:         category.Key,
			Label:       category.Label,
			Description: category.Description,
		})
	}

	return &gqlmodel.HomeWidgetCatalog{
		Widgets:     widgets,
		Metrics:     metrics,
		Categories:  categories,
		Densities:   catalog.Densities,
		GridColumns: catalog.GridColumns,
		MaxWidgets:  catalog.MaxWidgets,
	}
}

func homePresetToModel(view *homelayoutservice.PresetView) *gqlmodel.HomeLayoutPreset {
	preset := view.Preset

	roleIDs := make([]string, 0, len(preset.RoleIDs))
	for _, roleID := range preset.RoleIDs {
		roleIDs = append(roleIDs, roleID.String())
	}

	return &gqlmodel.HomeLayoutPreset{
		ID:                 preset.ID.String(),
		Name:               preset.Name,
		Description:        homeOptionalString(preset.Description),
		Widgets:            homeWidgetsToModel(preset.Layout),
		RoleIds:            roleIDs,
		CoreResponsibility: homeOptionalString(string(preset.CoreResponsibility)),
		IsOrgDefault:       preset.IsOrgDefault,
		Locked:             preset.Locked,
		Priority:           preset.Priority,
		AssignedUserCount:  view.AssignedUserCount,
		Version:            int(preset.Version),
		CreatedAt:          int(preset.CreatedAt),
		UpdatedAt:          int(preset.UpdatedAt),
	}
}

func homeWidgetsFromInput(inputs []*gqlmodel.HomeWidgetInput) *homelayout.Layout {
	widgets := make([]homelayout.Widget, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		widgets = append(widgets, homelayout.Widget{
			ID:     input.ID,
			Key:    input.Key,
			Title:  derefString(input.Title),
			W:      input.W,
			H:      input.H,
			Config: homeWidgetConfigFromInput(input.Config),
		})
	}

	return &homelayout.Layout{Widgets: widgets}
}

func homeWidgetConfigFromInput(input *gqlmodel.HomeWidgetConfigInput) homelayout.WidgetConfig {
	if input == nil {
		return homelayout.WidgetConfig{}
	}

	return homelayout.WidgetConfig{
		Metric:       derefString(input.Metric),
		Metrics:      input.Metrics,
		DefinitionID: homeParseID(input.DefinitionID),
		CannedKey:    derefString(input.CannedKey),
		ChartID:      derefString(input.ChartID),
		ColumnID:     derefString(input.ColumnID),
		DashboardID:  homeParseID(input.DashboardID),
		Text:         derefString(input.Text),
		Limit:        derefInt(input.Limit),
		WindowDays:   derefInt(input.WindowDays),
	}
}

func homeDocumentFromInput(input *gqlmodel.HomeLayoutInput) *homelayout.Document {
	document := &homelayout.Document{
		SchemaVersion: homelayout.DocumentSchemaVersion,
		Mode:          homelayout.ModePreset,
		Density:       input.Density,
	}

	if input.Customized {
		document.Mode = homelayout.ModeCustom
		document.Layout = homeWidgetsFromInput(input.Widgets)
	}

	return document
}

func homePresetRoleIDs(roleIDs []string) []pulid.ID {
	parsed := make([]pulid.ID, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		id, err := pulid.Parse(roleID)
		if err != nil {
			continue
		}
		parsed = append(parsed, id)
	}
	return parsed
}

func homeCoreResponsibility(value *string) permission.CoreResponsibility {
	if value == nil {
		return ""
	}
	return permission.CoreResponsibility(*value)
}

func homeOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func homeOptionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func homeOptionalID(value pulid.ID) *string {
	if value.IsNil() {
		return nil
	}
	id := value.String()
	return &id
}

func homeParseID(value *string) pulid.ID {
	if value == nil || *value == "" {
		return pulid.ID("")
	}
	id, err := pulid.Parse(*value)
	if err != nil {
		return pulid.ID("")
	}
	return id
}
