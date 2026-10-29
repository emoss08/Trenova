package watchtoweritem

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func ResourceRecord(resource permission.Resource, label string) *watchtower.RecordRef {
	kind, ok := resource.Kind()
	if !ok {
		return nil
	}
	return &watchtower.RecordRef{Kind: kind, Label: label}
}

func KindRecord(kind permission.RecordKind, label string) *watchtower.RecordRef {
	return &watchtower.RecordRef{Kind: kind, Label: label}
}

func ShipmentRecord(proNumber string) *watchtower.RecordRef {
	return ResourceRecord(permission.ResourceShipment, proNumber)
}

func Fact(label, value string) watchtower.Fact {
	return watchtower.Fact{Label: label, Value: value}
}

func FactAt(label string, at int64) watchtower.Fact {
	return watchtower.Fact{Label: label, At: at}
}

func FactAtPtr(label string, at *int64) watchtower.Fact {
	if at == nil {
		return watchtower.Fact{Label: label}
	}
	return FactAt(label, *at)
}

func DeadlineAt(at int64, label string) *watchtower.Deadline {
	if at <= 0 {
		return nil
	}
	return &watchtower.Deadline{At: at, Label: label}
}

func DeadlineAtPtr(at *int64, label string) *watchtower.Deadline {
	if at == nil {
		return nil
	}
	return DeadlineAt(*at, label)
}

func MinutesPhrase(minutes int64) string {
	if minutes < 60 {
		return strconv.FormatInt(minutes, 10) + "m"
	}
	hours := minutes / 60
	rest := minutes % 60
	if rest == 0 {
		return strconv.FormatInt(hours, 10) + "h"
	}
	return strconv.FormatInt(hours, 10) + "h " + strconv.FormatInt(rest, 10) + "m"
}
