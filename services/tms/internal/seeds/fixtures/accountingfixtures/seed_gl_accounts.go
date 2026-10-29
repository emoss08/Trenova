package accountingfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type seedGLAccount struct {
	code        string
	name        string
	description string
	category    string
}

type seedGLAccountsRequest struct {
	orgID    pulid.ID
	buID     pulid.ID
	accounts []seedGLAccount
}

func ensureGLAccounts(
	ctx context.Context,
	db *keel.DB,
	req *seedGLAccountsRequest,
) (map[string]pulid.ID, int, error) {
	ga := glaccountdb.GLAccounts
	existing, err := ga.AllTenants().Pick(ga.ID, ga.AccountCode).Where(
		ga.OrganizationID.Eq(req.orgID),
		ga.BusinessUnitID.Eq(req.buID),
	).All(ctx, db)
	if err != nil {
		return nil, 0, fmt.Errorf("get gl accounts: %w", err)
	}

	accounts := make(map[string]pulid.ID, len(existing)+len(req.accounts))
	for i := range existing {
		accounts[existing[i].AccountCode] = existing[i].ID
	}

	at := accounttypedb.AccountTypes
	accountTypes, err := at.AllTenants().Pick(at.ID, at.Category).Where(
		at.OrganizationID.Eq(req.orgID),
		at.BusinessUnitID.Eq(req.buID),
	).All(ctx, db)
	if err != nil {
		return nil, 0, fmt.Errorf("get account types: %w", err)
	}
	typeByCategory := make(map[string]pulid.ID, len(accountTypes))
	for _, at := range accountTypes {
		typeByCategory[string(at.Category)] = at.ID
	}

	missing := make([]*glaccount.GLAccount, 0, len(req.accounts))
	for _, def := range req.accounts {
		if _, ok := accounts[def.code]; ok {
			continue
		}
		typeID, ok := typeByCategory[def.category]
		if !ok {
			return nil, 0, fmt.Errorf("account type %s not found", def.category)
		}

		account := &glaccount.GLAccount{
			ID:             pulid.MustNew("gla_"),
			BusinessUnitID: req.buID,
			OrganizationID: req.orgID,
			Status:         domaintypes.StatusActive,
			AccountTypeID:  typeID,
			AllowManualJE:  true,
			AccountCode:    def.code,
			Name:           def.name,
			Description:    def.description,
			IsSystem:       true,
		}
		accounts[def.code] = account.ID
		missing = append(missing, account)
	}
	if err = ga.CreateMany(ctx, db, missing); err != nil {
		return nil, 0, fmt.Errorf("insert gl accounts: %w", err)
	}

	return accounts, len(missing), nil
}
