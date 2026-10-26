package edix12inspect

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/x12"
)

const isaSegmentID = "ISA"

type SeparatorSource string

const (
	SeparatorSourceISA      = SeparatorSource("isa")
	SeparatorSourceEnvelope = SeparatorSource("envelope")
	SeparatorSourceFallback = SeparatorSource("fallback")
)

type DiagnosticSource string

const (
	DiagnosticSourceInspection     = DiagnosticSource("inspection")
	DiagnosticSourceRender         = DiagnosticSource("render")
	DiagnosticSourceValidation     = DiagnosticSource("validation")
	DiagnosticSourceTransform      = DiagnosticSource("transform")
	DiagnosticSourceCondition      = DiagnosticSource("condition")
	DiagnosticSourceSourceContext  = DiagnosticSource("source_context")
	DiagnosticSourcePartnerSetting = DiagnosticSource("partner_setting")
)

type InspectX12Request struct {
	RawX12         string                   `json:"rawX12"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	X12Version     string                   `json:"x12Version"`
	Envelope       *edi.X12EnvelopeSettings `json:"envelope,omitempty"`
	Diagnostics    []edi.Diagnostic         `json:"diagnostics,omitempty"`
}

type InspectOverview struct {
	TransactionSet edi.TransactionSet     `json:"transactionSet"`
	X12Version     string                 `json:"x12Version"`
	Separators     X12Separators          `json:"separators"`
	Summary        InspectSummary         `json:"summary"`
	Envelope       X12Envelope            `json:"envelope"`
	Groups         []X12FunctionalGroup   `json:"groups"`
	Transactions   []X12Transaction       `json:"transactions"`
	Diagnostics    []NormalizedDiagnostic `json:"diagnostics"`
}

type InspectX12Result struct {
	InspectOverview

	RawX12    string       `json:"rawX12"`
	Segments  []X12Segment `json:"segments"`
	Formatted string       `json:"formatted"`

	Reading *x12.Reading `json:"-"`
}

type FileInspection struct {
	InspectOverview

	Interchange bool `json:"interchange"`
}

func (r *InspectX12Result) FileInspection() FileInspection {
	interchange := false
	for index := range r.Segments {
		if r.Segments[index].SegmentID == isaSegmentID {
			interchange = true
			break
		}
	}
	return FileInspection{
		InspectOverview: r.InspectOverview,
		Interchange:     interchange,
	}
}

type X12Separators struct {
	Element     string          `json:"element"`
	Segment     string          `json:"segment"`
	Component   string          `json:"component"`
	Repetition  string          `json:"repetition"`
	Source      SeparatorSource `json:"source"`
	HasConflict bool            `json:"hasConflict"`
}

type InspectSummary struct {
	SegmentCount     int `json:"segmentCount"`
	GroupCount       int `json:"groupCount"`
	TransactionCount int `json:"transactionCount"`
	ErrorCount       int `json:"errorCount"`
	WarningCount     int `json:"warningCount"`
	InfoCount        int `json:"infoCount"`
}

type X12Envelope struct {
	ISAControlNumber string `json:"isaControlNumber,omitempty"`
	IEAControlNumber string `json:"ieaControlNumber,omitempty"`
	ExpectedGroups   int    `json:"expectedGroups,omitempty"`
	ActualGroups     int    `json:"actualGroups"`
}

type X12FunctionalGroup struct {
	Index             int    `json:"index"`
	FunctionalIDCode  string `json:"functionalIdCode,omitempty"`
	GSControlNumber   string `json:"gsControlNumber,omitempty"`
	GEControlNumber   string `json:"geControlNumber,omitempty"`
	ExpectedCount     int    `json:"expectedCount,omitempty"`
	ActualCount       int    `json:"actualCount"`
	StartSegmentIndex int    `json:"startSegmentIndex"`
	EndSegmentIndex   int    `json:"endSegmentIndex,omitempty"`
}

type X12Transaction struct {
	Index             int    `json:"index"`
	TransactionSet    string `json:"transactionSet,omitempty"`
	STControlNumber   string `json:"stControlNumber,omitempty"`
	SEControlNumber   string `json:"seControlNumber,omitempty"`
	ExpectedSegments  int    `json:"expectedSegments,omitempty"`
	ActualSegments    int    `json:"actualSegments"`
	StartSegmentIndex int    `json:"startSegmentIndex"`
	EndSegmentIndex   int    `json:"endSegmentIndex,omitempty"`
}

type X12Segment struct {
	Index             int          `json:"index"`
	TransactionIndex  int          `json:"transactionIndex,omitempty"`
	SegmentID         string       `json:"segmentId"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	Loop              string       `json:"loop,omitempty"`
	Raw               string       `json:"raw"`
	RawWithTerminator string       `json:"rawWithTerminator"`
	StartOffset       int          `json:"startOffset"`
	EndOffset         int          `json:"endOffset"`
	Elements          []X12Element `json:"elements"`
	Malformed         bool         `json:"malformed"`
}

type X12Element struct {
	Position    int            `json:"position"`
	Label       string         `json:"label"`
	DataType    string         `json:"dataType,omitempty"`
	Value       string         `json:"value"`
	Empty       bool           `json:"empty"`
	Required    bool           `json:"required"`
	Known       bool           `json:"known"`
	StartOffset int            `json:"startOffset"`
	EndOffset   int            `json:"endOffset"`
	Components  []X12Component `json:"components"`
}

type X12Component struct {
	Position int    `json:"position"`
	Value    string `json:"value"`
	Empty    bool   `json:"empty"`
}

type NormalizedDiagnostic struct {
	Severity        edi.ValidationSeverity `json:"severity"`
	Code            string                 `json:"code"`
	Source          DiagnosticSource       `json:"source"`
	SegmentID       string                 `json:"segmentId,omitempty"`
	SegmentIndex    int                    `json:"segmentIndex,omitempty"`
	ElementPosition int                    `json:"elementPosition,omitempty"`
	Path            string                 `json:"path,omitempty"`
	Message         string                 `json:"message"`
	SuggestedFix    string                 `json:"suggestedFix,omitempty"`
}
