package netsuite

import (
	"net/url"
	"strings"
)

const (
	SystemHost      = "system.netsuite.com"
	apiHostSuffix   = ".suitetalk.api.netsuite.com"
	appHostSuffix   = ".app.netsuite.com"
	loginBaseURL    = "https://" + SystemHost
	authorizePath   = "/app/login/oauth2/authorize.nl"
	tokenPath       = "/services/rest/auth/oauth2/v1/token"
	revokePath      = "/services/rest/auth/oauth2/v1/revoke"
	recordPath      = "/services/rest/record/v1"
	suiteQLPath     = "/services/rest/query/v1/suiteql"
	transactionPath = "/app/accounting/transactions/transaction.nl"
)

func AllowedHosts(accountID string) ([]string, error) {
	apiHost, err := APIHost(accountID)
	if err != nil {
		return nil, err
	}
	appHost, err := AppHost(accountID)
	if err != nil {
		return nil, err
	}
	return []string{apiHost, appHost, SystemHost}, nil
}

func IsAllowedHost(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == SystemHost {
		return true
	}
	for _, suffix := range [...]string{apiHostSuffix, appHostSuffix} {
		if label, ok := strings.CutSuffix(value, suffix); ok {
			return len(label) <= maxAccountIDLength && accountHostPattern.MatchString(label)
		}
	}
	return false
}

func TransactionLink(accountID, internalID string) string {
	host, err := AppHost(accountID)
	if err != nil || !isInternalID(internalID) {
		return ""
	}
	return "https://" + host + transactionPath + "?id=" + url.QueryEscape(internalID)
}
