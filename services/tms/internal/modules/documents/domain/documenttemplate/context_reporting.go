package documenttemplate

import "html/template"

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

	Params  []KeyValue
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
	Tone string
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
	Headers []string
	Rows    []ReportRow
	Totals  []string
	Note    string
}

func newReportSampleContext() any {
	return ReportContext{
		Title:       sampleReportTitle,
		Description: "Total charges grouped by customer for the selected period",
		GeneratedAt: "2026-07-15 11:00 CDT",
		RequestedBy: "dispatch@trenova.app",
		Params: []KeyValue{
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
				{Text: sampleShipperName},
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
		Totals:    []string{sampleTotalsLabel, "67", "$142,324.05", "12.9%"},
		RowCount:  67,
		Truncated: false,
		Notice:    "",
	}
}

func newReportDeliverySampleContext() any {
	return ReportDeliveryEmailContext{
		Title:        sampleReportTitle,
		Description:  "Total charges grouped by customer for the selected period",
		Format:       "XLSX",
		CompletedAt:  "2026-07-15 11:00 CDT",
		RequestedBy:  "dispatch@trenova.app",
		RowCount:     67,
		FileSize:     "184 KB",
		Truncated:    false,
		Attached:     true,
		DownloadURL:  template.URL("https://app.example.com/reports/runs"),
		ExpiresAt:    "2026-07-22 11:00 CDT",
		AttachedNote: "The workbook is attached to this message.",
		Digest: ReportDigest{
			Headers: []string{"Customer", "Shipments", "Revenue"},
			Rows: []ReportRow{
				{Cells: []ReportCell{
					{Text: "Halstead Grocery Group"},
					{Text: "42"},
					{Text: "$98,441.20"},
				}},
				{Cells: []ReportCell{
					{Text: sampleShipperName},
					{Text: "17"},
					{Text: "$31,002.85"},
				}},
			},
			Totals: []string{sampleTotalsLabel, "67", "$142,324.05"},
			Note:   "Showing the first 2 of 67 rows.",
		},
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func (r *Registry) registerReportingKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindReportPDF,
		DisplayName: "Report Export",
		Description: "The printable form of any report. One template serves every report " +
			"definition, so columns are positional rather than named.",
		Category:      "Reporting",
		Channels:      []Channel{ChannelPDF},
		Paged:         true,
		sampleFactory: newReportSampleContext,
		Variables: []VariableDefinition{
			titleVariable(
				"The report's name. An export nobody can identify is an export nobody can file.",
			),
			descriptionVariable(),
			{
				Path:        "GeneratedAt",
				Type:        VariableDateTime,
				Description: "When it was run, in the schedule's timezone.",
			},
			requestedByVariable(),
			rowCountVariable(),
			truncatedVariable(),
			{Path: "Notice", Type: VariableString, Description: "The truncation warning text."},
			{
				Path:        "Params",
				Type:        VariableCollection,
				Description: "The filters the report ran with. Print these or a reader cannot tell what they are looking at.",
				Fields:      keyValueFields(),
			},
			{
				Path:        "Headers",
				Type:        VariableStringList,
				Required:    true,
				Description: "Column headings, in order. Required: a table with no headings cannot be read.",
			},
			{
				Path:        "Totals",
				Type:        VariableStringList,
				Description: "The grand-total row, in the same column order. Empty when the report has no totals.",
			},
			{
				Path:        "Rows",
				Type:        VariableCollection,
				Required:    true,
				Description: "The result rows. Required: this is the report.",
				Fields: []VariableDefinition{
					{
						Path:        "Cells",
						Type:        VariableCollection,
						Description: "Cells in the same order as Headers.",
						Fields:      reportCellFields(),
					},
				},
			},
		},
	})

	_ = r.Register(&KindDefinition{
		Kind:          KindReportDeliveryEmail,
		DisplayName:   "Scheduled Report Email",
		Description:   "The message that delivers a scheduled report run.",
		Category:      "Reporting",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newReportDeliverySampleContext,
		Variables: []VariableDefinition{
			titleVariable(
				"The report's name. A recipient subscribed to several schedules needs to know which one arrived.",
			),
			descriptionVariable(),
			{
				Path:        "Format",
				Type:        VariableString,
				Description: "The artifact format, such as XLSX or CSV.",
			},
			{
				Path:        "CompletedAt",
				Type:        VariableDateTime,
				Description: "When the run finished, in the schedule's timezone.",
			},
			requestedByVariable(),
			rowCountVariable(),
			{
				Path:        "FileSize",
				Type:        VariableString,
				Description: "Size of the artifact, human readable.",
			},
			truncatedVariable(),
			{
				Path:        "Attached",
				Type:        VariableBool,
				Description: "True when the artifact rode along with this message. Branch on it: a template that always says \"see attached\" will be wrong whenever the file was too large.",
			},
			{
				Path:        "AttachedNote",
				Type:        VariableString,
				Description: "A sentence explaining whether the file is attached or must be downloaded.",
			},
			{Path: "DownloadURL", Type: VariableString, Description: "Where to download the run."},
			{
				Path:        "ExpiresAt",
				Type:        VariableDateTime,
				Description: "When the download stops working.",
			},
			{
				Path:        "Digest",
				Type:        VariableObject,
				Description: "An inline sample of the results, so a recipient can see the numbers without opening the file.",
				Fields: []VariableDefinition{
					{
						Path:        "Headers",
						Type:        VariableStringList,
						Description: "Sample column headings.",
					},
					{Path: "Totals", Type: VariableStringList, Description: "Sample totals row."},
					{
						Path:        "Note",
						Type:        VariableString,
						Description: "How much of the result the sample covers.",
					},
					{
						Path:        "Rows",
						Type:        VariableCollection,
						Description: "Sample rows.",
						Fields: []VariableDefinition{
							{
								Path:        "Cells",
								Type:        VariableCollection,
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
