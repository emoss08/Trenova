package watchtowersources

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func resourceRecord(resource permission.Resource, label string) *watchtower.RecordRef {
	kind, ok := resource.Kind()
	if !ok {
		return nil
	}
	return &watchtower.RecordRef{Kind: kind, Label: label}
}

func kindRecord(kind permission.RecordKind, label string) *watchtower.RecordRef {
	return &watchtower.RecordRef{Kind: kind, Label: label}
}

func shipmentRecord(proNumber string) *watchtower.RecordRef {
	return resourceRecord(permission.ResourceShipment, proNumber)
}

func fact(label, value string) watchtower.Fact {
	return watchtower.Fact{Label: label, Value: value}
}

func factAt(label string, at int64) watchtower.Fact {
	return watchtower.Fact{Label: label, At: at}
}

func factAtPtr(label string, at *int64) watchtower.Fact {
	if at == nil {
		return watchtower.Fact{Label: label}
	}
	return factAt(label, *at)
}

func deadlineAt(at int64, label string) *watchtower.Deadline {
	if at <= 0 {
		return nil
	}
	return &watchtower.Deadline{At: at, Label: label}
}

func deadlineAtPtr(at *int64, label string) *watchtower.Deadline {
	if at == nil {
		return nil
	}
	return deadlineAt(*at, label)
}

func minutesPhrase(minutes int64) string {
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
