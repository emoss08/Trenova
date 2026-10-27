package x12

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/wasmmodule"
)

type Severity string

const (
	SeverityError   = Severity("error")
	SeverityWarning = Severity("warning")
	SeverityInfo    = Severity("info")
)

type Level string

const (
	LevelFile        = Level("file")
	LevelInterchange = Level("interchange")
	LevelGroup       = Level("group")
	LevelTransaction = Level("transaction")
	LevelSegment     = Level("segment")
	LevelElement     = Level("element")
)

type TransactionStatus string

const (
	TransactionAccepted           = TransactionStatus("accepted")
	TransactionAcceptedWithErrors = TransactionStatus("acceptedWithErrors")
	TransactionRejected           = TransactionStatus("rejected")
)

type SeparatorSource string

const (
	SeparatorSourceISA      = SeparatorSource("isa")
	SeparatorSourceRequest  = SeparatorSource("request")
	SeparatorSourceDetected = SeparatorSource("detected")
)

type LineEnding string

const (
	LineEndingNone = LineEnding("none")
	LineEndingLF   = LineEnding("lf")
	LineEndingCRLF = LineEnding("crLf")
	LineEndingCR   = LineEnding("cr")
)

type Separators struct {
	Element    string     `json:"element"`
	Component  string     `json:"component"`
	Repetition string     `json:"repetition"`
	Segment    string     `json:"segment"`
	LineEnding LineEnding `json:"lineEnding,omitempty"`
}

type GuideChoice struct {
	TransactionSet string                 `json:"transactionSet"`
	Guide          sonic.NoCopyRawMessage `json:"guide,omitempty"`
	Builtin        string                 `json:"builtin,omitempty"`
}

type ReadRequest struct {
	Text            string        `json:"text"`
	Separators      *Separators   `json:"separators,omitempty"`
	Guides          []GuideChoice `json:"guides,omitempty"`
	IncludeSegments bool          `json:"includeSegments"`
	Fragment        bool          `json:"fragment,omitempty"`
}

type Reading struct {
	Separators      Separators                `json:"separators"`
	SeparatorSource SeparatorSource           `json:"separatorSource"`
	Interchanges    []Interchange             `json:"interchanges"`
	Segments        []Segment                 `json:"segments,omitempty"`
	Definitions     map[string]SegmentSummary `json:"definitions,omitempty"`
	Findings        []Finding                 `json:"findings"`
	Summary         Summary                   `json:"summary"`
}

type Usage string

const (
	UsageMandatory   = Usage("mandatory")
	UsageOptional    = Usage("optional")
	UsageConditional = Usage("conditional")
	UsageNotUsed     = Usage("notUsed")
)

type SegmentSummary struct {
	Name     string           `json:"name"`
	Elements []ElementSummary `json:"elements"`
}

type ElementSummary struct {
	Reference  string           `json:"ref"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Usage      Usage            `json:"usage"`
	Components []ElementSummary `json:"components,omitempty"`
}

type Interchange struct {
	Index          int                       `json:"index"`
	FirstSegment   int                       `json:"firstSegment"`
	LastSegment    int                       `json:"lastSegment"`
	Header         *ISAHeader                `json:"header,omitempty"`
	Separators     Separators                `json:"separators"`
	FixedWidth     bool                      `json:"fixedWidth"`
	Trailer        *Trailer                  `json:"trailer,omitempty"`
	Groups         []Group                   `json:"groups"`
	Findings       []Finding                 `json:"findings"`
	Acknowledgment InterchangeAcknowledgment `json:"acknowledgment"`
}

type ISAHeader struct {
	AuthorizationQualifier  string `json:"authorizationQualifier"`
	SecurityQualifier       string `json:"securityQualifier"`
	SenderQualifier         string `json:"senderQualifier"`
	SenderID                string `json:"senderId"`
	ReceiverQualifier       string `json:"receiverQualifier"`
	ReceiverID              string `json:"receiverId"`
	Date                    string `json:"date"`
	Time                    string `json:"time"`
	Version                 string `json:"version"`
	ControlNumber           string `json:"controlNumber"`
	AcknowledgmentRequested string `json:"acknowledgmentRequested"`
	Usage                   string `json:"usage"`
}

type InterchangeAcknowledgment struct {
	Code string `json:"code"`
	Note string `json:"note"`
}

type Group struct {
	Index          int                 `json:"index"`
	FirstSegment   int                 `json:"firstSegment"`
	LastSegment    int                 `json:"lastSegment"`
	Header         *GSHeader           `json:"header,omitempty"`
	Trailer        *Trailer            `json:"trailer,omitempty"`
	Transactions   []Transaction       `json:"transactions"`
	Findings       []Finding           `json:"findings"`
	Acknowledgment GroupAcknowledgment `json:"acknowledgment"`
}

type GSHeader struct {
	FunctionalID  string `json:"functionalId"`
	Sender        string `json:"sender"`
	Receiver      string `json:"receiver"`
	Date          string `json:"date"`
	Time          string `json:"time"`
	ControlNumber string `json:"controlNumber"`
	Agency        string `json:"agency"`
	Version       string `json:"version"`
}

type GroupAcknowledgment struct {
	Code     string   `json:"code"`
	Included int      `json:"included"`
	Received int      `json:"received"`
	Accepted int      `json:"accepted"`
	Errors   []string `json:"errors,omitempty"`
}

type Transaction struct {
	Index          int               `json:"index"`
	FirstSegment   int               `json:"firstSegment"`
	LastSegment    int               `json:"lastSegment"`
	TransactionSet string            `json:"transactionSet"`
	ControlNumber  string            `json:"controlNumber"`
	Convention     string            `json:"convention,omitempty"`
	Guide          string            `json:"guide"`
	Trailer        *Trailer          `json:"trailer,omitempty"`
	Status         TransactionStatus `json:"status"`
	Findings       []Finding         `json:"findings"`
}

type Trailer struct {
	Segment       int    `json:"segment"`
	Count         string `json:"count"`
	ControlNumber string `json:"controlNumber"`
}

type Segment struct {
	Index       int      `json:"index"`
	ID          string   `json:"id"`
	Start       int      `json:"start"`
	End         int      `json:"end"`
	Interchange int      `json:"interchange"`
	Transaction *int     `json:"transaction,omitempty"`
	Position    *int     `json:"position,omitempty"`
	LoopPath    string   `json:"loopPath,omitempty"`
	Elements    []string `json:"elements"`
}

type Finding struct {
	Kind       string             `json:"kind"`
	Severity   Severity           `json:"severity"`
	Level      Level              `json:"level"`
	Code       string             `json:"code,omitempty"`
	Segment    *int               `json:"segment,omitempty"`
	SegmentID  string             `json:"segmentId,omitempty"`
	Position   *int               `json:"position,omitempty"`
	LoopID     string             `json:"loopId,omitempty"`
	Element    *int               `json:"element,omitempty"`
	Component  *int               `json:"component,omitempty"`
	Repetition *int               `json:"repetition,omitempty"`
	ElementRef string             `json:"elementRef,omitempty"`
	Value      string             `json:"value,omitempty"`
	Span       *[2]int            `json:"span,omitempty"`
	Message    wasmmodule.Message `json:"message"`
}

func (f *Finding) IsError() bool {
	return f.Severity == SeverityError
}

type Summary struct {
	Segments     int `json:"segments"`
	Interchanges int `json:"interchanges"`
	Groups       int `json:"groups"`
	Transactions int `json:"transactions"`
	Rejected     int `json:"rejected"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
	Infos        int `json:"infos"`
}

type AcknowledgmentKind string

const (
	AcknowledgmentKind997 = AcknowledgmentKind("997")
	AcknowledgmentKind999 = AcknowledgmentKind("999")
)

type AcknowledgeRequest struct {
	Kind            AcknowledgmentKind `json:"kind"`
	Separators      *Separators        `json:"separators,omitempty"`
	Acknowledgments []AcknowledgeInput `json:"acknowledgments"`
}

type AcknowledgeInput struct {
	Group    *Group                 `json:"group"`
	Envelope AcknowledgmentEnvelope `json:"envelope"`
}

type AcknowledgmentEnvelope struct {
	SenderQualifier          string `json:"senderQualifier"`
	SenderID                 string `json:"senderId"`
	ReceiverQualifier        string `json:"receiverQualifier"`
	ReceiverID               string `json:"receiverId"`
	ApplicationSender        string `json:"applicationSender"`
	ApplicationReceiver      string `json:"applicationReceiver"`
	Date                     string `json:"date"`
	Time                     string `json:"time"`
	InterchangeControlNumber string `json:"interchangeControlNumber"`
	GroupControlNumber       string `json:"groupControlNumber"`
	TransactionControlNumber string `json:"transactionControlNumber"`
	Usage                    string `json:"usage"`
	InterchangeVersion       string `json:"interchangeVersion,omitempty"`
}

type Acknowledgment struct {
	Text         string `json:"text"`
	Code         string `json:"code"`
	SegmentCount int    `json:"segmentCount"`
	Transactions int    `json:"transactions"`
}

type WriteRequest struct {
	Separators *Separators `json:"separators,omitempty"`
	Segments   [][]any     `json:"segments"`
	Counts     bool        `json:"counts,omitempty"`
}

type Written struct {
	Text         string    `json:"text"`
	SegmentCount int       `json:"segmentCount"`
	Findings     []Finding `json:"findings"`
}

type GuideSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	TransactionSet  string `json:"transactionSet"`
	FunctionalGroup string `json:"functionalGroup"`
	Version         string `json:"version"`
}

type GuideProblem struct {
	Path    string             `json:"path"`
	Kind    string             `json:"kind"`
	Message wasmmodule.Message `json:"message"`
}
