package documenttemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/templateengine"
)

// PublishVersion is the gate between a draft and what customers receive.
//
// Four checks run here and nowhere else, because every one of them fails in a
// way an author cannot see by looking at their own preview:
//
//   - strict parsing, which promotes findings that are advisory while drafting;
//   - required paths over the union of the version's channels, so an invoice
//     cannot go live having lost its total;
//   - a render against the kind's sample context, which exercises every declared
//     variable rather than the handful a hand-test happens to hit;
//   - a render against the hostile probe, because unsafe interpolation is
//     data-driven: the same template is clean with one value and broken with the
//     next, and the next one is a real customer's name.
func (s *Service) PublishVersion(
	ctx context.Context,
	req *services.PublishVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	version, err := s.versionRepo.GetByID(
		ctx,
		&repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	kind := s.kindOf(ctx, version, req.TenantInfo)
	def, ok := s.registry.Get(kind)
	if !ok {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"This template names kind %q, which is not registered, so it cannot render", kind,
		))
	}

	if multiErr := s.VerifyForPublish(ctx, def, version); multiErr != nil {
		return nil, multiErr
	}

	published, err := s.versionRepo.Publish(
		ctx,
		&repositories.PublishDocumentTemplateVersionRequest{
			TenantInfo:    req.TenantInfo,
			VersionID:     req.VersionID,
			PublishedByID: req.UserID,
			PublishNotes:  req.Notes,
		},
	)
	if err != nil {
		return nil, err
	}

	// Publishing changes what every customer of this organization receives, so
	// it is recorded as critical and the content hash goes into the entry — the
	// audit row alone answers which bytes went live.
	s.logAudit(
		ctx,
		published,
		version,
		permission.OpActivate,
		req.UserID,
		"Document template version published",
		auditservice.WithCritical(),
		auditservice.WithMetadata(map[string]any{
			"kind":          string(kind),
			"versionNumber": published.VersionNumber,
			"contentHash":   published.ContentHash,
			"publishNotes":  req.Notes,
		}),
	)

	// The published id is now a stable, immutable cache key, but the draft's
	// compilations were cached under the content hash while it was being
	// previewed. Dropping them keeps the LRU from holding dead drafts.
	s.invalidate(def, version)

	return published, nil
}

// VerifyForPublish runs the gate without publishing, so the editor can show the
// same verdict on the Publish button that the mutation would return.
func (s *Service) VerifyForPublish(
	ctx context.Context,
	def *documenttemplate.KindDefinition,
	version *documenttemplate.DocumentTemplateVersion,
) *errortypes.MultiError {
	if version.Status != documenttemplate.VersionStatusDraft {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("status", errortypes.ErrInvalid, fmt.Sprintf(
			"Only a draft can be published. This version is %s.", version.Status,
		))
		return multiErr
	}

	content := contentFromVersion(version)

	// No cache key: a draft changes under a stable id, so a cached compilation
	// would verify content the author has already replaced.
	compiled := s.compile(def, &content, compileOptions{Strict: true})
	diags := compiled.diags

	// The union across channels, not per channel: a subject line cannot restate
	// a charge table, and a charge table has no business in a subject.
	diags = append(diags, templateengine.CheckRequiredPaths(
		documenttemplate.ChannelPDF.Field(),
		compiled.paths,
		s.registry.RequiredPaths(def.Kind),
	)...)

	diags = append(diags, s.verifyAgainstContexts(ctx, def, compiled)...)

	return diagnosticsToMultiError(diags)
}

// verifyAgainstContexts renders every channel twice.
//
// The sample proves the template survives its own declared data. The hostile
// probe proves it survives data that fights back — the case ZgotmplZ exists to
// report, which a clean sample render cannot detect.
func (s *Service) verifyAgainstContexts(
	ctx context.Context,
	def *documenttemplate.KindDefinition,
	compiled *compiledVersion,
) templateengine.Diagnostics {
	sample, err := s.registry.SampleContext(def.Kind)
	if err != nil {
		return templateengine.Diagnostics{{
			Severity: templateengine.SeverityError,
			Code:     templateengine.CodeSampleRenderFailed,
			Field:    documenttemplate.ChannelPDF.Field(),
			Message: fmt.Sprintf(
				"this kind has no sample context, so the template cannot be verified: %s",
				err.Error(),
			),
		}}
	}

	probe, err := s.registry.HostileProbeContext(def.Kind)
	if err != nil {
		return templateengine.Diagnostics{{
			Severity: templateengine.SeverityError,
			Code:     templateengine.CodeSampleRenderFailed,
			Field:    documenttemplate.ChannelPDF.Field(),
			Message: fmt.Sprintf(
				"this kind has no hostile probe context, so the template cannot be verified: %s",
				err.Error(),
			),
		}}
	}

	var diags templateengine.Diagnostics

	for _, channel := range def.Channels {
		target := compiled.compiledFor(channel)
		if target == nil {
			continue
		}

		for _, data := range []any{sample, probe} {
			diags = append(diags, s.engine.Verify(ctx, templateengine.VerifyRequest{
				Field:    channel.Field(),
				Channel:  channel.Engine(),
				Compiled: target,
				Sample:   data,
			})...)
		}
	}

	return diags
}

// invalidate drops the compilations cached while a version was a draft.
func (s *Service) invalidate(
	def *documenttemplate.KindDefinition,
	version *documenttemplate.DocumentTemplateVersion,
) {
	content := contentFromVersion(version)
	prefix := string(def.Kind) + ":" + contentHashOr(version.ContentHash, &content)

	for _, channel := range def.Channels {
		s.engine.Invalidate(prefix + ":" + string(channel))
	}
}
