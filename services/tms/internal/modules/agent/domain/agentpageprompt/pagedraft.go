package agentpageprompt

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/shared/textfence"
)

func DescribePageDraft(draft *pagedraft.Draft) string {
	var builder strings.Builder
	switch {
	case draft.ShipmentImport != nil:
		builder.WriteString(
			"- The shipment the person is building on this page, as it stands now. It is not " +
				"saved. Change it only with the draft tools; what they set shows here on the " +
				"next turn:\n",
		)
		builder.WriteString(pageDraftOpenTag)
		writeImportDraft(&builder, draft.ShipmentImport)
	case draft.Formula != nil:
		builder.WriteString(
			"- The formula template in the person's editor, which may not be saved yet. Answer " +
				"about this source, and check and test it before you say what it charges. " +
				"\"The load\" or \"a load\" is the shipment this formula prices, not a record " +
				"to look up, and a request for a price is a change to write into this source:\n",
		)
		builder.WriteString(pageDraftOpenTag)
		writeSourceDraft(&builder, &sourceDraft{
			surface:     pagedraft.SurfaceFormula,
			noun:        "template",
			id:          draft.Formula.TemplateID,
			name:        draft.Formula.Name,
			description: draft.Formula.Description,
			source:      draft.Formula.Source,
		})
	case draft.FormulaLibrary != nil:
		builder.WriteString(
			"- The formula library in the person's editor, which may not be saved yet. Answer " +
				"about this source, and check and test it before you say what it does:\n",
		)
		builder.WriteString(pageDraftOpenTag)
		writeSourceDraft(&builder, &sourceDraft{
			surface:     pagedraft.SurfaceFormulaLibrary,
			noun:        "library",
			id:          draft.FormulaLibrary.LibraryID,
			name:        draft.FormulaLibrary.Name,
			description: draft.FormulaLibrary.Description,
			source:      draft.FormulaLibrary.Source,
		})
	default:
		return ""
	}
	builder.WriteString("\n")
	builder.WriteString(pageDraftCloseTag)

	return builder.String()
}

func draftText(value string) string {
	return textfence.FenceText(value, pageDraftCloseTag)
}

func writeImportDraft(builder *strings.Builder, draft *pagedraft.ShipmentImport) {
	builder.WriteString("\nsurface: shipment_import")
	for _, required := range pagedraft.AllRequiredFields() {
		builder.WriteString("\nrequired ")
		builder.WriteString(string(required))
		builder.WriteString(": ")
		if value := draft.Required.Value(required); value != "" {
			builder.WriteString(draftText(value))
		} else {
			builder.WriteString("not set")
		}
	}
	for _, field := range draft.Fields {
		builder.WriteString("\nfield ")
		builder.WriteString(draftText(field.Key))
		if field.Label != "" && field.Label != field.Key {
			builder.WriteString(" (")
			builder.WriteString(draftText(field.Label))
			builder.WriteString(")")
		}
		builder.WriteString(": ")
		if field.Value == "" {
			builder.WriteString("empty")
		} else {
			builder.WriteString(draftText(field.Value))
		}
		builder.WriteString(" [")
		builder.WriteString(string(field.Status))
		builder.WriteString(", confidence ")
		builder.WriteString(strconv.FormatFloat(field.Confidence, 'f', 2, 64))
		builder.WriteString("]")
	}

	attention := 0
	for idx := range draft.Stops {
		stop := &draft.Stops[idx]
		if !stop.HasLocation() || !stop.HasSchedule() {
			attention++
		}
		writeImportStop(builder, idx, stop)
	}
	builder.WriteString("\nstops: ")
	builder.WriteString(strconv.Itoa(len(draft.Stops)))
	builder.WriteString(", needing a location or a date: ")
	builder.WriteString(strconv.Itoa(attention))
}

func writeImportStop(builder *strings.Builder, idx int, stop *pagedraft.ImportStop) {
	builder.WriteString("\nstop ")
	builder.WriteString(strconv.Itoa(idx))
	builder.WriteString(" (")
	builder.WriteString(string(stop.Role))
	builder.WriteString("): ")
	parts := make([]string, 0, 5)
	for _, part := range []string{
		stop.Name, stop.AddressLine1, stop.City, stop.State, stop.PostalCode,
	} {
		if part != "" {
			parts = append(parts, draftText(part))
		}
	}
	if len(parts) == 0 {
		builder.WriteString("no address read")
	} else {
		builder.WriteString(strings.Join(parts, ", "))
	}
	builder.WriteString("; location: ")
	if stop.HasLocation() {
		builder.WriteString(draftText(stop.LocationID))
	} else {
		builder.WriteString("not matched")
	}
	builder.WriteString("; date: ")
	if stop.Date == "" {
		builder.WriteString("none")
	} else {
		builder.WriteString(draftText(stop.Date))
	}
	if !stop.HasSchedule() {
		builder.WriteString(" (not a usable date)")
	}
	if stop.TimeWindow != "" {
		builder.WriteString("; window: ")
		builder.WriteString(draftText(stop.TimeWindow))
	}
}

type sourceDraft struct {
	surface     pagedraft.Surface
	noun        string
	id          string
	name        string
	description string
	source      string
}

func writeSourceDraft(builder *strings.Builder, draft *sourceDraft) {
	builder.WriteString("\nsurface: ")
	builder.WriteString(string(draft.surface))
	builder.WriteString("\n")
	builder.WriteString(draft.noun)
	builder.WriteString(": ")
	if draft.id == "" {
		builder.WriteString("new, not saved yet")
	} else {
		builder.WriteString(draftText(draft.id))
	}
	if draft.name != "" {
		builder.WriteString("\nname: ")
		builder.WriteString(draftText(draft.name))
	}
	if draft.description != "" {
		builder.WriteString("\ndescription: ")
		builder.WriteString(draftText(draft.description))
	}
	builder.WriteString("\nsource: ")
	if strings.TrimSpace(draft.source) == "" {
		builder.WriteString("empty")
		return
	}
	builder.WriteString("\n")
	builder.WriteString(draftText(draft.source))
}
