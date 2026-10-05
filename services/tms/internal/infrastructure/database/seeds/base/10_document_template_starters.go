package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// DocumentTemplateStartersSeed gives every organization an editable copy of each
// built-in.
//
// It seeds drafts and never sets is_org_default, which is what makes it safe to
// re-run: a draft renders nothing, so no run of this seed can change a document
// a customer receives. Without it an administrator opening the editor would face
// an empty box and have to author an invoice from scratch to discover what the
// built-in already does.
type DocumentTemplateStartersSeed struct {
	seedhelpers.BaseSeed
}

func NewDocumentTemplateStartersSeed() *DocumentTemplateStartersSeed {
	seed := &DocumentTemplateStartersSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentTemplateStarters",
		"1.0.0",
		"Seeds an editable draft of every built-in document and message template",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

// Repeatable is safe here precisely because a seeded template is a draft that is
// never the organization default: re-running adds kinds that did not exist yet
// and touches nothing that does.
func (s *DocumentTemplateStartersSeed) Repeatable() bool { return true }

func (s *DocumentTemplateStartersSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			orgs, err := listOrganizations(ctx, tx)
			if err != nil {
				return err
			}

			created, orgsTouched := 0, 0
			for i := range orgs {
				count, err := tenantbootstrap.CreateDocumentTemplateStarters(ctx, tx, tenantbootstrap.Scope{
					OrganizationID: orgs[i].ID,
					BusinessUnitID: orgs[i].BusinessUnitID,
					Record:         seedRecorder(sc, s.Name()),
				})
				if err != nil {
					return fmt.Errorf(
						"seed document template starters for org %s: %w", orgs[i].Name, err,
					)
				}
				if count == 0 {
					continue
				}
				created += count
				orgsTouched++
			}

			if created > 0 {
				seedhelpers.LogSuccess(
					"Created document template starters",
					fmt.Sprintf("- Seeded drafts for %d organizations", orgsTouched),
					fmt.Sprintf("- Created %d template drafts", created),
				)
			}

			return nil
		},
	)
}

func (s *DocumentTemplateStartersSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *DocumentTemplateStartersSeed) CanRollback() bool { return true }
