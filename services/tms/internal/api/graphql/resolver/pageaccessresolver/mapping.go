package pageaccessresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pageaccess"
)

func draftFromInput(
	input *gqlmodel.PageAccessRoleDraftInput,
) (*pageaccessservice.RoleDraft, error) {
	if input == nil {
		return nil, nil //nolint:nilnil // no draft: a saved role is previewed
	}
	parents, err := gqlid.PulidsFromStrings(input.ParentRoleIds, "parentRoleIds")
	if err != nil {
		return nil, err
	}
	draft := &pageaccessservice.RoleDraft{
		MaxSensitivity: permission.FieldSensitivity(input.MaxSensitivity),
		ParentRoleIDs:  parents,
		Permissions:    make([]pageaccessservice.DraftPermission, 0, len(input.Permissions)),
	}
	for _, grant := range input.Permissions {
		if grant == nil {
			continue
		}
		ops := make([]permission.Operation, 0, len(grant.Operations))
		for _, op := range grant.Operations {
			ops = append(ops, permission.Operation(op))
		}
		draft.Permissions = append(draft.Permissions, pageaccessservice.DraftPermission{
			Resource:   grant.Resource,
			Operations: ops,
		})
	}

	return draft, nil
}

func pageAccessToModel(access *pageaccessservice.Access) *gqlmodel.PageAccess {
	if access == nil {
		return nil
	}
	out := &gqlmodel.PageAccess{
		Path:       access.Path,
		Name:       access.Name,
		Operations: access.Operations,
		Requires:   requirementsToModel(access.Requires),
		Endpoints:  make([]*gqlmodel.PageAccessEndpoint, 0, len(access.Endpoints)),
		Kinds:      make([]*gqlmodel.PageAccessKind, 0, len(access.Kinds)),
		Withheld:   access.Withheld,
		Blocked:    access.Blocked,
	}
	for _, endpoint := range access.Endpoints {
		out.Endpoints = append(out.Endpoints, &gqlmodel.PageAccessEndpoint{
			Method:   endpoint.Method,
			Path:     endpoint.Path,
			Requires: requirementsToModel(endpoint.Requires),
		})
	}
	for _, kind := range access.Kinds {
		model := &gqlmodel.PageAccessKind{
			Resource:     kind.Resource,
			ResourceName: kind.ResourceName,
			Use:          useToModel(kind.Use),
			CanRead:      kind.CanRead,
			Fields:       make([]*gqlmodel.PageAccessField, 0, len(kind.Fields)),
			Withheld:     kind.Withheld,
		}
		for _, field := range kind.Fields {
			model.Fields = append(model.Fields, &gqlmodel.PageAccessField{
				Name:     field.Name,
				InView:   field.InView,
				Decision: decisionToModel(field.Decision),
			})
		}
		out.Kinds = append(out.Kinds, model)
	}

	return out
}

func requirementsToModel(
	requires []pageaccessservice.Requirement,
) []*gqlmodel.PageAccessRequirement {
	out := make([]*gqlmodel.PageAccessRequirement, 0, len(requires))
	for _, r := range requires {
		out = append(out, &gqlmodel.PageAccessRequirement{
			Resource:      r.Resource,
			ResourceName:  r.ResourceName,
			Operation:     r.Operation,
			OperationName: r.OperationName,
			Allowed:       r.Allowed,
		})
	}
	return out
}

func useToModel(use pageaccess.Use) gqlmodel.PageAccessUse {
	switch use {
	case pageaccess.UseRequired:
		return gqlmodel.PageAccessUseRequired
	case pageaccess.UseRead:
		return gqlmodel.PageAccessUseRead
	case pageaccess.UseDeclared:
		return gqlmodel.PageAccessUseDeclared
	case pageaccess.UseReference:
		return gqlmodel.PageAccessUseReference
	default:
		return gqlmodel.PageAccessUseReference
	}
}

func decisionToModel(decision pageaccessservice.Decision) gqlmodel.PageAccessDecision {
	switch decision {
	case pageaccessservice.DecisionVisible:
		return gqlmodel.PageAccessDecisionVisible
	case pageaccessservice.DecisionWithheldSensitivity:
		return gqlmodel.PageAccessDecisionWithheldSensitivity
	case pageaccessservice.DecisionWithheldKind:
		return gqlmodel.PageAccessDecisionWithheldKind
	default:
		return gqlmodel.PageAccessDecisionWithheldKind
	}
}

func guidesToModel(guides []pageaccessservice.ResourceGuide) []*gqlmodel.RoleResourceGuide {
	out := make([]*gqlmodel.RoleResourceGuide, 0, len(guides))
	for _, guide := range guides {
		model := &gqlmodel.RoleResourceGuide{
			Resource: guide.Resource,
			Name:     guide.Name,
			Granted:  guide.Granted,
			Withheld: guide.Withheld,
			Screens:  make([]*gqlmodel.RoleResourceScreen, 0, len(guide.Screens)),
		}
		if guide.Ceiling != "" {
			ceiling := guide.Ceiling
			model.Ceiling = &ceiling
		}
		for _, screen := range guide.Screens {
			model.Screens = append(model.Screens, &gqlmodel.RoleResourceScreen{
				Path: screen.Path,
				Name: screen.Name,
				Use:  useToModel(screen.Use),
			})
		}
		out = append(out, model)
	}
	return out
}
