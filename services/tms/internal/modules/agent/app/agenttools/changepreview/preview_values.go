package changepreview

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func DayLabel(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format(BackfillDayLayout)
}

func RefusalOrFailure(err error) (refusal, failure error) {
	switch {
	case err == nil:
		return nil, nil
	case IsRefusal(err) || errortypes.IsNotFoundError(err):
		return err, nil
	default:
		return nil, err
	}
}

func RecordOf(entity string, id pulid.ID) *agentrecord.RecordRef {
	if id.IsNil() {
		return nil
	}

	return &agentrecord.RecordRef{EntityType: entity, ID: id.String()}
}

func WithNotes(summary string, notes []string) string {
	if len(notes) == 0 {
		return summary
	}

	return summary + " " + strings.Join(notes, "; ") + "."
}

func PinnedVersion(version int64) *int64 {
	return &version
}

func KnownAmount(amount decimal.Decimal) decimal.NullDecimal {
	return decimal.NewNullDecimal(amount)
}

func WouldFail(
	preview *agentpreview.ToolPreview,
	refusal error,
) (*agentpreview.ToolPreview, error) {
	return WarnWouldFail(preview, refusal), nil
}

func MinorAmount(minor int64) decimal.NullDecimal {
	return KnownAmount(money.DecimalFromMinor(minor))
}

func DayText(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format("Jan 2, 2006")
}

func SlicesNonEmpty(values []string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}

	return kept
}
