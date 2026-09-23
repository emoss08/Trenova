package assistantartifact

// DisplayVersion marks a payload whose table columns and card fields were
// projected for people when the artifact was made. A payload without it was
// stored before the projection existed, and the client reads it by the same
// rules applied to the raw record.
const DisplayVersion = 1

// DisplayType is how a person reads one column of a table or one field of a
// record card. The tool's own result keeps every key the model needs; the
// artifact keeps only the ones a person can reason with, each with the type
// that says how to draw it.
type DisplayType string

const (
	// DisplayText is a short string drawn as it is.
	DisplayText DisplayType = "text"
	// DisplayLongText is prose — a narrative, a recommendation — read in the
	// expanded row or below a card's fields rather than squeezed into a cell.
	DisplayLongText DisplayType = "longText"
	// DisplayDate is an instant read as the day it falls on in the reader's
	// timezone; the value is unix seconds, or a phrase the tool already wrote.
	DisplayDate DisplayType = "date"
	// DisplayDateTime is an instant whose hour matters: a window, an arrival.
	DisplayDateTime DisplayType = "datetime"
	// DisplayMoney is an amount, kept as the decimal the tool wrote.
	DisplayMoney DisplayType = "money"
	// DisplayNumber is a count or measure.
	DisplayNumber DisplayType = "number"
	// DisplayPercent is a value already in percentage points.
	DisplayPercent DisplayType = "percent"
	// DisplayEnum is one of a fixed set with no severity: a category, a type.
	DisplayEnum DisplayType = "enum"
	// DisplayStatus is one of a fixed set with an ordering — a lifecycle or a
	// severity — drawn as a badge whose tone follows its phase.
	DisplayStatus DisplayType = "status"
	// DisplayBoolean is a yes or no.
	DisplayBoolean DisplayType = "boolean"
	// DisplayFlag is a condition worth saying only when it holds, such as a
	// finding whose numbers have gone stale. False is drawn as nothing.
	DisplayFlag DisplayType = "flag"
	// DisplayMetrics is a list of labelled measurements.
	DisplayMetrics DisplayType = "metrics"
	// DisplayLinks is a list of labelled paths inside the app.
	DisplayLinks DisplayType = "links"
)

// AllDisplayTypes is the whole set. The client's DISPLAY_TYPES
// (components/assistant/readable-values.ts) mirrors it; a type added here and
// not there is dropped by the client rather than drawn wrongly.
func AllDisplayTypes() []DisplayType {
	return []DisplayType{
		DisplayText,
		DisplayLongText,
		DisplayDate,
		DisplayDateTime,
		DisplayMoney,
		DisplayNumber,
		DisplayPercent,
		DisplayEnum,
		DisplayStatus,
		DisplayBoolean,
		DisplayFlag,
		DisplayMetrics,
		DisplayLinks,
	}
}

// DisplayColumn is one column of a projected table, or one field of a
// projected card when it carries its value.
type DisplayColumn struct {
	Key   string      `json:"key"`
	Label string      `json:"label"`
	Type  DisplayType `json:"type"`
}

// DisplayField is one labelled value on a projected record card.
type DisplayField struct {
	DisplayColumn

	Value any `json:"value"`
}
