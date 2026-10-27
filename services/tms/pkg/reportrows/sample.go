package reportrows

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/shopspring/decimal"
)

const (
	MaxSampleRows  = 20
	MaxSampleBytes = 48 * 1024
)

type Sample struct {
	Schema []Column `json:"schema"`
	Rows   [][]any  `json:"rows"`
	Totals []any    `json:"totals,omitempty"`
	More   bool     `json:"more,omitempty"`
}

func RawValue(value any) any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case decimal.Decimal:
		return v.String()
	case time.Time:
		return v.Unix()
	default:
		return v
	}
}

func RawRow(row []any) []any {
	if row == nil {
		return nil
	}
	out := make([]any, len(row))
	for i := range row {
		out[i] = RawValue(row[i])
	}

	return out
}

func (s *Sample) Fit() *Sample {
	if s == nil {
		return nil
	}
	for len(s.Rows) > 0 {
		encoded, err := sonic.Marshal(s)
		if err == nil && len(encoded) <= MaxSampleBytes {
			return s
		}
		s.Rows = s.Rows[:len(s.Rows)-1]
		s.More = true
	}
	encoded, err := sonic.Marshal(s)
	if err != nil || len(encoded) > MaxSampleBytes {
		return nil
	}

	return s
}
