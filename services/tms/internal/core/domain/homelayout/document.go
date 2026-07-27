package homelayout

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// Mode records where a user's home screen comes from. A user who has never
// touched the canvas stays on ModePreset, so an administrator's later edits to
// the assigned preset reach them; the moment they rearrange anything they move
// to ModeCustom and own their own layout.
type Mode string

const (
	ModePreset = Mode("preset")
	ModeCustom = Mode("custom")
)

func (m Mode) IsValid() bool {
	switch m {
	case ModePreset, ModeCustom:
		return true
	default:
		return false
	}
}

// Widget is one card on the canvas. Position is deliberately not stored: the
// widget list is ordered, and x/y are derived by packing that order into the
// grid at render time. Storing only the order means a layout can never drift
// into an inconsistent state where coordinates and sequence disagree.
type Widget struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Title  string       `json:"title,omitempty"`
	W      int          `json:"w"`
	H      int          `json:"h"`
	Config WidgetConfig `json:"config,omitempty"`
}

type Layout struct {
	Widgets []Widget `json:"widgets"`
}

func (l *Layout) Clone() *Layout {
	if l == nil {
		return &Layout{Widgets: []Widget{}}
	}

	widgets := make([]Widget, len(l.Widgets))
	copy(widgets, l.Widgets)

	for i := range widgets {
		if metrics := widgets[i].Config.Metrics; metrics != nil {
			cloned := make([]string, len(metrics))
			copy(cloned, metrics)
			widgets[i].Config.Metrics = cloned
		}
	}

	return &Layout{Widgets: widgets}
}

func (l *Layout) Validate(multiErr *errortypes.MultiError, fieldPrefix string) {
	if l == nil {
		multiErr.Add(fieldPrefix, errortypes.ErrRequired, "Layout is required")
		return
	}

	if len(l.Widgets) > MaxWidgets {
		multiErr.Add(fieldPrefix+".widgets", errortypes.ErrInvalid,
			fmt.Sprintf("A home screen holds at most %d widgets", MaxWidgets))
		return
	}

	seen := make(map[string]struct{}, len(l.Widgets))
	for i := range l.Widgets {
		validateWidget(multiErr, &l.Widgets[i], seen,
			fmt.Sprintf("%s.widgets[%d]", fieldPrefix, i))
	}
}

// Normalize drops widgets whose key is no longer in the catalog, clamps every
// geometry to the definition's bounds, and strips configuration the widget's
// kind cannot use. Retiring a widget is therefore a one-line catalog change:
// stored layouts referencing it simply stop mentioning it.
func (l *Layout) Normalize() *Layout {
	if l == nil {
		return &Layout{Widgets: []Widget{}}
	}

	widgets := make([]Widget, 0, min(len(l.Widgets), MaxWidgets))
	seen := make(map[string]struct{}, len(l.Widgets))

	for _, widget := range l.Widgets {
		if len(widgets) == MaxWidgets {
			break
		}

		definition, ok := widgetDefinition(widget.Key)
		if !ok {
			continue
		}
		if widget.ID == "" {
			continue
		}
		if _, dup := seen[widget.ID]; dup {
			continue
		}
		seen[widget.ID] = struct{}{}

		widget.W = clampSpan(widget.W, definition.MinW, definition.MaxW, definition.DefaultW)
		widget.H = clampSpan(widget.H, definition.MinH, definition.MaxH, definition.DefaultH)
		widget.Config = normalizeConfig(definition.ConfigKind, widget.Config)

		widgets = append(widgets, widget)
	}

	return &Layout{Widgets: widgets}
}

func validateWidget(
	multiErr *errortypes.MultiError,
	widget *Widget,
	seen map[string]struct{},
	fieldPath string,
) {
	switch {
	case widget.ID == "":
		multiErr.Add(fieldPath+".id", errortypes.ErrRequired, "Widget ID is required")
		return
	case contains(seen, widget.ID):
		multiErr.Add(fieldPath+".id", errortypes.ErrDuplicate, "Duplicate widget ID")
		return
	}
	seen[widget.ID] = struct{}{}

	definition, ok := widgetDefinition(widget.Key)
	if !ok {
		multiErr.Add(fieldPath+".key", errortypes.ErrInvalid,
			fmt.Sprintf("Unknown widget: %s", widget.Key))
		return
	}

	validateWidgetGeometry(multiErr, widget, definition, fieldPath)
	validateWidgetConfig(multiErr, definition.ConfigKind, widget.Config, fieldPath)
}

func validateWidgetGeometry(
	multiErr *errortypes.MultiError,
	widget *Widget,
	definition WidgetDefinition,
	fieldPath string,
) {
	if widget.W < definition.MinW || widget.W > definition.MaxW {
		multiErr.Add(fieldPath+".w", errortypes.ErrInvalid,
			fmt.Sprintf("%s must be between %d and %d columns wide",
				definition.Label, definition.MinW, definition.MaxW))
	}
	if widget.H < definition.MinH || widget.H > definition.MaxH {
		multiErr.Add(fieldPath+".h", errortypes.ErrInvalid,
			fmt.Sprintf("%s must be between %d and %d rows tall",
				definition.Label, definition.MinH, definition.MaxH))
	}
}

// Document is the per-user home screen preference. It holds a layout only when
// the user has diverged from what they were assigned; until then it records
// which preset they are following so that preset's edits keep reaching them.
type Document struct {
	SchemaVersion   int      `json:"schemaVersion"`
	Mode            Mode     `json:"mode"`
	BasedOnPresetID pulid.ID `json:"basedOnPresetId,omitempty"`
	Layout          *Layout  `json:"layout,omitempty"`
	Density         string   `json:"density"`
}

func (d *Document) Validate(multiErr *errortypes.MultiError) {
	if d.SchemaVersion != DocumentSchemaVersion {
		multiErr.Add("schemaVersion", errortypes.ErrInvalid,
			fmt.Sprintf("Schema version must be %d", DocumentSchemaVersion))
	}

	if !d.Mode.IsValid() {
		multiErr.Add("mode", errortypes.ErrInvalid, "Mode must be preset or custom")
	}

	if !densityAllowed(d.Density) {
		multiErr.Add("density", errortypes.ErrInvalid,
			fmt.Sprintf("Density must be one of %v", Densities()))
	}

	if d.Mode == ModeCustom {
		if d.Layout == nil {
			multiErr.Add("layout", errortypes.ErrRequired,
				"A customized home screen needs a layout")
			return
		}
		d.Layout.Validate(multiErr, "layout")
	}
}

func (d *Document) Normalize() *Document {
	normalized := &Document{
		SchemaVersion:   DocumentSchemaVersion,
		Mode:            d.Mode,
		BasedOnPresetID: d.BasedOnPresetID,
		Density:         d.Density,
	}

	if !normalized.Mode.IsValid() {
		normalized.Mode = ModePreset
	}
	if !densityAllowed(normalized.Density) {
		normalized.Density = DensityComfortable
	}

	if normalized.Mode == ModeCustom {
		normalized.Layout = d.Layout.Normalize()
	}

	return normalized
}

func DefaultDocument() *Document {
	return &Document{
		SchemaVersion: DocumentSchemaVersion,
		Mode:          ModePreset,
		Density:       DensityComfortable,
	}
}

// NextWidgetID names a widget uniquely within its layout. The scheme matches
// the report dashboard's tile IDs so both canvases read the same way in stored
// JSON.
func NextWidgetID(widgets []Widget) string {
	suffix := len(widgets) + 1
	for {
		candidate := fmt.Sprintf("widget_%d", suffix)
		if !containsWidgetID(widgets, candidate) {
			return candidate
		}
		suffix++
	}
}

func containsWidgetID(widgets []Widget, id string) bool {
	for i := range widgets {
		if widgets[i].ID == id {
			return true
		}
	}
	return false
}

func contains(seen map[string]struct{}, key string) bool {
	_, ok := seen[key]
	return ok
}

func clampSpan(value, minimum, maximum, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
