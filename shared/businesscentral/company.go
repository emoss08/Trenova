package businesscentral

import (
	"strings"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/google/uuid"
)

const (
	guidLength           = 36
	compactGUIDLength    = 32
	refSeparator         = '_'
	maxEnvironmentLength = 29
)

type CompanyRef struct {
	TenantID    string
	Environment string
	CompanyID   string
}

func NewCompanyRef(tenantID, environment, companyID string) (CompanyRef, error) {
	tenant, err := GUID(tenantID)
	if err != nil {
		return CompanyRef{}, businesscentralerrors.ErrInvalidCompanyRef
	}
	company, err := GUID(companyID)
	if err != nil {
		return CompanyRef{}, businesscentralerrors.ErrInvalidCompanyRef
	}
	env := strings.TrimSpace(environment)
	if !ValidEnvironmentName(env) {
		return CompanyRef{}, businesscentralerrors.ErrInvalidEnvironment
	}
	return CompanyRef{TenantID: tenant, Environment: env, CompanyID: company}, nil
}

func ParseCompanyRef(raw string) (CompanyRef, error) {
	value := strings.TrimSpace(raw)
	envStart := 2*compactGUIDLength + 2
	if len(value) <= envStart ||
		value[compactGUIDLength] != refSeparator ||
		value[2*compactGUIDLength+1] != refSeparator {
		return CompanyRef{}, businesscentralerrors.ErrInvalidCompanyRef
	}
	tenant := value[:compactGUIDLength]
	company := value[compactGUIDLength+1 : 2*compactGUIDLength+1]
	if !isHex(tenant) || !isHex(company) {
		return CompanyRef{}, businesscentralerrors.ErrInvalidCompanyRef
	}
	return NewCompanyRef(tenant, value[envStart:], company)
}

func (r CompanyRef) Validate() error {
	_, err := NewCompanyRef(r.TenantID, r.Environment, r.CompanyID)
	return err
}

func (r CompanyRef) String() string {
	normalized, err := NewCompanyRef(r.TenantID, r.Environment, r.CompanyID)
	if err != nil {
		return ""
	}
	return compactGUID(normalized.TenantID) + string(refSeparator) +
		compactGUID(normalized.CompanyID) + string(refSeparator) +
		normalized.Environment
}

func ValidEnvironmentName(name string) bool {
	if name == "" || len(name) > maxEnvironmentLength || !IsLetter(name[0]) {
		return false
	}
	for idx := range len(name) {
		ch := name[idx]
		if !IsLetter(ch) && !IsDigit(ch) && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func GUID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", businesscentralerrors.ErrIDRequired
	}
	switch len(value) {
	case guidLength:
		if !isHex(strings.ReplaceAll(value, "-", "")) {
			return "", businesscentralerrors.ErrInvalidID
		}
	case compactGUIDLength:
		if !isHex(value) {
			return "", businesscentralerrors.ErrInvalidID
		}
	default:
		return "", businesscentralerrors.ErrInvalidID
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", businesscentralerrors.ErrInvalidID
	}
	return parsed.String(), nil
}

func GUIDs(ids []string, limit int) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id, err := GUID(raw)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > limit {
		return nil, businesscentralerrors.ErrTooManyIDs
	}
	return out, nil
}

func compactGUID(value string) string {
	return strings.ReplaceAll(value, "-", "")
}

func isHex(value string) bool {
	for idx := range len(value) {
		ch := value[idx]
		if !IsDigit(ch) && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return false
		}
	}
	return value != ""
}

func IsLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func IsDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
