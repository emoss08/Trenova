package nsbinding

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/netsuite"
)

var (
	ErrNoAccount = errors.New("netsuite: the NetSuite account id is required")
)

func AccountOf(realmID string) (string, error) {
	if strings.TrimSpace(realmID) == "" {
		return "", ErrNoAccount
	}
	ref, err := netsuite.ParseCompanyRef(realmID)
	if err != nil {
		return "", err
	}
	return ref.AccountID, nil
}
