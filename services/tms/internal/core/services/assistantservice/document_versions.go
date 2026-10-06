package assistantservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

const (
	maxRewriteRunes  = 4000
	maxPromptRunes   = 300
	maxVersionNote   = 120
	rewriteMaxTokens = 1200
)

// documentLineage reads a document's versions by any of their ids, checking
// the conversation is the person's and the artifact a document in it.
func (s *Service) documentLineage(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
) ([]*assistantartifact.Artifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	versions, err := s.artifacts.ListLineage(ctx, repositories.LineageRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		ID:         artifactID,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 || versions[0].Kind != assistantartifact.KindDocument {
		return nil, errortypes.NewValidationError(
			"artifact", errortypes.ErrInvalid, "Only a document has versions to edit",
		)
	}

	return versions, nil
}

func latestOf(versions []*assistantartifact.Artifact) *assistantartifact.Artifact {
	latest := versions[0]
	for _, version := range versions[1:] {
		if version.LineageSeq > latest.LineageSeq {
			latest = version
		}
	}

	return latest
}

// appendDocumentVersion keeps a person's text as the document's next version:
// the same title, sources and link, credited to them, with a note of what
// changed. Earlier versions stay as they were.
func (s *Service) appendDocumentVersion(
	ctx context.Context,
	req repositories.GetThreadRequest,
	versions []*assistantartifact.Artifact,
	body, note string,
) (*services.AssistantArtifact, error) {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return nil, errortypes.NewValidationError(
			"body", errortypes.ErrRequired, "A document cannot be saved empty",
		)
	case len(body) > assistantartifact.MaxDocumentBodyBytes:
		return nil, errortypes.NewValidationError(
			"body", errortypes.ErrInvalid, "This document is too long to save",
		)
	}
	note = strings.TrimSpace(note)
	if runes := []rune(note); len(runes) > maxVersionNote {
		note = string(runes[:maxVersionNote])
	}

	latest := latestOf(versions)
	next := &assistantartifact.Artifact{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ID,
		MessageID:      latest.MessageID,
		RunID:          latest.RunID,
		Kind:           assistantartifact.KindDocument,
		Status:         assistantartifact.StatusReady,
		Title:          latest.Title,
		Slug:           latest.Slug,
		Pinned:         latest.Pinned,
		Payload: assistantartifact.DocumentVersion(
			latest.Payload, body, assistantartifact.DocumentEditedByPerson,
			s.personName(ctx, &req.UserID, req.TenantInfo), note,
		),
	}
	next.FollowLineage(latest)

	multiErr := errortypes.NewMultiError()
	next.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.artifacts.InsertVersion(ctx, next)
	if err != nil {
		return nil, err
	}
	if s.activity != nil {
		s.activity.ArtifactChanged(ctx, saved, services.AuditActor{
			PrincipalType: services.PrincipalTypeUser,
			PrincipalID:   req.UserID,
			UserID:        req.UserID,
		}, services.ActivityUpdated)
	}
	out := toAssistantArtifact(saved)

	return &out, nil
}

// SaveDocumentVersion keeps a person's edit of a document as its next version.
func (s *Service) SaveDocumentVersion(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
	version services.SaveDocumentVersionRequest,
) (*services.AssistantArtifact, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	note := version.Note
	if strings.TrimSpace(note) == "" {
		note = "Your edits"
	}

	return s.appendDocumentVersion(ctx, req, versions, version.Body, note)
}

// RestoreDocumentVersion makes an earlier version the latest again, as a new
// version, so nothing written since is lost.
func (s *Service) RestoreDocumentVersion(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
) (*services.AssistantArtifact, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	var restored *assistantartifact.Artifact
	for _, version := range versions {
		if version.ID == artifactID {
			restored = version
		}
	}
	if restored == nil {
		return nil, errortypes.NewNotFoundError("Version not found")
	}

	return s.appendDocumentVersion(
		ctx, req, versions,
		typeutils.StringOfTrimmed(restored.Payload[assistantartifact.DocumentBody]),
		fmt.Sprintf("Restored v%d", max(restored.LineageSeq, 1)),
	)
}

const rewriteSystem = "You edit one passage of a document a logistics team wrote. " +
	"Rewrite only the passage you are given, in the same language and voice, keeping every " +
	"fact, figure, name, date and citation mark like [^2] exactly as written. Do not add " +
	"facts. Keep markdown emphasis where it still fits. Answer with the rewritten passage only."

// RewriteDocument suggests a new wording for a passage of a document. Nothing
// is saved: the person reads the suggestion against the original and accepts
// it as a new version or keeps what was there.
func (s *Service) RewriteDocument(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
	rewrite services.DocumentRewriteRequest,
) (*services.DocumentRewriteSuggestion, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	instruction, err := rewriteInstruction(rewrite)
	if err != nil {
		return nil, err
	}
	latest := latestOf(versions)
	body := typeutils.StringOfTrimmed(latest.Payload[assistantartifact.DocumentBody])
	passage := strings.TrimSpace(rewrite.Text)
	if !strings.Contains(body, passage) {
		return nil, errortypes.NewValidationError(
			"text", errortypes.ErrInvalid, "That passage is not in the latest version",
		)
	}
	if s.completion == nil {
		return nil, errortypes.NewBusinessError("Rewriting is not available here")
	}

	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil {
		return nil, err
	}
	definition := s.threadDefinition(ctx, req)
	attribution := services.AIUsageAttribution{
		UserID:   req.UserID,
		ThreadID: req.ID,
		Feature:  aiusage.FeatureAgentTurn,
	}
	preferred := thread.PreferredProviderID
	if definition != nil {
		attribution.AgentDefinitionID = definition.ID
		version := definition.Version
		attribution.DefinitionVersion = &version
		if preferred.IsNil() {
			preferred = definition.PreferredProviderID
		}
	}

	result, err := s.completion.CompleteStructured(ctx, &services.StructuredCompletionRequest{
		TenantInfo: req.TenantInfo,
		Task:       aiprovider.TaskAssistantChat,
		System:     rewriteSystem,
		Context: services.DelimitedContext{Sections: []services.ContextSection{
			{Title: "Document title", Trusted: true, Content: latest.Title},
			{Title: "Passage to rewrite", Content: passage},
			{Title: "How to rewrite it", Trusted: rewrite.Mode != services.DocumentRewriteAsk, Content: instruction},
		}},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required":             []string{"text"},
			"additionalProperties": false,
		},
		SchemaName:          "document_rewrite",
		MaxTokens:           rewriteMaxTokens,
		PreferredProviderID: preferred,
		Attribution:         attribution,
	})
	if err != nil {
		s.logger.Warn("document rewrite failed", zap.Error(err))
		return nil, errortypes.NewBusinessError("The rewrite could not be written. Try again.")
	}

	var out services.DocumentRewriteSuggestion
	if err = sonic.UnmarshalString(result.Text, &out); err != nil ||
		strings.TrimSpace(out.Text) == "" {
		return nil, errortypes.NewBusinessError("The rewrite came back empty. Try again.")
	}
	out.Text = strings.TrimSpace(out.Text)

	return &out, nil
}

func rewriteInstruction(rewrite services.DocumentRewriteRequest) (string, error) {
	text := strings.TrimSpace(rewrite.Text)
	if text == "" || len([]rune(text)) > maxRewriteRunes {
		return "", errortypes.NewValidationError(
			"text", errortypes.ErrInvalid, "Select a passage of the document to rewrite",
		)
	}
	switch rewrite.Mode {
	case services.DocumentRewriteShorter:
		return "Make it shorter: about half the words, keeping what matters.", nil
	case services.DocumentRewritePlainer:
		return "Make it plainer: short sentences and everyday words, no jargon.", nil
	case services.DocumentRewriteAsk:
		prompt := strings.TrimSpace(rewrite.Prompt)
		if prompt == "" || len([]rune(prompt)) > maxPromptRunes {
			return "", errortypes.NewValidationError(
				"prompt", errortypes.ErrInvalid, "Say how to change it in a sentence",
			)
		}

		return prompt, nil
	default:
		return "", errortypes.NewValidationError(
			"mode", errortypes.ErrInvalid, "Choose shorter, plainer or say how to change it",
		)
	}
}
