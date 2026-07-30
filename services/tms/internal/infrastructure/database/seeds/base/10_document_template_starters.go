package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documenttemplateservice"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
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
	registry *documenttemplate.Registry
}

func NewDocumentTemplateStartersSeed() *DocumentTemplateStartersSeed {
	seed := &DocumentTemplateStartersSeed{
		registry: documenttemplate.NewRegistry(),
	}
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
			var orgs []tenant.Organization
			if err := tx.NewSelect().Model(&orgs).Order("created_at ASC").Scan(ctx); err != nil {
				return fmt.Errorf("get organizations: %w", err)
			}

			if len(orgs) == 0 {
				return fmt.Errorf("no organizations found")
			}

			created, orgsTouched := 0, 0
			for i := range orgs {
				count, err := s.seedOrganization(ctx, tx, sc, &orgs[i])
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

func (s *DocumentTemplateStartersSeed) seedOrganization(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
) (int, error) {
	existing, err := s.existingKinds(ctx, tx, org)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, kind := range documenttemplate.AllKinds() {
		if existing[kind] {
			// The organization already has a template of this kind. Adding a
			// second would give the editor two entries for one slot.
			continue
		}

		if err = s.seedKind(ctx, tx, sc, org, kind); err != nil {
			return 0, fmt.Errorf("seed %s: %w", kind, err)
		}
		created++
	}

	return created, nil
}

func (s *DocumentTemplateStartersSeed) seedKind(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
	kind documenttemplate.Kind,
) error {
	def, ok := s.registry.Get(kind)
	if !ok {
		return fmt.Errorf("kind %s is not registered", kind)
	}

	starter, err := starters.For(kind)
	if err != nil {
		return err
	}

	template := &documenttemplate.DocumentTemplate{
		ID:             pulid.MustNew("dtpl_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Kind:           kind,
		Code:           templateCodeFor(kind),
		Name:           def.DisplayName,
		Description:    def.Description,
		// Never the default and never live: publishing is the administrator's
		// decision, and until they make it the built-in renders unchanged.
		IsOrgDefault:    false,
		ActiveVersionID: nil,
	}

	if _, err = tx.NewInsert().Model(template).Exec(ctx); err != nil {
		return fmt.Errorf("insert template: %w", err)
	}

	if err = sc.TrackCreated(ctx, "document_templates", template.ID, s.Name()); err != nil {
		return fmt.Errorf("track template: %w", err)
	}

	version := &documenttemplate.DocumentTemplateVersion{
		ID:             pulid.MustNew("dtv_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		TemplateID:     template.ID,
		VersionNumber:  1,
		Status:         documenttemplate.VersionStatusDraft,
		Subject:        starter.Subject,
		BodyHTML:       starter.BodyHTML,
		BodyText:       starter.BodyText,
		CSSContent:     starter.CSS,
	}

	if def.Paged {
		version.PageSize = starter.PageSize
		version.Orientation = starter.Orientation
		version.MarginTop = &starter.Margins.Top
		version.MarginBottom = &starter.Margins.Bottom
		version.MarginLeft = &starter.Margins.Left
		version.MarginRight = &starter.Margins.Right
	}

	content := services.TemplateContent{
		Subject:    version.Subject,
		BodyHTML:   version.BodyHTML,
		BodyText:   version.BodyText,
		CSSContent: version.CSSContent,
	}
	version.ContentHash = documenttemplateservice.ContentHash(&content)

	// The starter's own hash is recorded separately so a later deploy can tell
	// the administrator the built-in has moved on since they copied it.
	starterHash, err := starters.Hash(kind)
	if err != nil {
		return err
	}
	version.StarterHash = starterHash

	if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
		return fmt.Errorf("insert version: %w", err)
	}

	return sc.TrackCreated(ctx, "document_template_versions", version.ID, s.Name())
}

func (s *DocumentTemplateStartersSeed) existingKinds(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (map[documenttemplate.Kind]bool, error) {
	rows := make([]string, 0, len(documenttemplate.AllKinds()))
	if err := tx.NewSelect().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Column("kind").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("get existing template kinds: %w", err)
	}

	existing := make(map[documenttemplate.Kind]bool, len(rows))
	for _, row := range rows {
		existing[documenttemplate.Kind(row)] = true
	}

	return existing, nil
}

// templateCodeFor turns a kind into the human-facing code the code column holds.
//
// A kind is dotted and lowercase; a code is the uppercase identifier an
// administrator sees and searches by, and it is unique per organization.
func templateCodeFor(kind documenttemplate.Kind) string {
	code := strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(string(kind)))
	if len(code) > documenttemplate.MaxCodeLength {
		code = code[:documenttemplate.MaxCodeLength]
	}

	return code
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
