package tenant

import (
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
)

var requiredSequenceTypes = []SequenceType{
	SequenceTypeProNumber,
	SequenceTypeConsolidation,
	SequenceTypeInvoice,
	SequenceTypeWorkOrder,
	SequenceTypeJournalBatch,
	SequenceTypeJournalEntry,
	SequenceTypeManualJournalRequest,
}

var sequenceTypeOrder = map[SequenceType]int{
	SequenceTypeProNumber:            1,
	SequenceTypeConsolidation:        2,
	SequenceTypeInvoice:              3,
	SequenceTypeWorkOrder:            4,
	SequenceTypeJournalBatch:         5,
	SequenceTypeJournalEntry:         6,
	SequenceTypeManualJournalRequest: 7,
	SequenceTypeCreditMemo:           8,
	SequenceTypeDebitMemo:            9,
}

func RequiredSequenceTypes() []SequenceType {
	return append([]SequenceType(nil), requiredSequenceTypes...)
}

func IsSupportedSequenceType(sequenceType SequenceType) bool {
	_, ok := sequenceTypeOrder[sequenceType]
	return ok
}

func SequenceTypeSortOrder(sequenceType SequenceType) int {
	if order, ok := sequenceTypeOrder[sequenceType]; ok {
		return order
	}

	return len(sequenceTypeOrder) + 1
}

func DefaultSequenceFormat(sequenceType SequenceType) (*SequenceFormat, error) {
	switch sequenceType {
	case SequenceTypeProNumber:
		return &SequenceFormat{Type: sequenceType, Prefix: "S", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 4, IncludeRandomDigits: true, RandomDigitsCount: 6}, nil
	case SequenceTypeConsolidation:
		return &SequenceFormat{Type: sequenceType, Prefix: "C", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 5}, nil
	case SequenceTypeInvoice:
		return &SequenceFormat{Type: sequenceType, Prefix: "INV", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeCreditMemo:
		return &SequenceFormat{Type: sequenceType, Prefix: "CM", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeDebitMemo:
		return &SequenceFormat{Type: sequenceType, Prefix: "DM", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeWorkOrder:
		return &SequenceFormat{Type: sequenceType, Prefix: "WO", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeJournalBatch:
		return &SequenceFormat{Type: sequenceType, Prefix: "JB", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeJournalEntry:
		return &SequenceFormat{Type: sequenceType, Prefix: "JE", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	case SequenceTypeManualJournalRequest:
		return &SequenceFormat{Type: sequenceType, Prefix: "MJR", IncludeYear: true, YearDigits: 2, IncludeMonth: true, SequenceDigits: 6}, nil
	default:
		return nil, fmt.Errorf("invalid sequence type: %s", sequenceType)
	}
}

func DefaultSequenceConfig(orgID, buID pulid.ID, sequenceType SequenceType) *SequenceConfig {
	format, err := DefaultSequenceFormat(sequenceType)
	if err != nil {
		return nil
	}

	return &SequenceConfig{
		ID:                      pulid.MustNew("sqcfg_"),
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		SequenceType:            sequenceType,
		Prefix:                  format.Prefix,
		IncludeYear:             format.IncludeYear,
		YearDigits:              int16(format.YearDigits),
		IncludeMonth:            format.IncludeMonth,
		IncludeWeekNumber:       format.IncludeWeekNumber,
		IncludeDay:              format.IncludeDay,
		SequenceDigits:          int16(format.SequenceDigits),
		IncludeLocationCode:     format.IncludeLocationCode,
		IncludeRandomDigits:     format.IncludeRandomDigits,
		RandomDigitsCount:       int16(format.RandomDigitsCount),
		IncludeCheckDigit:       format.IncludeCheckDigit,
		IncludeBusinessUnitCode: format.IncludeBusinessUnitCode,
		UseSeparators:           format.UseSeparators,
		SeparatorChar:           format.SeparatorChar,
		AllowCustomFormat:       format.AllowCustomFormat,
		CustomFormat:            format.CustomFormat,
		Version:                 0,
	}
}
