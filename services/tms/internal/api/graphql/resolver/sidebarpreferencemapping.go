package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/sidebarpreference"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/sidebarpreferenceservice"
	"github.com/emoss08/trenova/pkg/authctx"
)

func sidebarPreferenceRequest(authCtx *authctx.AuthContext) sidebarpreferenceservice.Request {
	return sidebarpreferenceservice.Request{
		TenantInfo: tenantInfo(authCtx),
		Principal: services.PrincipalInfo{
			Type:     services.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
	}
}

func sidebarPreferencesToModel(
	effective *sidebarpreferenceservice.EffectivePreferences,
) *gqlmodel.SidebarPreferences {
	doc := effective.Document

	sections := make([]*gqlmodel.SidebarSectionPreference, 0, len(doc.Sections))
	for _, section := range doc.Sections {
		sections = append(sections, &gqlmodel.SidebarSectionPreference{
			Key:    section.Key,
			Hidden: section.Hidden,
		})
	}

	return &gqlmodel.SidebarPreferences{
		SchemaVersion:    doc.SchemaVersion,
		Version:          int(effective.Version),
		Sections:         sections,
		AttentionMetrics: doc.AttentionMetrics,
		QuickActionIds:   doc.QuickActionIDs,
		Activity: &gqlmodel.SidebarActivityPreference{
			PageSize:    doc.Activity.PageSize,
			DefaultOpen: doc.Activity.DefaultOpen,
		},
	}
}

func sidebarOptionsToModel(
	options *sidebarpreferenceservice.CustomizationOptions,
) *gqlmodel.SidebarCustomizationOptions {
	sections := make([]*gqlmodel.SidebarSectionOption, 0, len(options.Sections))
	for _, section := range options.Sections {
		sections = append(sections, &gqlmodel.SidebarSectionOption{
			Key:      section.Key,
			Label:    section.Label,
			Hideable: section.Hideable,
		})
	}

	metrics := make([]*gqlmodel.SidebarAttentionMetricOption, 0, len(options.AttentionMetrics))
	for _, metric := range options.AttentionMetrics {
		metrics = append(metrics, &gqlmodel.SidebarAttentionMetricOption{
			Key:   metric.Key,
			Label: metric.Label,
		})
	}

	actions := make([]*gqlmodel.SidebarQuickActionOption, 0, len(options.QuickActions))
	for _, action := range options.QuickActions {
		actions = append(actions, &gqlmodel.SidebarQuickActionOption{
			ID:    action.ID,
			Label: action.Label,
		})
	}

	return &gqlmodel.SidebarCustomizationOptions{
		Sections:          sections,
		AttentionMetrics:  metrics,
		QuickActions:      actions,
		MaxQuickActions:   options.MaxQuickActions,
		ActivityPageSizes: options.ActivityPageSizes,
	}
}

func sidebarPreferencesInputToDocument(
	input *gqlmodel.SidebarPreferencesInput,
) *sidebarpreference.Document {
	sections := make([]sidebarpreference.SectionPreference, 0, len(input.Sections))
	for _, section := range input.Sections {
		if section == nil {
			continue
		}
		sections = append(sections, sidebarpreference.SectionPreference{
			Key:    section.Key,
			Hidden: section.Hidden,
		})
	}

	var activity sidebarpreference.ActivityPreference
	if input.Activity != nil {
		activity = sidebarpreference.ActivityPreference{
			PageSize:    input.Activity.PageSize,
			DefaultOpen: input.Activity.DefaultOpen,
		}
	}

	return &sidebarpreference.Document{
		SchemaVersion:    sidebarpreference.DocumentSchemaVersion,
		Sections:         sections,
		AttentionMetrics: input.AttentionMetrics,
		QuickActionIDs:   input.QuickActionIds,
		Activity:         activity,
	}
}
