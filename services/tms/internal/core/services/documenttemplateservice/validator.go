package documenttemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Registry *documenttemplate.Registry
}

type Validator struct {
	registry *documenttemplate.Registry
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{registry: p.Registry}
}

// ValidateTemplate checks a template's identity and that the kind it names can
// actually be rendered.
//
// The kind check lives here rather than on the entity because the domain cannot
// import the registry without a cycle, and a template naming an unregistered
// kind has no variable catalog, no sample context, and no starter — it would be
// unrenderable from the moment it was saved.
func (v *Validator) ValidateTemplate(
	_ context.Context,
	entity *documenttemplate.DocumentTemplate,
	original *documenttemplate.DocumentTemplate,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Kind != "" {
		if _, ok := v.registry.Get(entity.Kind); !ok {
			multiErr.Add("kind", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a template kind this system can render", entity.Kind,
			))
		}
	}

	if original != nil && entity.Kind != original.Kind {
		multiErr.Add("kind", errortypes.ErrInvalid,
			"A template's kind cannot change. Its versions are written against the "+
				"variables of the original kind and would stop rendering.")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// ValidateVersion checks the shape of a version's content.
//
// It deliberately does not compile the template: a draft is allowed to be wrong
// while it is being written, and refusing to save half-finished markup would
// make the editor unusable. Compilation is the publish gate's job.
func (v *Validator) ValidateVersion(
	_ context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
	kind documenttemplate.Kind,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	def, ok := v.registry.Get(kind)
	if !ok {
		if multiErr.HasErrors() {
			return multiErr
		}
		return nil
	}

	v.checkChannelsBelong(entity, def, multiErr)

	// Page setup on a message is not harmless noise: it would show a page-size
	// control in the editor for something that is never printed.
	if !def.Paged {
		if entity.PageSize != "" || entity.Orientation != "" {
			multiErr.Add("pageSize", errortypes.ErrInvalid, fmt.Sprintf(
				"%s is a message, not a document, so it has no page setup",
				def.DisplayName,
			))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
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
	_ context.Context,
	entity *documenttemplate.DocumentTemplateAssignment,
	template *documenttemplate.DocumentTemplate,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if template != nil {
		def, ok := v.registry.Get(template.Kind)
		switch {
		case !ok:
			multiErr.Add("templateId", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a template kind this system can render", template.Kind,
			))
		case !def.CustomerScoped:
			// Assigning a kind with no customer in scope would look like it
			// worked and then never fire on any render.
			multiErr.Add("kind", errortypes.ErrInvalid, fmt.Sprintf(
				"%s is not sent per customer, so assigning it to one has no effect",
				def.DisplayName,
			))
		}

		if !template.HasActiveVersion() {
			multiErr.Add("templateId", errortypes.ErrInvalid,
				"That template has no published version, so the customer would keep "+
					"receiving the current one. Publish it first.")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
