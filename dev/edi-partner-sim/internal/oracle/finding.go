package oracle

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Level string

const (
	LevelInterchange Level = "interchange"
	LevelGroup       Level = "group"
	LevelTransaction Level = "transaction"
	LevelSegment     Level = "segment"
	LevelElement     Level = "element"
)

const (
	codeControlNumberMismatch = "001"
	codeInvalidTerminator     = "004"
	codeGroupCountMismatch    = "021"
	codeControlStructure      = "022"
	codeTrailerMissing        = "023"
	codeDuplicateInterchange  = "025"
	codeInvalidElementSep     = "026"
	codeInvalidComponentSep   = "027"
)

const (
	codeGroupTrailerMissing   = "3"
	codeGroupControlMismatch  = "4"
	codeTransactionCountWrong = "5"
	codeGroupControlSyntax    = "6"
	codeDuplicateGroupControl = "19"
)

const (
	codeSetNotSupported      = "1"
	codeSetTrailerMissing    = "2"
	codeSetControlMismatch   = "3"
	codeSegmentCountWrong    = "4"
	codeSetIdentifierInvalid = "6"
	codeSetControlInvalid    = "7"
	codeDuplicateSetControl  = "23"
)

const (
	codeSegmentUnrecognized  = "1"
	codeSegmentUnexpected    = "2"
	codeSegmentMissing       = "3"
	codeLoopOverMax          = "4"
	codeSegmentOverMax       = "5"
	codeSegmentNotInSet      = "6"
	codeSegmentOutOfSequence = "7"
)

const (
	codeElementMissing        = "1"
	codeConditionalMissing    = "2"
	codeTooManyElements       = "3"
	codeTooShort              = "4"
	codeTooLong               = "5"
	codeInvalidCharacter      = "6"
	codeInvalidCode           = "7"
	codeInvalidDate           = "8"
	codeInvalidTime           = "9"
	codeExclusionViolated     = "10"
	codeTooManyRepetitions    = "12"
	codeTooManyComponents     = "13"
	codeNotUsedElementPresent = "I10"
)

const (
	maxFindingsPerTransaction = 1000
	transactionLoopID         = ""
)

type Finding struct {
	Severity     Severity `json:"severity"`
	Level        Level    `json:"level"`
	Code         string   `json:"code,omitempty"`
	SegmentID    string   `json:"segmentId,omitempty"`
	SegmentIndex int      `json:"segmentIndex,omitempty"`
	Offset       int      `json:"offset"`
	LoopID       string   `json:"loopId,omitempty"`
	ElementPos   int      `json:"elementPos,omitempty"`
	ComponentPos int      `json:"componentPos,omitempty"`
	Value        string   `json:"value,omitempty"`
	Rule         string   `json:"rule,omitempty"`
	Message      string   `json:"message"`
}

func errorAt(level Level, code, rule, message string) Finding {
	return Finding{
		Severity: SeverityError,
		Level:    level,
		Code:     code,
		Rule:     rule,
		Message:  message,
	}
}

func hasErrors(findings []Finding) bool {
	for i := range findings {
		if findings[i].Severity == SeverityError {
			return true
		}
	}
	return false
}

func (f *Finding) X12Code() string {
	if f.Code == "" {
		return ""
	}
	switch f.Level {
	case LevelInterchange:
		return "TA105 " + f.Code
	case LevelGroup:
		return "AK905 " + f.Code
	case LevelTransaction:
		return "AK502 " + f.Code
	case LevelSegment:
		return "AK304 " + f.Code
	case LevelElement:
		return "AK403 " + f.Code
	}
	return f.Code
}
