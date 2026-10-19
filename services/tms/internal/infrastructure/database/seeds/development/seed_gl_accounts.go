package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type seedGLAccount struct {
	code        string
	name        string
	description string
	category    string
}

type seedGLAccountsRequest struct {
	seedName string
	orgID    pulid.ID
	buID     pulid.ID
	accounts []seedGLAccount
}

func ensureGLAccounts(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	req *seedGLAccountsRequest,
) (map[string]pulid.ID, int, error) {
	var existing []glaccount.GLAccount
	if err := tx.NewSelect().Model(&existing).Column("id", "account_code").
		Where("organization_id = ?", req.orgID).
		Where("business_unit_id = ?", req.buID).
		Scan(ctx); err != nil {
		return nil, 0, fmt.Errorf("get gl accounts: %w", err)
	}

	accounts := make(map[string]pulid.ID, len(existing)+len(req.accounts))
	for i := range existing {
		accounts[existing[i].AccountCode] = existing[i].ID
	}

	var accountTypes []struct {
		ID       pulid.ID `bun:"id"`
		Category string   `bun:"category"`
	}
	if err := tx.NewSelect().Table("account_types").Column("id", "category").
		Where("organization_id = ?", req.orgID).
		Where("business_unit_id = ?", req.buID).
		Scan(ctx, &accountTypes); err != nil {
		return nil, 0, fmt.Errorf("get account types: %w", err)
	}
	typeByCategory := make(map[string]pulid.ID, len(accountTypes))
	for _, at := range accountTypes {
		typeByCategory[at.Category] = at.ID
	}

	created := 0
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
		if _, err := tx.NewInsert().Model(account).Exec(ctx); err != nil {
			return nil, 0, fmt.Errorf("insert gl account %s: %w", def.code, err)
		}
		if err := sc.TrackCreated(ctx, "gl_accounts", account.ID, req.seedName); err != nil {
			return nil, 0, fmt.Errorf("track gl account: %w", err)
		}
		accounts[def.code] = account.ID
		created++
	}

	return accounts, created, nil
}
