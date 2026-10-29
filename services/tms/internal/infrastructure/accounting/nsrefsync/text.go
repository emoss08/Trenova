package nsrefsync

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxLineTextLength = 999
)

func RefOf(id string) *nsbinding.RecordRef {
	value := strings.TrimSpace(id)
	if value == "" {
		return nil
	}
	return &nsbinding.RecordRef{ID: value}
}

func CleanText(value string, limit int) string {
	stripped := strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, value)
	return strings.TrimSpace(stringutils.OneLine(stripped, limit))
}

func unixOf(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	value := t.Unix()
	return &value
}
