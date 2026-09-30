package xeroconnector

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/xero"
)

const (
	accountCodesTTL     = 10 * time.Minute
	accountCodesRefetch = 30 * time.Second
	maxAccountTenants   = 1024
)

type accountIndex struct {
	codeByID map[string]string
	idByCode map[string]string
	loadedAt time.Time
}

type accountCodes struct {
	now      func() time.Time
	mu       sync.Mutex
	byTenant map[string]*accountIndex
}

func newAccountCodes(now func() time.Time) *accountCodes {
	return &accountCodes{now: now, byTenant: make(map[string]*accountIndex)}
}

func (a *accountCodes) store(tenant string, accounts []xero.Account) *accountIndex {
	index := &accountIndex{
		codeByID: make(map[string]string, len(accounts)),
		idByCode: make(map[string]string, len(accounts)),
		loadedAt: a.now(),
	}
	for idx := range accounts {
		account := &accounts[idx]
		id := strings.ToLower(strings.TrimSpace(account.AccountID))
		code := strings.TrimSpace(account.Code)
		if id == "" {
			continue
		}
		index.codeByID[id] = code
		if code != "" {
			index.idByCode[strings.ToUpper(code)] = id
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if _, known := a.byTenant[tenant]; !known && len(a.byTenant) >= maxAccountTenants {
		clear(a.byTenant)
	}
	a.byTenant[tenant] = index
	return index
}

func (a *accountCodes) cached(tenant string) (*accountIndex, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	index, ok := a.byTenant[tenant]
	if !ok || a.now().Sub(index.loadedAt) > accountCodesTTL {
		return index, false
	}
	return index, true
}

func (a *accountCodes) lookup(
	ctx context.Context,
	client *xero.Client,
	find func(*accountIndex) (string, bool),
) (value string, found bool, err error) {
	tenant := client.TenantID()
	index, fresh := a.cached(tenant)
	if fresh {
		if cachedValue, ok := find(index); ok {
			return cachedValue, true, nil
		}
		if a.now().Sub(index.loadedAt) < accountCodesRefetch {
			return "", false, nil
		}
	}

	accounts, err := client.Accounts(ctx, nil)
	if err != nil {
		return "", false, err
	}
	value, found = find(a.store(tenant, accounts))
	return value, found, nil
}

func (a *accountCodes) codeFor(
	ctx context.Context,
	client *xero.Client,
	accountID string,
) (code string, found bool, err error) {
	id := strings.ToLower(strings.TrimSpace(accountID))
	return a.lookup(ctx, client, func(index *accountIndex) (string, bool) {
		accountCode, ok := index.codeByID[id]
		return accountCode, ok && accountCode != ""
	})
}

func (a *accountCodes) idFor(
	ctx context.Context,
	client *xero.Client,
	code string,
) (id string, found bool, err error) {
	key := strings.ToUpper(strings.TrimSpace(code))
	if key == "" {
		return "", false, nil
	}
	return a.lookup(ctx, client, func(index *accountIndex) (string, bool) {
		accountID, ok := index.idByCode[key]
		return accountID, ok
	})
}

func (c *Connector) accountCode(
	ctx context.Context,
	client *xero.Client,
	accountID, purpose string,
) (string, error) {
	if strings.TrimSpace(accountID) == "" {
		return "", unmappedAccount(purpose)
	}
	code, ok, err := c.codes.codeFor(ctx, client, accountID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", &accountingsync.SyncError{
			Category: accountingsync.SyncErrorMapping,
			Code:     "account-code",
			Message: purpose + " is mapped to a " + providerName +
				" account that no longer exists or has no code",
			Resolution: "Refresh the reference data, give the account a code in " + providerName +
				" or map a different account",
		}
	}
	return code, nil
}

func unmappedAccount(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "account",
		Message:    purpose + " has no " + providerName + " account mapped",
		Resolution: "Map it to a " + providerName + " account",
	}
}
