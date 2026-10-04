package tenantbootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documenttemplateservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func CreateDocumentTemplateStarters(ctx context.Context, tx bun.IDB, scope Scope) (int, error) {
	registry := documenttemplate.NewRegistry()
	existing, err := existingTemplateKinds(ctx, tx, scope)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, kind := range documenttemplate.AllKinds() {
		if existing[kind] {
			continue
		}

		if err = createTemplateStarter(ctx, tx, scope, registry, kind); err != nil {
			return 0, fmt.Errorf("seed %s: %w", kind, err)
		}
		created++
	}

	return created, nil
}

func createTemplateStarter(
	ctx context.Context,
	tx bun.IDB,
	scope Scope,
	registry *documenttemplate.Registry,
	kind documenttemplate.Kind,
) error {
	def, ok := registry.Get(kind)
	if !ok {
		return fmt.Errorf("kind %s is not registered", kind)
	}

	starter, err := starters.For(kind)
	if err != nil {
		return err
	}

	template := &documenttemplate.DocumentTemplate{
		ID:              pulid.MustNew("dtpl_"),
		OrganizationID:  scope.OrganizationID,
		BusinessUnitID:  scope.BusinessUnitID,
		Kind:            kind,
		Code:            templateCodeFor(kind),
		Name:            def.DisplayName,
		Description:     def.Description,
		IsOrgDefault:    false,
		ActiveVersionID: nil,
	}

	if _, err = tx.NewInsert().Model(template).Exec(ctx); err != nil {
		return fmt.Errorf("insert template: %w", err)
	}

	if err = scope.record(ctx, "document_templates", template.ID); err != nil {
		return fmt.Errorf("track template: %w", err)
	}

	version := &documenttemplate.DocumentTemplateVersion{
		ID:             pulid.MustNew("dtv_"),
		OrganizationID: scope.OrganizationID,
		BusinessUnitID: scope.BusinessUnitID,
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

	starterHash, err := starters.Hash(kind)
	if err != nil {
		return err
	}
	version.StarterHash = starterHash

	if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
		return fmt.Errorf("insert version: %w", err)
	}

	return scope.record(ctx, "document_template_versions", version.ID)
}

func existingTemplateKinds(
	ctx context.Context,
	tx bun.IDB,
	scope Scope,
) (map[documenttemplate.Kind]bool, error) {
	rows := make([]string, 0, len(documenttemplate.AllKinds()))
	if err := tx.NewSelect().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Column("kind").
		Where("organization_id = ?", scope.OrganizationID).
		Where("business_unit_id = ?", scope.BusinessUnitID).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("get existing template kinds: %w", err)
	}

	existing := make(map[documenttemplate.Kind]bool, len(rows))
	for _, row := range rows {
		existing[documenttemplate.Kind(row)] = true
	}

	return existing, nil
}

func templateCodeFor(kind documenttemplate.Kind) string {
	code := strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(string(kind)))
	if len(code) > documenttemplate.MaxCodeLength {
		code = code[:documenttemplate.MaxCodeLength]
	}

	return code
}
