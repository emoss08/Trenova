package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// NormalAccountSeed Creates NormalAccount data
type NormalAccountSeed struct {
	seedhelpers.BaseSeed
}

// NewNormalAccountSeed creates a new NormalAccount seed
func NewNormalAccountSeed() *NormalAccountSeed {
	seed := &NormalAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"NormalAccount",
		"1.0.0",
		"Creates NormalAccount data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.BaseSeedIDs...)

	return seed
}

func (s *NormalAccountSeed) Run(ctx context.Context, tx bun.Tx) error {
	var orgs []tenant.Organization
	if err := tx.NewSelect().Model(&orgs).Order("created_at ASC").Scan(ctx); err != nil {
		return fmt.Errorf("get organizations: %w", err)
	}

	if len(orgs) == 0 {
		return fmt.Errorf("no organizations found")
	}

	primaryOrg := orgs[0]

	normalUser := &tenant.User{
		CurrentOrganizationID: primaryOrg.ID,
		BusinessUnitID:        primaryOrg.BusinessUnitID,
		Name:                  "Normal User",
		Username:              "normal",
		EmailAddress:          "normal@trenova.app",
		Status:                domaintypes.StatusActive,
		Timezone:              "America/Los_Angeles",
		MustChangePassword:    false, // TODO(wolfred): Change this to true in production
		CreatedAt:             timeutils.NowUnix(),
		UpdatedAt:             timeutils.NowUnix(),
	}

	hashedPassword, err := normalUser.GeneratePassword("normal123!")
	if err != nil {
		return fmt.Errorf("generate password hash: %w", err)
	}
	normalUser.Password = hashedPassword

	if _, err = tx.NewInsert().Model(normalUser).Exec(ctx); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	for i, org := range orgs {
		membership := &tenant.OrganizationMembership{
			BusinessUnitID: org.BusinessUnitID,
			UserID:         normalUser.ID,
			JoinedAt:       timeutils.NowUnix(),
			OrganizationID: org.ID,
			GrantedByID:    normalUser.ID,
			IsDefault:      i == 0,
		}
		if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
			return fmt.Errorf("create organization membership for org %s: %w", org.Name, err)
		}
	}
	return nil
}
