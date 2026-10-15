package factoringremit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

type Field string

const (
	FieldInvoiceNumber   = Field("invoiceNumber")
	FieldFactorReference = Field("factorReference")
	FieldDebtor          = Field("debtor")
	FieldKind            = Field("kind")
	FieldFace            = Field("face")
	FieldAdvance         = Field("advance")
	FieldFee             = Field("fee")
	FieldReserve         = Field("reserve")
	FieldCash            = Field("cash")
	FieldDate            = Field("date")
)

func Fields() []Field {
	return []Field{
		FieldInvoiceNumber,
		FieldFactorReference,
		FieldDebtor,
		FieldKind,
		FieldFace,
		FieldAdvance,
		FieldFee,
		FieldReserve,
		FieldCash,
		FieldDate,
	}
}

func amountFields() []Field {
	return []Field{FieldCash, FieldAdvance, FieldFace, FieldReserve}
}

func (f Field) String() string { return string(f) }

func (f Field) IsValid() bool {
	for _, known := range Fields() {
		if known == f {
			return true
		}
	}
	return false
}

func (f Field) Label() string {
	switch f {
	case FieldInvoiceNumber:
		return "Invoice number"
	case FieldFactorReference:
		return "Factor reference"
	case FieldDebtor:
		return "Debtor"
	case FieldKind:
		return "Transaction type"
	case FieldFace:
		return "Invoice amount"
	case FieldAdvance:
		return "Advance"
	case FieldFee:
		return "Fee"
	case FieldReserve:
		return "Reserve"
	case FieldCash:
		return "Amount paid"
	case FieldDate:
		return "Date"
	}
	return string(f)
}

type Mapping map[Field]int

var ErrUnknownField = errors.New("unknown remittance field")

func MappingFromStrings(raw map[string]int) (Mapping, error) {
	mapping := make(Mapping, len(raw))
	for name, index := range raw {
		field := Field(name)
		if !field.IsValid() {
			return nil, errors.Join(ErrUnknownField, errors.New(name))
		}
		if index < 0 {
			return nil, errors.New("column index for " + name + " cannot be negative")
		}
		mapping[field] = index
	}
	return mapping, nil
}

func (m Mapping) Strings() map[string]int {
	out := make(map[string]int, len(m))
	for field, index := range m {
		out[string(field)] = index
	}
	return out
}

func (m Mapping) Has(field Field) bool {
	_, ok := m[field]
	return ok
}

func (m Mapping) FitsHeaders(width int) bool {
	for _, index := range m {
		if index >= width {
			return false
		}
	}
	return true
}

type MappingError struct {
	Field   Field
	Message string
}

func (p MappingError) Error() string { return p.Message }

func Validate(mapping Mapping) []MappingError {
	problems := make([]MappingError, 0, 3)
	if !mapping.Has(FieldInvoiceNumber) && !mapping.Has(FieldFactorReference) {
		problems = append(problems, MappingError{
			Field:   FieldInvoiceNumber,
			Message: "Choose the column that holds the invoice number or the factor's reference",
		})
	}
	hasAmount := false
	for _, field := range amountFields() {
		if mapping.Has(field) {
			hasAmount = true
			break
		}
	}
	if !hasAmount {
		problems = append(problems, MappingError{
			Field:   FieldCash,
			Message: "Choose the column that holds the amount paid, the advance, the reserve or the invoice amount",
		})
	}

	seen := make(map[int]Field, len(mapping))
	for _, field := range Fields() {
		index, ok := mapping[field]
		if !ok {
			continue
		}
		if other, dup := seen[index]; dup {
			problems = append(problems, MappingError{
				Field: field,
				Message: field.Label() + " and " + strings.ToLower(
					other.Label(),
				) + " use the same column",
			})
			continue
		}
		seen[index] = field
	}
	return problems
}

var synonyms = map[Field][]string{
	FieldInvoiceNumber: {
		"invoice", "invoice number", "invoice no", "invoice num", "inv", "inv no",
		"inv number", "inv num", "invoice id", "client invoice", "client invoice number",
		"customer invoice", "load number", "load no", "load", "pro number", "pro",
	},
	FieldFactorReference: {
		"reference", "ref", "ref no", "reference number", "factor reference",
		"schedule", "schedule number", "schedule no", "batch", "batch number",
		"transaction id", "transaction number", "purchase id", "purchase number",
		"verification number", "item id",
	},
	FieldDebtor: {
		"debtor", "debtor name", "customer", "customer name", "broker", "broker name",
		"shipper", "account debtor", "payor", "payer",
	},
	FieldKind: {
		"type", "transaction type", "trans type", "kind", "activity", "activity type",
		"category", "description", "status",
	},
	FieldFace: {
		"invoice amount", "face", "face amount", "face value", "gross", "gross amount",
		"invoice total", "amount invoiced", "purchased amount", "original amount",
	},
	FieldAdvance: {
		"advance", "advance amount", "advanced", "amount advanced", "funded amount",
		"net funded", "net advance", "advance paid",
	},
	FieldFee: {
		"fee", "fees", "factoring fee", "discount", "discount fee", "factor fee",
		"fee amount", "service fee",
	},
	FieldReserve: {
		"reserve", "reserve amount", "reserve held", "escrow", "holdback", "rebate",
		"reserve balance",
	},
	FieldCash: {
		"amount", "amount paid", "paid", "net", "net amount", "payment", "payment amount",
		"net paid", "total paid", "disbursed", "disbursement", "remitted", "cash",
	},
	FieldDate: {
		"date", "transaction date", "purchase date", "funding date", "funded date",
		"payment date", "posting date", "release date", "remit date", "settlement date",
	},
}

func normalizeHeader(header string) string {
	var b strings.Builder
	b.Grow(len(header))
	lastSpace := true
	for _, r := range strings.ToLower(strings.TrimSpace(header)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func GuessMapping(headers []string) Mapping {
	normalized := make([]string, len(headers))
	for i, header := range headers {
		normalized[i] = normalizeHeader(header)
	}
	mapping := make(Mapping, len(Fields()))
	claimed := make(map[int]struct{}, len(headers))
	for _, field := range Fields() {
		for _, synonym := range synonyms[field] {
			key := normalizeHeader(synonym)
			for i, header := range normalized {
				if _, taken := claimed[i]; taken || header == "" || header != key {
					continue
				}
				mapping[field] = i
				claimed[i] = struct{}{}
				break
			}
			if mapping.Has(field) {
				break
			}
		}
	}
	return mapping
}

func UnmappedHeaders(headers []string, mapping Mapping) []string {
	claimed := make(map[int]struct{}, len(mapping))
	for _, index := range mapping {
		claimed[index] = struct{}{}
	}
	unmapped := make([]string, 0, len(headers))
	for i, header := range headers {
		if _, taken := claimed[i]; taken || strings.TrimSpace(header) == "" {
			continue
		}
		unmapped = append(unmapped, header)
	}
	return unmapped
}

func HeaderSignature(headers []string) string {
	parts := make([]string, 0, len(headers))
	for _, header := range headers {
		parts = append(parts, normalizeHeader(header))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}
