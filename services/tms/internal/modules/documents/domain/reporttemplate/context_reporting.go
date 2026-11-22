package reporttemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// ReportContext is the data a tabular report export renders against.
//
// The shape mirrors what the reporting executor already streams, so a report of any
// definition renders through one template: columns are positional, and a row is a
// slice of cells in the same order as Headers.
type ReportContext struct {
	Title       string
	Description string
	GeneratedAt string
	RequestedBy string

	Params  []templatecatalog.KeyValue
	Headers []string
	Rows    []ReportRow
	Totals  []string

	RowCount  int64
	Truncated bool
	Notice    string
}

// ReportRow is one row of a report.
type ReportRow struct {
	Cells []ReportCell
}

// ReportCell carries the emphasis the report definition asked for alongside the
// text, so a template can colour an exception without re-deriving why it is one.
type ReportCell struct {
	Text string
	// Tone is positive, warning, negative, neutral, or empty.
	Tone    string
	Numeric bool
}

// ReportDeliveryEmailContext is the message that delivers a scheduled report.
type ReportDeliveryEmailContext struct {
	Title       string
	Description string
	Format      string
	CompletedAt string
	RequestedBy string

	RowCount  int64
	FileSize  string
	Truncated bool

	// Attached says whether the artifact rode along or was too large and has to be
	// downloaded. A template that does not branch on it will tell half its
	// recipients to look for a file that is not there.
	Attached bool

	// DownloadURL is typed template.URL because this system builds it from the
	// configured delivery base URL, not from anything a user typed.
	DownloadURL  template.URL
	ExpiresAt    string
	AttachedNote string

	// Digest is a sample of the result, so a recipient can see the numbers without
	// opening anything.
	Digest ReportDigest
}

// ReportDigest is the inline sample of a report's results.
type ReportDigest struct {
	Headers []ReportColumn
	Rows    []ReportRow
	Totals  []ReportCell
	Note    string
}

type ReportColumn struct {
	Label   string
	Numeric bool
}

func newReportSampleContext() any {
	return ReportContext{
		Title:       templatecatalog.SampleReportTitle,
		Description: "Total charges grouped by customer for the selected period",
		GeneratedAt: "2026-07-15 11:00 CDT",
		RequestedBy: "dispatch@example.com",
		Params: []templatecatalog.KeyValue{
			{Label: "Period", Value: "2026-07-01 to 2026-07-14"},
			{Label: "Status", Value: "Delivered"},
		},
		Headers: []string{"Customer", "Shipments", "Revenue", "Margin"},
		Rows: []ReportRow{
			{Cells: []ReportCell{
				{Text: "Halstead Grocery Group"},
				{Text: "42"},
				{Text: "$98,441.20"},
				{Text: "18.4%", Tone: "positive"},
			}},
			{Cells: []ReportCell{
				{Text: templatecatalog.SampleShipperName},
				{Text: "17"},
				{Text: "$31,002.85"},
				{Text: "4.1%", Tone: "warning"},
			}},
			{Cells: []ReportCell{
				{Text: "Cordell Manufacturing"},
				{Text: "8"},
				{Text: "$12,880.00"},
				{Text: "-2.6%", Tone: "negative"},
			}},
		},
		Totals:    []string{templatecatalog.SampleTotalsLabel, "67", "$142,324.05", "12.9%"},
		RowCount:  67,
		Truncated: false,
		Notice:    "",
	}
}

func newReportDeliverySampleContext() any {
	return ReportDeliveryEmailContext{
		Title:        templatecatalog.SampleReportTitle,
		Description:  "Total charges grouped by customer for the selected period",
		Format:       "XLSX",
		CompletedAt:  "2026-07-15 11:00 CDT",
		RequestedBy:  "dispatch@example.com",
		RowCount:     67,
		FileSize:     "184 KB",
		Truncated:    false,
		Attached:     true,
		DownloadURL:  template.URL("https://app.example.com/reports/runs"),
		ExpiresAt:    "2026-07-22 11:00 CDT",
		AttachedNote: "The workbook is attached to this message.",
		Digest: ReportDigest{
			Headers: []ReportColumn{
				{Label: "Customer"},
				{Label: "Shipments", Numeric: true},
				{Label: "Revenue", Numeric: true},
			},
			Rows: []ReportRow{
				{Cells: []ReportCell{
					{Text: "Halstead Grocery Group", Tone: "neutral"},
					{Text: "42", Numeric: true},
					{Text: "$98,441.20", Numeric: true, Tone: "positive"},
				}},
				{Cells: []ReportCell{
					{Text: templatecatalog.SampleShipperName, Tone: "neutral"},
					{Text: "17", Numeric: true},
					{Text: "$31,002.85", Numeric: true},
				}},
			},
			Totals: []ReportCell{
				{Text: templatecatalog.SampleTotalsLabel},
				{Text: "67", Numeric: true},
				{Text: "$142,324.05", Numeric: true},
			},
			Note: "Showing the first 2 of 67 rows.",
		},
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func RegisterReportingKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindReportPDF,
		DisplayName: "Report Export",
		Description: "The printable form of any report. One template serves every report " +
			"definition, so columns are positional rather than named.",
		Category:      "Reporting",
		Channels:      []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:         true,
		SampleFactory: newReportSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			titleVariable(
				"The report's name. An export nobody can identify is an export nobody can file.",
			),
			descriptionVariable(),
			{
				Path:        "GeneratedAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "When it was run, in the schedule's timezone.",
			},
			requestedByVariable(),
			templatecatalog.RowCountVariable(),
			templatecatalog.TruncatedVariable(),
			{
				Path:        "Notice",
				Type:        templatecatalog.VariableString,
				Description: "The truncation warning text.",
			},
			{
				Path:        "Params",
				Type:        templatecatalog.VariableCollection,
				Description: "The filters the report ran with. Print these or a reader cannot tell what they are looking at.",
				Fields:      templatecatalog.KeyValueFields(),
			},
			{
				Path:        "Headers",
				Type:        templatecatalog.VariableStringList,
				Required:    true,
				Description: "Column headings, in order. Required: a table with no headings cannot be read.",
			},
			{
				Path:        "Totals",
				Type:        templatecatalog.VariableStringList,
				Description: "The grand-total row, in the same column order. Empty when the report has no totals.",
			},
			{
				Path:        "Rows",
				Type:        templatecatalog.VariableCollection,
				Required:    true,
				Description: "The result rows. Required: this is the report.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Cells",
						Type:        templatecatalog.VariableCollection,
						Description: "Cells in the same order as Headers.",
						Fields:      reportCellFields(),
					},
				},
			},
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindReportDeliveryEmail,
		DisplayName: "Scheduled Report Email",
		Description: "The message that delivers a scheduled report run.",
		Category:    "Reporting",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newReportDeliverySampleContext,
		Variables: []templatecatalog.VariableDefinition{
			titleVariable(
				"The report's name. A recipient subscribed to several schedules needs to know which one arrived.",
			),
			descriptionVariable(),
			{
				Path:        "Format",
				Type:        templatecatalog.VariableString,
				Description: "The artifact format, such as XLSX or CSV.",
			},
			{
				Path:        "CompletedAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "When the run finished, in the schedule's timezone.",
			},
			requestedByVariable(),
			templatecatalog.RowCountVariable(),
			{
				Path:        "FileSize",
				Type:        templatecatalog.VariableString,
				Description: "Size of the artifact, human readable.",
			},
			templatecatalog.TruncatedVariable(),
			{
				Path:        "Attached",
				Type:        templatecatalog.VariableBool,
				Description: "True when the artifact rode along with this message. Branch on it: a template that always says \"see attached\" will be wrong whenever the file was too large.",
			},
			{
				Path:        "AttachedNote",
				Type:        templatecatalog.VariableString,
				Description: "A sentence explaining whether the file is attached or must be downloaded.",
			},
			{
				Path:        "DownloadURL",
				Type:        templatecatalog.VariableString,
				Description: "Where to download the run.",
			},
			{
				Path:        "ExpiresAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "When the download stops working.",
			},
			{
				Path:        "Digest",
				Type:        templatecatalog.VariableObject,
				Description: "An inline sample of the results, so a recipient can see the numbers without opening the file.",
				Fields: []templatecatalog.VariableDefinition{
					{
						Path:        "Headers",
						Type:        templatecatalog.VariableCollection,
						Description: "Sample column headings.",
						Fields: []templatecatalog.VariableDefinition{
							{
								Path:        "Label",
								Type:        templatecatalog.VariableString,
								Description: "The column heading.",
							},
							numericVariable(
								"True when the column holds numbers, which read right-aligned.",
							),
						},
					},
					{
						Path:        "Totals",
						Type:        templatecatalog.VariableCollection,
						Description: "Sample totals row, in header order.",
						Fields:      reportCellFields(),
					},
					{
						Path:        "Note",
						Type:        templatecatalog.VariableString,
						Description: "How much of the result the sample covers.",
					},
					{
						Path:        "Rows",
						Type:        templatecatalog.VariableCollection,
						Description: "Sample rows.",
						Fields: []templatecatalog.VariableDefinition{
							{
								Path:        "Cells",
								Type:        templatecatalog.VariableCollection,
								Description: "Cells in header order.",
								Fields:      reportCellFields(),
							},
						},
					},
				},
			},
		},
	})
}

func titleVariable(description string) templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "Title",
		Type:        templatecatalog.VariableString,
		Required:    true,
		Description: description,
	}
}

func descriptionVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "Description",
		Type:        templatecatalog.VariableString,
		Description: "What the report shows.",
	}
}

func requestedByVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "RequestedBy",
		Type:        templatecatalog.VariableString,
		Description: "Who ran the report or owns the schedule.",
	}
}

func reportCellFields() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "Text",
			Type:        templatecatalog.VariableString,
			Description: "The formatted cell value.",
		},
		{
			Path: "Tone",
			Type: templatecatalog.VariableString,
			Description: "Emphasis the report asked for: positive, warning, negative, " +
				"neutral, or blank. Use it to colour exceptions.",
		},
		numericVariable("True when the cell holds a number, which reads right-aligned."),
	}
}

func numericVariable(description string) templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        "Numeric",
		Type:        templatecatalog.VariableBool,
		Description: description,
	}
}
