package tenantbootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templateregistry"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/shared/pulid"
)

func CreateDocumentTemplateStarters(ctx context.Context, db *keel.DB, scope Scope) (int, error) {
	registry := templateregistry.NewRegistry()
	existing, err := existingTemplateKinds(ctx, db, scope)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, kind := range documenttemplate.AllKinds() {
		if existing[kind] {
			continue
		}

		if err = createTemplateStarter(ctx, db, scope, registry, kind); err != nil {
			return 0, fmt.Errorf("seed %s: %w", kind, err)
		}
		created++
	}

	return created, nil
}

func createTemplateStarter(
	ctx context.Context,
	db *keel.DB,
	scope Scope,
	registry *templatecatalog.Registry,
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

	if err = documenttemplatedb.DocumentTemplates.Create(ctx, db, template); err != nil {
		return fmt.Errorf("insert template: %w", err)
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

	content := documenttemplateports.TemplateContent{
		Subject:    version.Subject,
		BodyHTML:   version.BodyHTML,
		BodyText:   version.BodyText,
		CSSContent: version.CSSContent,
	}
	version.ContentHash = content.Hash()

	starterHash, err := starters.Hash(kind)
	if err != nil {
		return err
	}
	version.StarterHash = starterHash

	if err = documenttemplatedb.DocumentTemplateVersions.Create(ctx, db, version); err != nil {
		return fmt.Errorf("insert version: %w", err)
	}

	return nil
}

func existingTemplateKinds(
	ctx context.Context,
	db *keel.DB,
	scope Scope,
) (map[documenttemplate.Kind]bool, error) {
	t := documenttemplatedb.DocumentTemplates
	rows, err := scoped(ctx, t.Pick(t.Kind).Where(
		t.OrganizationID.Eq(scope.OrganizationID),
		t.BusinessUnitID.Eq(scope.BusinessUnitID),
	)).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get existing template kinds: %w", err)
	}

	existing := make(map[documenttemplate.Kind]bool, len(rows))
	for _, row := range rows {
		existing[row.Kind] = true
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
