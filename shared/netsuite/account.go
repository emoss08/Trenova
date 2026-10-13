package netsuite

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	maxAccountIDLength  = 64
	maxSubsidiaryDigits = 18
	subsidiaryMarker    = "-s"
	sandboxMarker       = "_SB"
)

var (
	accountIDPattern   = regexp.MustCompile(`^[A-Z0-9]+(_(SB\d+|RP))?$`)
	accountHostPattern = regexp.MustCompile(`^[a-z0-9]+(-(sb\d+|rp))?$`)
)

type CompanyRef struct {
	AccountID    string
	SubsidiaryID string
}

func NormalizeAccountID(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" || len(value) > maxAccountIDLength || !accountIDPattern.MatchString(value) {
		return "", ErrInvalidAccountID
	}
	return value, nil
}

func ValidAccountID(raw string) bool {
	_, err := NormalizeAccountID(raw)
	return err == nil
}

func IsSandboxAccount(accountID string) bool {
	id, err := NormalizeAccountID(accountID)
	return err == nil && strings.Contains(id, sandboxMarker)
}

func AccountHost(accountID string) (string, error) {
	id, err := NormalizeAccountID(accountID)
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.ReplaceAll(id, "_", "-")), nil
}

func APIHost(accountID string) (string, error) {
	host, err := AccountHost(accountID)
	if err != nil {
		return "", err
	}
	return host + apiHostSuffix, nil
}

func AppHost(accountID string) (string, error) {
	host, err := AccountHost(accountID)
	if err != nil {
		return "", err
	}
	return host + appHostSuffix, nil
}

func NewCompanyRef(accountID, subsidiaryID string) (CompanyRef, error) {
	account, err := NormalizeAccountID(accountID)
	if err != nil {
		return CompanyRef{}, err
	}
	subsidiary := strings.TrimSpace(subsidiaryID)
	if subsidiary != "" && !validSubsidiaryID(subsidiary) {
		return CompanyRef{}, ErrInvalidSubsidiaryID
	}
	return CompanyRef{AccountID: account, SubsidiaryID: subsidiary}, nil
}

func ParseCompanyRef(raw string) (CompanyRef, error) {
	value := strings.TrimSpace(raw)
	account, subsidiary, found := strings.Cut(value, "-")
	if found {
		rest, ok := strings.CutPrefix(subsidiary, subsidiaryMarker[1:])
		if !ok || rest == "" {
			return CompanyRef{}, ErrInvalidCompanyRef
		}
		subsidiary = rest
	}
	ref, err := NewCompanyRef(account, subsidiary)
	if err != nil {
		return CompanyRef{}, fmt.Errorf("%w: %w", ErrInvalidCompanyRef, err)
	}
	return ref, nil
}

func (r CompanyRef) Validate() error {
	_, err := NewCompanyRef(r.AccountID, r.SubsidiaryID)
	return err
}

func (r CompanyRef) String() string {
	normalized, err := NewCompanyRef(r.AccountID, r.SubsidiaryID)
	if err != nil {
		return ""
	}
	if normalized.SubsidiaryID == "" {
		return normalized.AccountID
	}
	return normalized.AccountID + subsidiaryMarker + normalized.SubsidiaryID
}

func (r CompanyRef) Sandbox() bool {
	return IsSandboxAccount(r.AccountID)
}

func (r CompanyRef) APIHost() string {
	host, err := APIHost(r.AccountID)
	if err != nil {
		return ""
	}
	return host
}

func (r CompanyRef) AppHost() string {
	host, err := AppHost(r.AccountID)
	if err != nil {
		return ""
	}
	return host
}

func validSubsidiaryID(value string) bool {
	if value == "" || len(value) > maxSubsidiaryDigits || value[0] == '0' {
		return false
	}
	for idx := range len(value) {
		if value[idx] < '0' || value[idx] > '9' {
			return false
		}
	}
	return true
}
