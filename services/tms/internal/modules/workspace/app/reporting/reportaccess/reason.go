package reportaccess

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	default:
		return "access"
	}
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
	reasons := make([]string, 0, len(a.denials))
	denials := a.Denials()
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
