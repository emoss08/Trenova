package intelkit

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/shared/stringutils"
)

const defaultDocketPrefix = "MC"

func AuthorityStatus(value string) carrierintel.AuthorityStatus {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "A", "ACTIVE", "AUTHORIZED", "Y":
		return carrierintel.AuthorityStatusActive
	case "I", "INACTIVE":
		return carrierintel.AuthorityStatusInactive
	case "", "N", "NONE", "NO":
		return carrierintel.AuthorityStatusNone
	case "R", "REVOKED", "REVOCATION":
		return carrierintel.AuthorityStatusRevoked
	default:
		return carrierintel.AuthorityStatusUnknown
	}
}

func SafetyRating(value string) carrierintel.SafetyRating {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "S", "SATISFACTORY":
		return carrierintel.SafetyRatingSatisfactory
	case "C", "CONDITIONAL":
		return carrierintel.SafetyRatingConditional
	case "U", "UNSATISFACTORY":
		return carrierintel.SafetyRatingUnsatisfactory
	default:
		return carrierintel.SafetyRatingNotRated
	}
}

func USDOTStatus(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "":
		return ""
	case "A", "ACTIVE", "Y":
		return "ACTIVE"
	case "I", "INACTIVE", "N":
		return "INACTIVE"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func SplitDocket(value string) (prefix, number string) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "", ""
	}
	split := strings.IndexFunc(trimmed, func(r rune) bool { return r >= '0' && r <= '9' })
	if split < 0 {
		return "", ""
	}
	prefix = strings.TrimRight(strings.TrimSpace(trimmed[:split]), "-# ")
	number = stringutils.DigitsOnly(trimmed[split:])
	return prefix, number
}

func DocketPrefixOrDefault(prefix string) string {
	if trimmed := strings.ToUpper(strings.TrimSpace(prefix)); trimmed != "" {
		return trimmed
	}
	return defaultDocketPrefix
}

func SplitList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		for part := range strings.FieldsFuncSeq(value, isListSeparator) {
			if trimmed := strings.TrimSpace(part); trimmed != "" && !slices.Contains(out, trimmed) {
				out = append(out, trimmed)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isListSeparator(r rune) bool {
	return r == ';'
}

type AddressParts struct {
	Full        string
	Street      string
	City        string
	State       string
	PostalCode  string
	Country     string
	Undelivered *bool
}

func Address(parts AddressParts) *carrierintel.Address {
	line1 := strings.TrimSpace(parts.Street)
	if line1 == "" && strings.TrimSpace(parts.City) == "" {
		line1 = strings.TrimSpace(parts.Full)
	}
	address := &carrierintel.Address{
		Line1:       line1,
		City:        strings.TrimSpace(parts.City),
		State:       strings.ToUpper(strings.TrimSpace(parts.State)),
		PostalCode:  strings.TrimSpace(parts.PostalCode),
		Country:     strings.ToUpper(strings.TrimSpace(parts.Country)),
		Undelivered: parts.Undelivered,
	}
	if address.IsZero() {
		return nil
	}
	return address
}
