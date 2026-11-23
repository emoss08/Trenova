package reportaccess

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
)

func (r *Resolver) Reason(ctx context.Context, d *Denial) string {
	locale := i18n.FromContext(ctx)
	entity := r.catalog.Label(ctx, d.Entity.Key, d.Entity.Label)
	plural := r.catalog.Label(ctx, d.Entity.Key+"@plural", d.Entity.Plural)
	switch {
	case d.Use.Decision == Available:
		return ""
	case d.Use.Decision == NotOnPlan:
		return planReason(locale, d, entity, plural)
	case d.Use.Decision == NeedsOperation:
		return i18n.Translate(
			locale,
			"{0} need the \"{1}\" permission on {2}",
			plural,
			r.OperationLabel(locale, d.Use.Kind, d.Use.Operation),
			r.ResourceLabel(locale, d.Use.Kind),
		)
	case d.Label == "" && d.Use.Decision == NeedsKindRead:
		return i18n.Translate(locale, "you do not have read access to {0}", plural)
	case d.Label == "" && d.Use.Decision == NeedsExport:
		return i18n.Translate(locale, "you do not have export access to {0}", plural)
	case d.Use.Decision == OwnScopeUnsupported:
		kind := r.KindLabel(ctx, d.Use.Kind)
		return i18n.Translate(
			locale,
			"your access to {0} is limited to your own records, but {0} cannot be limited that way in reports",
			kind,
		)
	case d.Use.Decision == NeedsKindRead:
		return i18n.Translate(
			locale,
			"you do not have read access to {0}, which \"{1}\" on {2} needs; without it a report can use only its reference fields",
			r.KindLabel(ctx, d.Use.Kind),
			d.Label,
			entity,
		)
	case d.Use.Decision == WithheldByCeiling:
		return i18n.Translate(
			locale,
			"field \"{0}\" on {1} requires {2} access",
			d.Label, entity, d.Use.Sensitivity.String(),
		)
	case d.Use.Decision == NeedsFieldGroup:
		return i18n.Translate(
			locale,
			"field \"{0}\" on {1} is in the {2} protected field group, which your role does not grant",
			d.Label,
			entity,
			d.Use.Group.String(),
		)
	case d.Use.Decision == NeedsExport:
		return i18n.Translate(
			locale,
			"you do not have export access to {0}, which \"{1}\" on {2} needs",
			r.KindLabel(ctx, d.Use.Kind), d.Label, entity,
		)
	default:
		return i18n.Translate(
			locale,
			"field \"{0}\" on {1} is not available to you",
			d.Label,
			entity,
		)
	}
}

func (r *Resolver) Need(use Use) string {
	switch use.Decision {
	case Available:
		return ""
	case NeedsKindRead:
		return use.Kind.String() + " read"
	case WithheldByCeiling:
		return use.Sensitivity.String() + " access"
	case NeedsFieldGroup:
		return "the " + use.Group.String() + " field group"
	case NeedsExport:
		return use.Kind.String() + " export"
	case OwnScopeUnsupported:
		return use.Kind.String() + " access beyond your own records"
	case NeedsOperation:
		return use.Kind.String() + " " + string(use.Operation)
	case NotOnPlan:
		if use.Plan == nil || use.Plan.AddOnName == "" {
			return "a plan that includes it"
		}
		return "the " + use.Plan.AddOnName + " add-on"
	default:
		return "access"
	}
}

func (r *Resolver) ResourceLabel(locale i18n.Locale, resource permission.Resource) string {
	if def, ok := r.registry.Get(resource.String()); ok && def.DisplayName != "" {
		return i18n.Translate(locale, def.DisplayName)
	}
	return resource.String()
}

func (r *Resolver) OperationLabel(
	locale i18n.Locale,
	resource permission.Resource,
	operation permission.Operation,
) string {
	if def, ok := r.registry.Get(resource.String()); ok {
		for _, op := range def.Operations {
			if op.Operation == operation && op.DisplayName != "" {
				return i18n.Translate(locale, op.DisplayName)
			}
		}
	}
	return string(operation)
}

func (r *Resolver) KindLabel(ctx context.Context, resource permission.Resource) string {
	fallback := ""
	for e, k := range r.kinds {
		if k.resource != resource {
			continue
		}
		label := r.catalog.Label(ctx, e.Key+"@plural", e.Plural)
		if !k.withinRecord {
			return label
		}
		if fallback == "" || label < fallback {
			fallback = label
		}
	}
	if fallback != "" {
		return fallback
	}
	return resource.String()
}

func (a *Access) Refusal(ctx context.Context, err error) error {
	if analytics.ErrorCode(err) != analytics.CodeDenied {
		return err
	}
	denials := a.Denials()
	if restriction := a.planRefusal(ctx, denials); restriction != nil {
		return restriction
	}
	reasons := make([]string, 0, len(denials))
	for i := range denials {
		if reason := a.resolver.Reason(
			ctx,
			&denials[i],
		); reason != "" &&
			!slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	if len(reasons) == 0 {
		var ae *analytics.Error
		if errors.As(err, &ae) {
			reasons = append(reasons, ae.Message)
		}
	}
	return errortypes.NewAuthorizationError(
		"This report references data you do not have access to: {0}",
		strings.Join(reasons, "; "),
	)
}

func planReason(locale i18n.Locale, d *Denial, entity, plural string) string {
	addOn := ""
	if d.Use.Plan != nil {
		addOn = i18n.Translate(locale, d.Use.Plan.AddOnName)
	}
	switch {
	case addOn == "" && d.Label == "":
		return i18n.Translate(locale, "{0} are not included in your plan", plural)
	case addOn == "":
		return i18n.Translate(
			locale,
			"\"{0}\" on {1} is not included in your plan",
			d.Label,
			entity,
		)
	case d.Label == "":
		return i18n.Translate(
			locale,
			"{0} are part of the {1} add-on, which is not on your plan",
			plural,
			addOn,
		)
	default:
		return i18n.Translate(
			locale,
			"\"{0}\" on {1} is part of the {2} add-on, which is not on your plan",
			d.Label,
			entity,
			addOn,
		)
	}
}

var planRefusalMessage = errortypes.BaseError{
	Message: "This report reads data your plan does not include: {0}",
}

func (a *Access) planRefusal(ctx context.Context, denials []Denial) error {
	var first *permissionports.PlanDenial
	reasons := make([]string, 0, 1)
	for i := range denials {
		if denials[i].Use.Decision != NotOnPlan || denials[i].Use.Plan == nil {
			continue
		}
		if first == nil {
			first = denials[i].Use.Plan
		}
		if reason := a.resolver.Reason(ctx, &denials[i]); !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	if first == nil {
		return nil
	}
	restriction := first.Restriction()
	restriction.Message = planRefusalMessage.Message
	restriction.Args = []any{strings.Join(reasons, "; ")}
	return restriction
}
