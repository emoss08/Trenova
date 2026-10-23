package pageaccessservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/emoss08/trenova/shared/i18n"
)

// carriersNamed is how many linking kinds a sentence names before it counts
// the rest.
const carriersNamed = 3

// viewWord is one phrase a sentence names part of a view by, and the view
// fields it stands for; viewwords_gen.go holds every kind's.
type viewWord struct {
	Label  string
	Fields []string
}

type GuideRequest struct {
	Resources      []string
	MaxSensitivity permission.FieldSensitivity
}

// ScreenUse is a screen a resource's grant changes, and how the screen
// depends on it.
type ScreenUse struct {
	Path string
	Name string
	Use  pageaccess.Use
}

// ResourceGuide is what granting and withholding a resource's read means, in
// sentences built from the registry, the resource's reference view and the
// kinds that link to it.
type ResourceGuide struct {
	Resource string
	Name     string
	// Granted is what a role with the resource's read sees.
	Granted string
	// Withheld is what a role without it still sees of the resource where
	// another record links to one.
	Withheld string
	// Ceiling names the view's fields that rank above the role's
	// sensitivity ceiling, which stay withheld without the read; empty when
	// none do.
	Ceiling string
	Screens []ScreenUse
}

// Guide words each resource for a role whose sensitivity reaches
// MaxSensitivity, with the screens each one's grant changes.
func (s *Service) Guide(ctx context.Context, req *GuideRequest) ([]ResourceGuide, error) {
	if !req.MaxSensitivity.IsValid() {
		return nil, errortypes.NewValidationError("maxSensitivity", errortypes.ErrInvalid,
			"Choose a valid sensitivity")
	}
	if len(req.Resources) > maxGuideResources {
		return nil, errortypes.NewValidationError("resources", errortypes.ErrInvalid,
			"Ask for at most 64 resources at a time")
	}
	ceiling := fieldsensitivity.RoleCeiling(req.MaxSensitivity)

	out := make([]ResourceGuide, 0, len(req.Resources))
	for _, resource := range req.Resources {
		def, ok := s.registry.Get(resource)
		if !ok {
			return nil, errortypes.NewValidationError("resources", errortypes.ErrInvalid,
				"A resource is not one the registry declares")
		}
		out = append(out, s.guide(ctx, def, ceiling))
	}
	return out, nil
}

func (s *Service) guide(
	ctx context.Context,
	def *permission.ResourceDefinition,
	ceiling permission.FieldSensitivity,
) ResourceGuide {
	name := i18n.T(ctx, def.DisplayName)
	read := s.operationName(ctx, def.Resource, permission.OpRead)
	words := viewWords[def.Resource]
	carriers := s.carriers(ctx, def.Resource)

	guide := ResourceGuide{
		Resource: def.Resource,
		Name:     name,
		Granted: i18n.T(ctx,
			"Can see every field of {0} records, on their own screens and wherever another "+
				"record shows one.", name),
		Screens: s.screens(def.Resource),
	}

	switch {
	case carriers == "":
		guide.Withheld = i18n.T(ctx,
			"No other record shows {0} records, so without {1} on {0} they see none of them.",
			name, read)
	case len(words) == 0:
		guide.Withheld = i18n.T(ctx,
			"{0} records show only that one is linked wherever a record they can read links "+
				"to one ({1}); every field needs {2} on {0}.",
			name, carriers, read)
	default:
		labels := make([]string, 0, len(words))
		for _, word := range words {
			labels = append(labels, i18n.T(ctx, word.Label))
		}
		guide.Withheld = i18n.T(ctx,
			"{0} records show only their {1} wherever a record they can read links to one "+
				"({2}); the rest needs {3} on {0}.",
			name, i18n.List(ctx, labels), carriers, read)
	}

	var above []string
	for _, word := range words {
		if slices.ContainsFunc(word.Fields, func(field string) bool {
			level := s.registry.GetFieldSensitivity(def.Resource, field)
			return field != "id" && !fieldsensitivity.VisibleAt(level, ceiling)
		}) {
			above = append(above, i18n.T(ctx, word.Label))
		}
	}
	if len(above) > 0 && carriers != "" {
		guide.Ceiling = i18n.T(ctx,
			"Without {0} on {1}, its {2} stay withheld there too: they rank above this "+
				"role's sensitivity ceiling.",
			read, name, i18n.List(ctx, above))
	}

	return guide
}

// carriers names the kinds whose records link to a resource, the first few
// by name and the rest by count, or "" when none do.
func (s *Service) carriers(ctx context.Context, resource string) string {
	kinds := linkedFrom[resource]
	if len(kinds) == 0 {
		return ""
	}
	// The kinds the most screens read come first: those are the records a
	// person meets this one on.
	ordered := slices.Clone(kinds)
	slices.SortStableFunc(ordered, func(a, b string) int {
		if d := s.readBy[b] - s.readBy[a]; d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	names := make([]string, 0, len(ordered))
	for _, kind := range ordered {
		names = append(names, s.resourceName(ctx, kind))
	}
	if len(names) <= carriersNamed+1 {
		return i18n.List(ctx, names)
	}
	named := append(slices.Clone(names[:carriersNamed]),
		i18n.T(ctx, "{0, plural, one {# other} other {# others}}", len(names)-carriersNamed))

	return i18n.List(ctx, named)
}

// screens is every screen that depends on a resource, by path.
func (s *Service) screens(resource string) []ScreenUse {
	uses := s.manifest.ScreensUsing(resource)
	out := make([]ScreenUse, 0, len(uses))
	for _, use := range uses {
		out = append(out, ScreenUse{Path: use.Screen.Path, Name: use.Screen.Name, Use: use.Use})
	}
	return out
}
