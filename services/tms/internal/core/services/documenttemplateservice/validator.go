package documenttemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// fieldKind is the json path the kind errors are reported against and the
// column they are checked on; goconst rightly objects to repeating it.
const fieldKind = "kind"

type ValidatorParams struct {
	fx.In

	DB       *postgres.Connection `optional:"true"`
	Registry *documenttemplate.Registry
}

type Validator struct {
	registry   *documenttemplate.Registry
	template   *validationframework.TenantedValidator[*documenttemplate.DocumentTemplate]
	version    *validationframework.TenantedValidator[*documenttemplate.DocumentTemplateVersion]
	assignment *validationframework.TenantedValidator[*documenttemplate.DocumentTemplateAssignment]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		registry:   p.Registry,
		template:   buildTemplateValidator(p),
		version:    buildVersionValidator(p),
		assignment: buildAssignmentValidator(p),
	}
}

// withCheckers attaches the database-backed checkers when a connection is
// available. Without one the builder still records the field configuration and
// the framework simply skips those stages, which is what lets the domain rules
// be exercised without a database.
func withCheckers[T validationframework.TenantedEntity](
	builder *validationframework.TenantedValidatorBuilder[T],
	db *postgres.Connection,
) *validationframework.TenantedValidatorBuilder[T] {
	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return db.DB() }),
		).
		WithReferenceChecker(
			validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return db.DB() }),
		)
}

func buildTemplateValidator(
	p ValidatorParams,
) *validationframework.TenantedValidator[*documenttemplate.DocumentTemplate] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*documenttemplate.DocumentTemplate]().
		WithModelName("DocumentTemplate").
		WithUniqueField(
			"code",
			"code",
			"A document template with this code already exists in your organization",
			func(t *documenttemplate.DocumentTemplate) any { return t.Code },
		).
		// A template's kind is what its versions are written against. Changing it
		// would leave every version authored for a different variable catalog.
		WithImmutableField(
			fieldKind,
			"A template's kind cannot change. Its versions are written against the "+
				"variables of the original kind and would stop rendering.",
			func(t *documenttemplate.DocumentTemplate) any { return t.Kind },
		).
		WithCustomRule(newKindRegisteredRule(p.Registry))

	if p.DB != nil {
		builder = builder.WithCustomRule(newSingleOrgDefaultRule(p.DB))
	}

	return withCheckers(builder, p.DB).Build()
}

func buildVersionValidator(
	p ValidatorParams,
) *validationframework.TenantedValidator[*documenttemplate.DocumentTemplateVersion] {
	return withCheckers(
		validationframework.
			NewTenantedValidatorBuilder[*documenttemplate.DocumentTemplateVersion]().
			WithModelName("DocumentTemplateVersion").
			WithReferenceCheck(
				"templateId",
				"document_templates",
				"That template does not exist in your organization",
				func(v *documenttemplate.DocumentTemplateVersion) pulid.ID { return v.TemplateID },
			).
			WithOptionalReferenceCheck(
				"sourceVersionId",
				"document_template_versions",
				"The version this was copied from does not exist in your organization",
				func(v *documenttemplate.DocumentTemplateVersion) pulid.ID {
					if v.SourceVersionID == nil {
						return pulid.Nil
					}
					return *v.SourceVersionID
				},
			),
		p.DB,
	).Build()
}

func buildAssignmentValidator(
	p ValidatorParams,
) *validationframework.TenantedValidator[*documenttemplate.DocumentTemplateAssignment] {
	return withCheckers(
		validationframework.
			NewTenantedValidatorBuilder[*documenttemplate.DocumentTemplateAssignment]().
			WithModelName("DocumentTemplateAssignment").
			WithCompositeUniqueFields(
				"customerKind",
				"That customer already has a template assigned for this kind",
				validationframework.CompositeField[*documenttemplate.DocumentTemplateAssignment]{
					FieldName: "customerId",
					Column:    "customer_id",
					GetValue: func(a *documenttemplate.DocumentTemplateAssignment) any {
						return a.CustomerID
					},
					CaseSensitive: true,
				},
				validationframework.CompositeField[*documenttemplate.DocumentTemplateAssignment]{
					FieldName: fieldKind,
					Column:    fieldKind,
					GetValue: func(a *documenttemplate.DocumentTemplateAssignment) any {
						return a.Kind
					},
					CaseSensitive: true,
				},
			).
			WithReferenceCheck(
				"templateId",
				"document_templates",
				"That template does not exist in your organization",
				func(a *documenttemplate.DocumentTemplateAssignment) pulid.ID { return a.TemplateID },
			).
			WithReferenceCheck(
				"customerId",
				"customers",
				"That customer does not exist in your organization",
				func(a *documenttemplate.DocumentTemplateAssignment) pulid.ID { return a.CustomerID },
			),
		p.DB,
	).Build()
}

// newKindRegisteredRule refuses a kind the registry does not publish.
//
// The check lives here rather than on the entity because the domain cannot
// import the registry without a cycle, and a template naming an unregistered
// kind has no variable catalog, no sample context, and no starter — it would be
// unrenderable from the moment it was saved.
func newKindRegisteredRule(
	registry *documenttemplate.Registry,
) validationframework.TenantedRule[*documenttemplate.DocumentTemplate] {
	return validationframework.
		NewTenantedRule[*documenttemplate.DocumentTemplate]("kind_registered").
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithParallelSafeExecution().
		WithValidation(func(
			_ context.Context,
			entity *documenttemplate.DocumentTemplate,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Kind == "" {
				return nil
			}

			if _, ok := registry.Get(entity.Kind); !ok {
				multiErr.Add(fieldKind, errortypes.ErrInvalid, fmt.Sprintf(
					"%q is not a template kind this system can render", entity.Kind,
				))
			}

			return nil
		})
}

// newSingleOrgDefaultRule mirrors the partial unique index that allows one
// default per kind, so a second default is a field error on the checkbox that
// caused it rather than a constraint violation.
func newSingleOrgDefaultRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*documenttemplate.DocumentTemplate] {
	return validationframework.
		NewTenantedRule[*documenttemplate.DocumentTemplate]("single_org_default").
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *documenttemplate.DocumentTemplate,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.IsOrgDefault {
				return nil
			}

			query := db.DB().NewSelect().
				Table("document_templates").
				ColumnExpr("1").
				Where("organization_id = ?", valCtx.OrganizationID).
				Where("business_unit_id = ?", valCtx.BusinessUnitID).
				Where("kind = ?", entity.Kind).
				Where("is_org_default = TRUE")

			if valCtx.EntityID.IsNotNil() {
				query = query.Where("id != ?", valCtx.EntityID)
			}

			exists, err := query.Exists(ctx)
			if err != nil {
				return err
			}

			if exists {
				multiErr.Add("isOrgDefault", errortypes.ErrInvalid,
					"Another template is already the organization default for this kind. "+
						"Clear that one first.")
			}

			return nil
		})
}

// ValidateTemplate checks a template's identity, that the kind it names can
// actually be rendered, and that it does not collide with what is already stored.
func (v *Validator) ValidateTemplate(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
	original *documenttemplate.DocumentTemplate,
) *errortypes.MultiError {
	if original == nil {
		return v.template.ValidateCreate(ctx, entity)
	}

	return v.template.ValidateUpdateWithOriginal(ctx, entity, original)
}

// ValidateVersion checks the shape of a version's content.
//
// It deliberately does not compile the template: a draft is allowed to be wrong
// while it is being written, and refusing to save half-finished markup would
// make the editor unusable. Compilation is the publish gate's job.
func (v *Validator) ValidateVersion(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
	kind documenttemplate.Kind,
) *errortypes.MultiError {
	var multiErr *errortypes.MultiError
	if entity.GetID().IsNil() {
		multiErr = v.version.ValidateCreate(ctx, entity)
	} else {
		multiErr = v.version.ValidateUpdate(ctx, entity)
	}

	def, ok := v.registry.Get(kind)
	if !ok {
		return multiErr
	}

	channelErr := errortypes.NewMultiError()
	v.checkChannelsBelong(entity, def, channelErr)

	// Page setup on a message is not harmless noise: it would show a page-size
	// control in the editor for something that is never printed.
	if !def.Paged && (entity.PageSize != "" || entity.Orientation != "") {
		channelErr.Add("pageSize", errortypes.ErrInvalid, fmt.Sprintf(
			"%s is a message, not a document, so it has no page setup",
			def.DisplayName,
		))
	}

	return mergeErrors(multiErr, channelErr)
}

// checkChannelsBelong refuses content in a slot the kind does not produce.
//
// Without this an author can type a plain-text body for a PDF-only kind, see it
// saved, and never understand why it is not in the output.
func (v *Validator) checkChannelsBelong(
	entity *documenttemplate.DocumentTemplateVersion,
	def *documenttemplate.KindDefinition,
	multiErr *errortypes.MultiError,
) {
	if entity.Subject != "" &&
		!def.HasChannel(documenttemplate.ChannelSubject) &&
		!def.HasChannel(documenttemplate.ChannelNotificationTitle) {
		multiErr.Add("subject", errortypes.ErrInvalid, fmt.Sprintf(
			"%s does not have a subject line", def.DisplayName,
		))
	}

	if entity.BodyText != "" &&
		!def.HasChannel(documenttemplate.ChannelEmailText) &&
		!def.HasChannel(documenttemplate.ChannelNotificationBody) {
		multiErr.Add("bodyText", errortypes.ErrInvalid, fmt.Sprintf(
			"%s does not have a plain-text body", def.DisplayName,
		))
	}

	if entity.CSSContent != "" && !def.HasChannel(documenttemplate.ChannelPDF) {
		multiErr.Add("cssContent", errortypes.ErrInvalid, fmt.Sprintf(
			"%s is delivered by email, where clients strip stylesheets. Use inline "+
				"styles on the elements instead.", def.DisplayName,
		))
	}

	if (entity.HeaderHTML != "" || entity.FooterHTML != "") &&
		!def.HasChannel(documenttemplate.ChannelPDF) {
		multiErr.Add("headerHtml", errortypes.ErrInvalid, fmt.Sprintf(
			"%s is not printed, so it has no running header or footer",
			def.DisplayName,
		))
	}
}

// ValidateAssignment checks that an override can actually take effect.
func (v *Validator) ValidateAssignment(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateAssignment,
	template *documenttemplate.DocumentTemplate,
) *errortypes.MultiError {
	var multiErr *errortypes.MultiError
	if entity.GetID().IsNil() {
		multiErr = v.assignment.ValidateCreate(ctx, entity)
	} else {
		multiErr = v.assignment.ValidateUpdate(ctx, entity)
	}

	if template == nil {
		return multiErr
	}

	scopeErr := errortypes.NewMultiError()

	def, ok := v.registry.Get(template.Kind)
	switch {
	case !ok:
		scopeErr.Add("templateId", errortypes.ErrInvalid, fmt.Sprintf(
			"%q is not a template kind this system can render", template.Kind,
		))
	case !def.CustomerScoped:
		// Assigning a kind with no customer in scope would look like it worked
		// and then never fire on any render.
		scopeErr.Add(fieldKind, errortypes.ErrInvalid, fmt.Sprintf(
			"%s is not sent per customer, so assigning it to one has no effect",
			def.DisplayName,
		))
	}

	if !template.HasActiveVersion() {
		scopeErr.Add("templateId", errortypes.ErrInvalid,
			"That template has no published version, so the customer would keep "+
				"receiving the current one. Publish it first.")
	}

	return mergeErrors(multiErr, scopeErr)
}

// mergeErrors folds the rules that need context the framework does not carry
// into the framework's own verdict, so a caller sees one set of field errors.
func mergeErrors(base, extra *errortypes.MultiError) *errortypes.MultiError {
	if !extra.HasErrors() {
		return base
	}

	if base == nil {
		return extra
	}

	for _, err := range extra.Errors {
		base.AddError(err)
	}

	return base
}
