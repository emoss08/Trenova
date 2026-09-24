package agent

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

type EgressClass string

const (
	EgressNone              = EgressClass("none")
	EgressPersonal          = EgressClass("personal")
	EgressInternal          = EgressClass("internal")
	EgressCustomerVisible   = EgressClass("customer_visible")
	EgressDriverVisible     = EgressClass("driver_visible")
	EgressExternalRecipient = EgressClass("external_recipient")
	EgressMoney             = EgressClass("money")
)

func EgressClasses() []EgressClass {
	return []EgressClass{
		EgressNone,
		EgressPersonal,
		EgressInternal,
		EgressCustomerVisible,
		EgressDriverVisible,
		EgressExternalRecipient,
		EgressMoney,
	}
}

func (c EgressClass) IsValid() bool {
	switch c {
	case EgressNone,
		EgressPersonal,
		EgressInternal,
		EgressCustomerVisible,
		EgressDriverVisible,
		EgressExternalRecipient,
		EgressMoney:
		return true
	default:
		return false
	}
}

func (c EgressClass) String() string { return string(c) }

func (c EgressClass) Leaves() bool {
	switch c {
	case EgressCustomerVisible, EgressDriverVisible, EgressExternalRecipient, EgressMoney:
		return true
	default:
		return false
	}
}

func (c EgressClass) Ceiling() AutonomyTier {
	switch c {
	case EgressCustomerVisible, EgressDriverVisible, EgressExternalRecipient:
		return TierActWithApproval
	default:
		return TierAutoExecute
	}
}

type ToolKind string

const (
	ToolKindQuery   = ToolKind("query")
	ToolKindAction  = ToolKind("action")
	ToolKindRuntime = ToolKind("runtime")
)

func (k ToolKind) IsValid() bool {
	switch k {
	case ToolKindQuery, ToolKindAction, ToolKindRuntime:
		return true
	default:
		return false
	}
}

func (k ToolKind) String() string { return string(k) }

type ToolScope string

const (
	ToolScopeTenant = ToolScope("tenant")
	ToolScopeSelf   = ToolScope("self")
	ToolScopeRun    = ToolScope("run")
)

func (s ToolScope) IsValid() bool {
	switch s {
	case ToolScopeTenant, ToolScopeSelf, ToolScopeRun:
		return true
	default:
		return false
	}
}

func (s ToolScope) String() string { return string(s) }

type ExternalRead string

const (
	ExternalReadNever  = ExternalRead("never")
	ExternalReadAlways = ExternalRead("always")
	ExternalReadMarked = ExternalRead("marked")
)

func (r ExternalRead) IsValid() bool {
	switch r {
	case ExternalReadNever, ExternalReadAlways, ExternalReadMarked:
		return true
	default:
		return false
	}
}

func (r ExternalRead) String() string { return string(r) }

type TaintSource string

const (
	TaintSourceInboundMessage = TaintSource("inbound_message")
	TaintSourceDocument       = TaintSource("document")
	TaintSourceEDI            = TaintSource("edi")
	TaintSourceBankReceipt    = TaintSource("bank_receipt")
	TaintSourceWeather        = TaintSource("weather")
	TaintSourceAttachment     = TaintSource("attachment")
	TaintSourceMemory         = TaintSource("memory")
	TaintSourceRunRecord      = TaintSource("run_record")
	TaintSourceWeb            = TaintSource("web")
	TaintSourceRecordNote     = TaintSource("record_note")
)

func (s TaintSource) IsValid() bool {
	switch s {
	case TaintSourceInboundMessage,
		TaintSourceDocument,
		TaintSourceEDI,
		TaintSourceBankReceipt,
		TaintSourceWeather,
		TaintSourceAttachment,
		TaintSourceMemory,
		TaintSourceRunRecord,
		TaintSourceWeb,
		TaintSourceRecordNote:
		return true
	default:
		return false
	}
}

func (s TaintSource) String() string { return string(s) }

const (
	MaxTaintMarks  = 16
	HeldByTaintKey = "tainted"
)

type TaintMark struct {
	Source   TaintSource `json:"source"`
	ToolName string      `json:"toolName"`
	CallID   string      `json:"callId"`
	Ref      *RecordRef  `json:"ref,omitempty"`
	At       int64       `json:"at"`
}

func (m TaintMark) sameOrigin(other TaintMark) bool {
	if m.Source != other.Source || m.ToolName != other.ToolName {
		return false
	}
	if m.Ref != nil || other.Ref != nil {
		return m.Ref != nil && other.Ref != nil &&
			m.Ref.EntityType == other.Ref.EntityType && m.Ref.ID == other.Ref.ID
	}

	return m.CallID == other.CallID
}

type RunTaint struct {
	Marks []TaintMark `json:"marks"`
}

func (t *RunTaint) Add(mark TaintMark) bool {
	if t == nil || !mark.Source.IsValid() {
		return false
	}
	if slices.ContainsFunc(t.Marks, mark.sameOrigin) {
		return false
	}
	if len(t.Marks) >= MaxTaintMarks {
		return false
	}
	t.Marks = append(t.Marks, mark)

	return true
}

func (t *RunTaint) Tainted() bool {
	return t != nil && len(t.Marks) > 0
}

func (t *RunTaint) Merge(other *RunTaint) {
	if t == nil || other == nil {
		return
	}
	for _, mark := range other.Marks {
		t.Add(mark)
	}
}

func (t *RunTaint) Clone() *RunTaint {
	if t == nil {
		return nil
	}

	return &RunTaint{Marks: slices.Clone(t.Marks)}
}

func (t *RunTaint) Absorb(marks []TaintMark) []TaintMark {
	if t == nil || len(marks) == 0 {
		return nil
	}

	added := make([]TaintMark, 0, len(marks))
	for _, mark := range marks {
		if t.Add(mark) {
			added = append(added, mark)
		}
	}

	return added
}

func (t *RunTaint) Unknown() bool {
	return t == nil
}

func (t *RunTaint) Sources() []TaintSource {
	if t == nil {
		return nil
	}

	sources := make([]TaintSource, 0, len(t.Marks))
	for _, mark := range t.Marks {
		if !slices.Contains(sources, mark.Source) {
			sources = append(sources, mark.Source)
		}
	}

	return sources
}

type TaintCarrier interface {
	TaintedRecords() []RecordRef
}

const (
	TaintEntityInboundMessage  = "inbound_message"
	TaintEntityDocument        = "document"
	TaintEntityEDIInboundFile  = "edi_inbound_file"
	TaintEntityBankReceipt     = "bank_receipt"
	TaintEntityAgentMemory     = "agent_memory"
	TaintEntityAgentRun        = "agent_run"
	TaintEntityShipmentComment = "shipment_comment"
)

func (s SubjectType) TaintSource() (TaintSource, string, bool) {
	switch s {
	case SubjectInboundMessage:
		return TaintSourceInboundMessage, TaintEntityInboundMessage, true
	case SubjectDocument:
		return TaintSourceDocument, TaintEntityDocument, true
	case SubjectEDIInboundFile:
		return TaintSourceEDI, TaintEntityEDIInboundFile, true
	case SubjectBankReceipt:
		return TaintSourceBankReceipt, TaintEntityBankReceipt, true
	default:
		return "", "", false
	}
}

func SubjectTaint(subjectType SubjectType, subjectID string, at int64) (TaintMark, bool) {
	source, entity, ok := subjectType.TaintSource()
	if !ok || subjectID == "" {
		return TaintMark{}, false
	}

	return TaintMark{
		Source: source,
		Ref:    &RecordRef{EntityType: entity, ID: subjectID},
		At:     at,
	}, true
}

func RunRecordTaint(runID pulid.ID, at int64) *RunTaint {
	mark := TaintMark{Source: TaintSourceRunRecord, At: at}
	if runID.IsNotNil() {
		mark.Ref = &RecordRef{EntityType: TaintEntityAgentRun, ID: runID.String()}
	}

	taint := &RunTaint{}
	taint.Add(mark)

	return taint
}

func AttachmentTaint(documentID string, at int64) TaintMark {
	return TaintMark{
		Source: TaintSourceAttachment,
		Ref:    &RecordRef{EntityType: TaintEntityDocument, ID: documentID},
		At:     at,
	}
}
