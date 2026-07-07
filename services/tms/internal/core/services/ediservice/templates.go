//nolint:gocritic // Template request value types are repository and handler contracts.
package ediservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edistarlark"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) GetTemplate(
	ctx context.Context,
	req repositories.GetEDITemplateByIDRequest,
) (*edi.EDITemplate, error) {
	return s.templateRepo.GetTemplateByID(ctx, req)
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	req *CreateEDITemplateRequest,
	actor *services.RequestActor,
) (*edi.EDITemplate, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"template",
			errortypes.ErrRequired,
			"EDI template is required",
		)
	}
	if err := s.validateTemplateCreateRequest(ctx, req); err != nil {
		return nil, err
	}

	template := &edi.EDITemplate{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		DocumentTypeID: req.DocumentTypeID,
		Name:           strings.TrimSpace(req.Name),
		Description:    strings.TrimSpace(req.Description),
		Direction:      req.Direction,
		Standard:       req.Standard,
		TransactionSet: req.TransactionSet,
		Status:         edi.TemplateStatusDraft,
	}
	version := &edi.EDITemplateVersion{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		VersionNumber:  1,
		X12Version: stringutils.FirstNonEmpty(
			req.X12Version,
			defaultX12Version(req.TransactionSet),
		),
		FunctionalGroupID: stringutils.FirstNonEmpty(
			req.FunctionalGroupID,
			edi.FunctionalGroupDefault(req.TransactionSet),
		),
		Status: edi.TemplateStatusDraft,
		Notes:  strings.TrimSpace(req.Notes),
	}
	segments := cloneTemplateSegments(req.TenantInfo, pulid.Nil, req.Segments)
	if len(segments) == 0 {
		var starterErr error
		segments, starterErr = editemplates.StarterSegments(
			req.TenantInfo,
			pulid.Nil,
			req.TransactionSet,
		)
		if starterErr != nil {
			return nil, starterErr
		}
	}

	created, createdVersion, err := s.templateRepo.CreateTemplate(
		ctx,
		&repositories.CreateEDITemplateRequest{
			Template: template,
			Version:  version,
			Segments: segments,
			ScriptLibraries: cloneTemplateScriptLibraries(
				req.TenantInfo,
				pulid.Nil,
				edi.TemplateStatusDraft,
				req.ScriptLibraries,
			),
		},
	)
	if err != nil {
		return nil, err
	}
	if createdVersion != nil {
		populateTemplateScriptLibraryFunctionNames(createdVersion.ScriptLibraries)
	}
	if createdVersion != nil &&
		createdVersion.Status == edi.TemplateStatusActive &&
		createdVersion.IsActive {
		created.ActiveVersion = createdVersion
	}
	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI template created")
	return created, nil
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	req *UpdateEDITemplateRequest,
	actor *services.RequestActor,
) (*edi.EDITemplate, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"template",
			errortypes.ErrRequired,
			"EDI template is required",
		)
	}
	current, err := s.templateRepo.GetTemplateByID(ctx, repositories.GetEDITemplateByIDRequest{
		ID:         req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if current.Status == edi.TemplateStatusArchived {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived templates cannot be edited",
		)
	}
	if err = validateTemplateStatus(req.Status); err != nil {
		return nil, err
	}
	if err = s.validateTemplateStatusTransition(ctx, current, req); err != nil {
		return nil, err
	}
	original := *current
	if strings.TrimSpace(req.Name) != "" {
		current.Name = strings.TrimSpace(req.Name)
	}
	current.Description = strings.TrimSpace(req.Description)
	if req.Status != "" {
		current.Status = req.Status
	}
	current.Version = req.Version

	updated, err := s.templateRepo.UpdateTemplate(ctx, current)
	if err != nil {
		return nil, err
	}
	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI template updated")
	return updated, nil
}

func (s *Service) CreateDraftVersion(
	ctx context.Context,
	req *CreateEDITemplateDraftRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	source, err := s.sourceTemplateVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	switch source.Status {
	case edi.TemplateStatusActive, edi.TemplateStatusCertified:
	case edi.TemplateStatusDraft,
		edi.TemplateStatusDeprecated,
		edi.TemplateStatusArchived,
		edi.TemplateStatusSuperseded:
		return nil, errortypes.NewValidationError(
			"sourceVersionId",
			errortypes.ErrInvalidOperation,
			"Drafts can only be cloned from active or certified template versions",
		)
	}

	versions, err := s.templateRepo.ListTemplateVersions(
		ctx,
		repositories.ListEDITemplateVersionsRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	nextVersion := int64(1)
	for _, version := range versions {
		if version.VersionNumber >= nextVersion {
			nextVersion = version.VersionNumber + 1
		}
	}

	draft := &edi.EDITemplateVersion{
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		TemplateID:        req.TemplateID,
		SourceVersionID:   source.ID,
		VersionNumber:     nextVersion,
		X12Version:        source.X12Version,
		FunctionalGroupID: source.FunctionalGroupID,
		Status:            edi.TemplateStatusDraft,
		Notes:             strings.TrimSpace(req.Notes),
	}
	segments := cloneTemplateSegments(req.TenantInfo, draft.ID, source.Segments)
	libraries := cloneTemplateScriptLibraries(
		req.TenantInfo,
		draft.ID,
		edi.TemplateStatusDraft,
		source.ScriptLibraries,
	)
	created, err := s.templateRepo.CreateTemplateVersion(
		ctx,
		&repositories.CreateEDITemplateVersionRequest{
			Version:         draft,
			Segments:        segments,
			ScriptLibraries: libraries,
		},
	)
	if err != nil {
		return nil, err
	}
	populateTemplateScriptLibraryFunctionNames(created.ScriptLibraries)
	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI draft version created")
	return created, nil
}

func (s *Service) ListTemplateVersions(
	ctx context.Context,
	req repositories.ListEDITemplateVersionsRequest,
) ([]*edi.EDITemplateVersion, error) {
	return s.templateRepo.ListTemplateVersions(ctx, req)
}

func (s *Service) GetTemplateVersion(
	ctx context.Context,
	req repositories.GetEDITemplateVersionByIDRequest,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(ctx, req)
	if err != nil {
		return nil, err
	}
	populateTemplateScriptLibraryFunctionNames(version.ScriptLibraries)
	return version, nil
}

func (s *Service) UpdateDraftVersion(
	ctx context.Context,
	req *UpdateEDITemplateVersionRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.editableTemplateVersion(ctx, req.TemplateID, req.VersionID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	original := *version
	version.X12Version = stringutils.FirstNonEmpty(req.X12Version, version.X12Version)
	version.FunctionalGroupID = stringutils.FirstNonEmpty(
		req.FunctionalGroupID,
		version.FunctionalGroupID,
	)
	version.Notes = strings.TrimSpace(req.Notes)
	version.Version = req.Version

	updated, err := s.templateRepo.UpdateTemplateVersionMetadata(ctx, version)
	if err != nil {
		return nil, err
	}
	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		&original,
		updated,
		"EDI draft version updated",
	)
	return updated, nil
}

func (s *Service) ReplaceDraftSegments(
	ctx context.Context,
	req *ReplaceEDITemplateSegmentsRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.editableTemplateVersion(ctx, req.TemplateID, req.VersionID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	original := *version
	version.Version = req.Version
	segments := cloneTemplateSegments(req.TenantInfo, version.ID, req.Segments)
	updated, err := s.templateRepo.ReplaceTemplateVersionSegments(
		ctx,
		repositories.ReplaceEDITemplateVersionSegmentsRequest{
			Version:  version,
			Segments: segments,
		},
	)
	if err != nil {
		return nil, err
	}
	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		&original,
		updated,
		"EDI draft segments updated",
	)
	return updated, nil
}

func (s *Service) ListTemplateScriptLibraries(
	ctx context.Context,
	req repositories.ListEDITemplateScriptLibrariesRequest,
) ([]*edi.EDITemplateScriptLibrary, error) {
	libraries, err := s.templateRepo.ListTemplateScriptLibraries(ctx, req)
	if err != nil {
		return nil, err
	}
	populateTemplateScriptLibraryFunctionNames(libraries)
	return libraries, nil
}

func (s *Service) ReplaceDraftScriptLibraries(
	ctx context.Context,
	req *ReplaceEDITemplateScriptLibrariesRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"scriptLibraries",
			errortypes.ErrRequired,
			"Script libraries request is required",
		)
	}
	version, err := s.editableTemplateVersion(ctx, req.TemplateID, req.VersionID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	original := *version
	version.Version = req.Version
	libraries := cloneTemplateScriptLibraries(
		req.TenantInfo,
		version.ID,
		edi.TemplateStatusDraft,
		req.ScriptLibraries,
	)
	updated, err := s.templateRepo.ReplaceTemplateVersionScriptLibraries(
		ctx,
		repositories.ReplaceEDITemplateVersionScriptLibrariesRequest{
			Version:         version,
			ScriptLibraries: libraries,
		},
	)
	if err != nil {
		return nil, err
	}
	populateTemplateScriptLibraryFunctionNames(updated.ScriptLibraries)
	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		&original,
		updated,
		"EDI draft script libraries updated",
	)
	return updated, nil
}

func (s *Service) ValidateTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
) ([]edix12.Diagnostic, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrRequired,
			"Template version is required",
		)
	}
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: req.TemplateID,
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	sourceContext, schemaMissing, err := s.templateSourceContextIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	partnerSettings, err := s.templatePartnerSettingIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	return validateTemplateVersionDefinitionWithSourceContext(
		version,
		sourceContext,
		schemaMissing,
		partnerSettings,
	), nil
}

func (s *Service) CertifyTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.editableTemplateVersion(ctx, req.TemplateID, req.VersionID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	sourceContext, schemaMissing, err := s.templateSourceContextIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	partnerSettings, err := s.templatePartnerSettingIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	diagnostics := validateTemplateVersionDefinitionWithSourceContext(
		version,
		sourceContext,
		schemaMissing,
		partnerSettings,
	)
	if hasTemplateValidationErrors(diagnostics) {
		return nil, diagnosticsToValidationError(diagnostics)
	}

	original := *version
	now := timeutils.NowUnix()
	version.Status = edi.TemplateStatusCertified
	version.CertifiedAt = &now
	version.CertifiedByID = actorID(actor)
	version.CertificationNotes = strings.TrimSpace(req.Notes)
	updated, err := s.templateRepo.UpdateTemplateVersionMetadata(ctx, version)
	if err != nil {
		return nil, err
	}
	s.logAction(updated, actor, permission.OpUpdate, &original, updated, "EDI version certified")
	return updated, nil
}

func (s *Service) ActivateTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: req.TemplateID,
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if version.Status != edi.TemplateStatusCertified {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only certified template versions can be activated",
		)
	}
	return s.activateTemplateVersion(ctx, req, actor, false)
}

func (s *Service) ArchiveTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: req.TemplateID,
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if version.IsActive || version.Status == edi.TemplateStatusActive {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Active template versions cannot be archived",
		)
	}
	if version.Status == edi.TemplateStatusArchived {
		return version, nil
	}
	archived, err := s.templateRepo.ArchiveTemplateVersion(
		ctx,
		repositories.ArchiveEDITemplateVersionRequest{
			VersionID:  req.VersionID,
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			ActorID:    actorID(actor),
			Notes:      strings.TrimSpace(req.Notes),
		},
	)
	if err != nil {
		return nil, err
	}
	s.logAction(archived, actor, permission.OpUpdate, version, archived, "EDI version archived")
	return archived, nil
}

func (s *Service) RollbackTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
	actor *services.RequestActor,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: req.TemplateID,
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	switch version.Status {
	case edi.TemplateStatusCertified, edi.TemplateStatusSuperseded, edi.TemplateStatusDeprecated:
	case edi.TemplateStatusDraft, edi.TemplateStatusActive, edi.TemplateStatusArchived:
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only certified, superseded, or deprecated versions can be rolled back",
		)
	}
	return s.activateTemplateVersion(ctx, req, actor, true)
}

func (s *Service) activateTemplateVersion(
	ctx context.Context,
	req *EDIActionNotesRequest,
	actor *services.RequestActor,
	isRollback bool,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: req.TemplateID,
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if version.Status == edi.TemplateStatusArchived {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived template versions cannot be activated",
		)
	}
	sourceContext, schemaMissing, err := s.templateSourceContextIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	partnerSettings, err := s.templatePartnerSettingIndex(ctx, version, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	diagnostics := validateTemplateVersionDefinitionWithSourceContext(
		version,
		sourceContext,
		schemaMissing,
		partnerSettings,
	)
	if hasTemplateValidationErrors(diagnostics) {
		return nil, diagnosticsToValidationError(diagnostics)
	}
	activated, err := s.templateRepo.ActivateTemplateVersion(
		ctx,
		repositories.ActivateEDITemplateVersionRequest{
			VersionID:  req.VersionID,
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			ActorID:    actorID(actor),
			Notes:      strings.TrimSpace(req.Notes),
			IsRollback: isRollback,
		},
	)
	if err != nil {
		return nil, err
	}
	comment := "EDI version activated"
	if isRollback {
		comment = "EDI version rolled back"
	}
	s.logAction(activated, actor, permission.OpUpdate, version, activated, comment)
	return activated, nil
}

func (s *Service) sourceTemplateVersion(
	ctx context.Context,
	req *CreateEDITemplateDraftRequest,
) (*edi.EDITemplateVersion, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"template",
			errortypes.ErrRequired,
			"Draft request is required",
		)
	}
	if req.SourceVersionID.IsNotNil() {
		return s.templateRepo.GetTemplateVersionByID(
			ctx,
			repositories.GetEDITemplateVersionByIDRequest{
				TemplateID: req.TemplateID,
				VersionID:  req.SourceVersionID,
				TenantInfo: req.TenantInfo,
			},
		)
	}
	return s.templateRepo.GetActiveTemplateVersion(
		ctx,
		repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
		},
	)
}

func (s *Service) editableTemplateVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*edi.EDITemplateVersion, error) {
	version, err := s.templateRepo.GetTemplateVersionByID(
		ctx,
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: templateID,
			VersionID:  versionID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if version.Status != edi.TemplateStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft template versions can be edited",
		)
	}
	return version, nil
}

func (s *Service) templateSourceContextIndex(
	ctx context.Context,
	version *edi.EDITemplateVersion,
	tenantInfo pagination.TenantInfo,
) (*sourceContextIndex, bool, error) {
	if version == nil {
		return nil, false, nil
	}
	if version.Template == nil {
		return nil, false, nil
	}

	schema, err := s.sourceContextRepo.GetActiveSourceContextSchema(
		ctx,
		repositories.GetActiveEDISourceContextSchemaRequest{
			TenantInfo:     tenantInfo,
			Standard:       version.Template.Standard,
			TransactionSet: version.Template.TransactionSet,
			Direction:      version.Template.Direction,
			X12Version:     stringutils.FirstNonEmpty(version.X12Version, edi.DefaultX12204Version),
			ContextKey:     "loadTender",
		},
	)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	fields, err := s.sourceContextRepo.ListSourceContextFields(
		ctx,
		&repositories.ListEDISourceContextFieldsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{
					Limit: 1000,
				},
			},
			SchemaID: schema.ID,
		},
	)
	if err != nil {
		return nil, false, err
	}
	return newSourceContextIndex(fields.Items), false, nil
}

func (s *Service) templatePartnerSettingIndex(
	ctx context.Context,
	version *edi.EDITemplateVersion,
	tenantInfo pagination.TenantInfo,
) (*partnerSettingIndex, error) {
	if version == nil || version.Template == nil {
		return newPartnerSettingIndex(nil), nil
	}

	schema, err := s.partnerSettingRepo.GetActivePartnerSettingSchema(
		ctx,
		repositories.GetActiveEDIPartnerSettingSchemaRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: version.Template.DocumentTypeID,
			Standard:       version.Template.Standard,
			TransactionSet: version.Template.TransactionSet,
			Direction:      version.Template.Direction,
			X12Version:     stringutils.FirstNonEmpty(version.X12Version, edi.DefaultX12204Version),
		},
	)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return newPartnerSettingIndex(nil), nil
		}
		return nil, err
	}

	fields, err := s.partnerSettingRepo.ListPartnerSettingFields(
		ctx,
		&repositories.ListEDIPartnerSettingFieldsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 1000},
			},
			SchemaID: schema.ID,
		},
	)
	if err != nil {
		return nil, err
	}
	return newPartnerSettingIndex(fields.Items), nil
}

func (s *Service) validateTemplateCreateRequest(
	ctx context.Context,
	req *CreateEDITemplateRequest,
) error {
	multiErr := errortypes.NewMultiError()
	if req.DocumentTypeID.IsNil() {
		multiErr.Add("documentTypeId", errortypes.ErrRequired, "Document type is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Template name is required")
	}
	if req.Standard == "" {
		req.Standard = edi.EDIStandardX12
	}
	if req.Direction == "" {
		req.Direction = edi.DocumentDirectionOutbound
	}
	if req.TransactionSet == "" {
		req.TransactionSet = edi.TransactionSet204
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	documentTypes, err := s.documentTypeRepo.ListDocumentTypes(
		ctx,
		repositories.ListEDIDocumentTypesRequest{
			Standard:       req.Standard,
			TransactionSet: req.TransactionSet,
			Direction:      req.Direction,
		},
	)
	if err != nil {
		return err
	}
	if !documentTypesContainID(documentTypes, req.DocumentTypeID) {
		return errortypes.NewValidationError(
			"documentTypeId",
			errortypes.ErrInvalidReference,
			"Document type is not valid for the selected EDI standard, transaction set, and direction",
		)
	}
	return nil
}

func (s *Service) validateTemplateStatusTransition(
	ctx context.Context,
	current *edi.EDITemplate,
	req *UpdateEDITemplateRequest,
) error {
	if req.Status == "" || req.Status == current.Status {
		return nil
	}

	if req.Status == edi.TemplateStatusSuperseded {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Superseded template status is managed by the system and cannot be set directly",
		)
	}
	if req.Status != edi.TemplateStatusActive && req.Status != edi.TemplateStatusCertified {
		return nil
	}

	versions, err := s.templateRepo.ListTemplateVersions(
		ctx,
		repositories.ListEDITemplateVersionsRequest{
			TemplateID: current.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if version == nil {
			continue
		}
		if version.Status == edi.TemplateStatusActive ||
			version.Status == edi.TemplateStatusCertified {
			return nil
		}
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		fmt.Sprintf(
			"Template cannot be marked %s without a certified or active template version",
			req.Status,
		),
	)
}

func validateTemplateStatus(status edi.TemplateStatus) error {
	if status == "" {
		return nil
	}
	switch status {
	case edi.TemplateStatusDraft,
		edi.TemplateStatusCertified,
		edi.TemplateStatusActive,
		edi.TemplateStatusDeprecated,
		edi.TemplateStatusArchived,
		edi.TemplateStatusSuperseded:
		return nil
	default:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Template status is invalid",
		)
	}
}

func documentTypesContainID(documentTypes []*edi.EDIDocumentType, documentTypeID pulid.ID) bool {
	for _, documentType := range documentTypes {
		if documentType != nil && documentType.ID == documentTypeID {
			return true
		}
	}

	return false
}

func cloneTemplateSegments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
	source []*edi.EDITemplateSegment,
) []*edi.EDITemplateSegment {
	segments := make([]*edi.EDITemplateSegment, 0, len(source))
	for _, segment := range source {
		if segment == nil {
			continue
		}
		cloned := *segment
		cloned.ID = pulid.Nil
		cloned.BusinessUnitID = tenantInfo.BuID
		cloned.OrganizationID = tenantInfo.OrgID
		cloned.TemplateVersionID = versionID
		cloned.Elements = append([]edi.TemplateElement{}, segment.Elements...)
		segments = append(segments, &cloned)
	}

	return segments
}

func cloneTemplateScriptLibraries(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
	status edi.TemplateStatus,
	source []*edi.EDITemplateScriptLibrary,
) []*edi.EDITemplateScriptLibrary {
	libraries := make([]*edi.EDITemplateScriptLibrary, 0, len(source))
	for _, library := range source {
		if library == nil {
			continue
		}
		cloned := *library
		cloned.ID = pulid.Nil
		cloned.BusinessUnitID = tenantInfo.BuID
		cloned.OrganizationID = tenantInfo.OrgID
		cloned.TemplateVersionID = versionID
		cloned.Status = status
		cloned.Version = 0
		cloned.FunctionNames = nil
		libraries = append(libraries, &cloned)
	}

	return libraries
}

func populateTemplateScriptLibraryFunctionNames(libraries []*edi.EDITemplateScriptLibrary) {
	for _, library := range libraries {
		if library == nil {
			continue
		}
		names, err := edistarlark.DiscoverFunctionNames(library.Script)
		if err != nil {
			library.FunctionNames = []string{}
			continue
		}
		library.FunctionNames = names
	}
}

func validateTemplateVersionDefinitionWithSourceContext(
	version *edi.EDITemplateVersion,
	sourceContext *sourceContextIndex,
	schemaMissing bool,
	partnerSettings ...*partnerSettingIndex,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if version == nil {
		return append(
			diagnostics,
			templateDiagnostic(
				"template",
				"",
				0,
				"template_required",
				"Template version is required",
				"Select a template version.",
			),
		)
	}

	diagnostics = append(diagnostics, validateTemplateMetadata(version)...)
	diagnostics = append(diagnostics, validateTemplateScriptLibraries(version.ScriptLibraries)...)
	if len(version.Segments) == 0 {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"segments",
				"",
				0,
				"segments_required",
				"At least one template segment is required",
				"Add template segments before certification.",
			),
		)

		return diagnostics
	}

	diagnostics = append(
		diagnostics,
		validateTemplateSegments(version.Segments, version.ScriptLibraries)...)
	diagnostics = append(
		diagnostics,
		validateTemplateSourceContext(version, sourceContext, schemaMissing)...,
	)
	if len(partnerSettings) > 0 {
		diagnostics = append(
			diagnostics,
			validateTemplatePartnerSettings(version, partnerSettings[0], false)...,
		)
	}

	return diagnostics
}

func validateTemplateMetadata(version *edi.EDITemplateVersion) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if strings.TrimSpace(version.X12Version) == "" {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"x12Version",
				"",
				0,
				"metadata_required",
				"X12 version is required",
				"Set the X12 version.",
			),
		)
	}
	if strings.TrimSpace(version.FunctionalGroupID) == "" {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"functionalGroupId",
				"GS",
				1,
				"metadata_required",
				"Functional group ID is required",
				"Set a functional group ID.",
			),
		)
	}

	return diagnostics
}

func validateTemplateSegments(
	source []*edi.EDITemplateSegment,
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	segments := append([]*edi.EDITemplateSegment{}, source...)
	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].Sequence < segments[j].Sequence
	})
	seenSequences := make(map[int64]string, len(segments))
	requiredSegments := map[string]bool{
		"ISA": false,
		"GS":  false,
		"ST":  false,
		"SE":  false,
		"GE":  false,
		"IEA": false,
	}
	for _, segment := range segments {
		if segment == nil {
			diagnostics = append(diagnostics, templateDiagnostic(
				"segments",
				"",
				0,
				"segment_required",
				"Template segment is required",
				"Remove empty segment entries before certification.",
			))
			continue
		}
		if previous, ok := seenSequences[segment.Sequence]; ok {
			diagnostics = append(diagnostics, templateDiagnostic(
				fmt.Sprintf("segments.%d.sequence", segment.Sequence),
				segment.SegmentID,
				0,
				"duplicate_sequence",
				fmt.Sprintf(
					"Segment sequence %d is used by both %s and %s",
					segment.Sequence,
					previous,
					segment.SegmentID,
				),
				"Use a unique sequence for each segment.",
			))
		}
		seenSequences[segment.Sequence] = segment.SegmentID
		if _, ok := requiredSegments[segment.SegmentID]; ok {
			requiredSegments[segment.SegmentID] = true
		}
		diagnostics = append(diagnostics, validateTemplateSegment(segment, libraries)...)
	}
	for segmentID, found := range requiredSegments {
		if found {
			continue
		}
		diagnostics = append(diagnostics, templateDiagnostic(
			"segments",
			segmentID,
			0,
			"required_control_segment_missing",
			segmentID+" segment is required",
			"Add the required X12 control segment.",
		))
	}

	return diagnostics
}

func validateTemplateSegment(
	segment *edi.EDITemplateSegment,
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if strings.TrimSpace(segment.SegmentID) == "" {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"segmentId",
				"",
				0,
				"segment_id_required",
				"Segment ID is required",
				"Set the X12 segment ID.",
			),
		)
	}
	diagnostics = append(
		diagnostics,
		validateTemplateCondition(segment.Condition, segment, nil, libraries)...,
	)
	for idx := range segment.Elements {
		element := &segment.Elements[idx]
		diagnostics = append(diagnostics, validateTemplateElement(segment, element, libraries)...)
	}

	return diagnostics
}

func validateTemplateElement(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if element.Position <= 0 {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"position",
				segment.SegmentID,
				element.Position,
				"element_position_required",
				"Element position is required",
				"Set a 1-based element position.",
			),
		)
	}
	switch element.Source {
	case edi.TemplateElementSourceConstant:
		if element.Validation.Required && strings.TrimSpace(element.Value) == "" {
			diagnostics = append(
				diagnostics,
				templateDiagnostic(
					"value",
					segment.SegmentID,
					element.Position,
					"constant_required",
					"Required constant element has no value",
					"Set the constant value.",
				),
			)
		}
	case edi.TemplateElementSourceFieldPath:
		requireElementPath(&diagnostics, segment, element, element.FieldPath, "fieldPath")
	case edi.TemplateElementSourcePartnerSetting:
		requireElementPath(
			&diagnostics,
			segment,
			element,
			element.PartnerSettingPath,
			"partnerSettingPath",
		)
	case edi.TemplateElementSourceMapping:
		requireElementPath(
			&diagnostics,
			segment,
			element,
			element.MappingSourcePath,
			"mappingSourcePath",
		)
	case edi.TemplateElementSourceRuntime:
		requireElementPath(&diagnostics, segment, element, element.RuntimeKey, "runtimeKey")
	case edi.TemplateElementSourceRepeat:
		requireElementPath(&diagnostics, segment, element, element.RepeatPath, "repeatPath")
	case edi.TemplateElementSourceTransform:
		diagnostics = append(diagnostics, validateTransformElement(segment, element)...)
	case edi.TemplateElementSourceStarlark:
		diagnostics = append(diagnostics, validateStarlarkElement(segment, element, libraries)...)
	default:
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"source",
				segment.SegmentID,
				element.Position,
				"unsupported_source",
				"Element source is unsupported",
				"Choose a supported template element source.",
			),
		)
	}
	diagnostics = append(
		diagnostics,
		validateTemplateCondition(element.Condition, segment, element, libraries)...,
	)
	return diagnostics
}

func requireElementPath(
	diagnostics *[]edix12.Diagnostic,
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	value string,
	path string,
) {
	if strings.TrimSpace(value) != "" || !element.Validation.Required {
		return
	}
	*diagnostics = append(*diagnostics, templateDiagnostic(
		path,
		segment.SegmentID,
		element.Position,
		"source_path_required",
		"Required element source path is missing",
		"Set the source path or make the element optional.",
	))
}

func validateTransformElement(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if element.BaseSource == nil {
		return append(diagnostics, templateDiagnostic(
			"baseSource",
			segment.SegmentID,
			element.Position,
			"transform_base_source_required",
			"Transform base source is required",
			"Choose the source value that feeds the transform pipeline.",
		))
	}

	if !edix12.IsDirectElementSource(element.BaseSource.Source) {
		diagnostics = append(diagnostics, templateDiagnostic(
			"baseSource.source",
			segment.SegmentID,
			element.Position,
			"transform_base_source_invalid",
			"Transform base source must be a direct source",
			"Use a constant, field path, partner setting, runtime, repeat, or mapping source.",
		))
	}

	for _, step := range element.TransformPipeline {
		if !edix12.IsSupportedTransformOperation(step.Operation) {
			diagnostics = append(diagnostics, templateDiagnostic(
				"transformPipeline.operation",
				segment.SegmentID,
				element.Position,
				"unsupported_transform_operation",
				"Unsupported transform operation "+step.Operation,
				"Choose a supported transform operation.",
			))
		}
	}

	return diagnostics
}

func validateTemplateScriptLibraries(
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	seenNames := make(map[string]string, len(libraries))
	starlarkLibs := make([]edistarlark.ScriptLibrary, 0, len(libraries))
	for idx, library := range libraries {
		path := fmt.Sprintf("scriptLibraries.%d", idx)
		if library == nil {
			diagnostics = append(diagnostics, templateDiagnostic(
				path,
				"",
				0,
				"script_library_required",
				"Script library is required",
				"Remove empty library entries.",
			))
			continue
		}

		name := strings.TrimSpace(library.Name)
		if name == "" {
			diagnostics = append(diagnostics, templateDiagnostic(
				path+".name",
				"",
				0,
				"script_library_name_required",
				"Script library name is required",
				"Set a unique library name.",
			))
		}
		nameKey := strings.ToLower(name)
		if previous, ok := seenNames[nameKey]; ok && nameKey != "" {
			diagnostics = append(diagnostics, templateDiagnostic(
				path+".name",
				"",
				0,
				"script_library_duplicate_name",
				fmt.Sprintf(
					"Script library name %q duplicates %q for this template version",
					name,
					previous,
				),
				"Use a unique library name for this template version.",
			))
		}
		seenNames[nameKey] = name

		if library.Language != edi.ScriptLanguageStarlark {
			diagnostics = append(diagnostics, templateDiagnostic(
				path+".language",
				"",
				0,
				"script_library_language_invalid",
				"Script library language is invalid",
				"Use Starlark.",
			))
		}
		if strings.TrimSpace(library.Script) == "" {
			diagnostics = append(diagnostics, templateDiagnostic(
				path+".script",
				"",
				0,
				"script_library_required",
				"Script library script is required",
				"Add Starlark functions to this library.",
			))
			continue
		}
		starlarkLibs = append(starlarkLibs, edistarlark.ScriptLibrary{
			Name:   name,
			Script: library.Script,
		})
	}

	for _, diagnostic := range edistarlark.ValidateLibraries(starlarkLibs) {
		diagnostics = append(diagnostics, starlarkTemplateDiagnostic(diagnostic))
	}
	return diagnostics
}

func validateStarlarkElement(
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if strings.TrimSpace(element.StarlarkScript) == "" &&
		strings.TrimSpace(element.StarlarkFunction) == "" {
		diagnostics = append(
			diagnostics,
			templateDiagnostic(
				"starlarkFunction",
				segment.SegmentID,
				element.Position,
				"script_function_not_found",
				"Starlark function is required",
				"Set a library function or add an inline Starlark script.",
			),
		)
		return diagnostics
	}

	functionName := stringutils.FirstNonEmpty(strings.TrimSpace(element.StarlarkFunction), "value")
	starlarkDiagnostics := edistarlark.ValidateScriptFunction(edistarlark.EvalRequest{
		Script:       element.StarlarkScript,
		FunctionName: functionName,
		Libraries:    starlarkLibraries(libraries),
		Context:      map[string]any{},
		SegmentID:    segment.SegmentID,
		Path:         "starlark:" + functionName,
	})
	for _, diagnostic := range starlarkDiagnostics {
		converted := starlarkTemplateDiagnostic(diagnostic)
		converted.SegmentID = segment.SegmentID
		converted.ElementPosition = element.Position
		diagnostics = append(diagnostics, converted)
	}
	return diagnostics
}

func validateTemplateCondition(
	condition string,
	segment *edi.EDITemplateSegment,
	element *edi.TemplateElement,
	libraries []*edi.EDITemplateScriptLibrary,
) []edix12.Diagnostic {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return nil
	}
	if !strings.HasPrefix(condition, "starlark:") {
		if diagnostic := edix12.ValidateConditionSyntax(condition); diagnostic != nil {
			diagnostic.SegmentID = segment.SegmentID
			if element != nil {
				diagnostic.ElementPosition = element.Position
			}
			return []edix12.Diagnostic{*diagnostic}
		}
		return nil
	}

	body := strings.TrimSpace(strings.TrimPrefix(condition, "starlark:"))
	functionName := "include"
	script := body
	path := "condition:starlark:include"
	if isFunctionReference(body) {
		functionName = body
		script = ""
		path = "condition:starlark:" + functionName
	}
	starlarkDiagnostics := edistarlark.ValidateScriptFunction(edistarlark.EvalRequest{
		Script:       script,
		FunctionName: functionName,
		Libraries:    starlarkLibraries(libraries),
		Context:      map[string]any{},
		SegmentID:    segment.SegmentID,
		Path:         path,
	})
	diagnostics := make([]edix12.Diagnostic, 0, len(starlarkDiagnostics))
	for _, diagnostic := range starlarkDiagnostics {
		converted := starlarkTemplateDiagnostic(diagnostic)
		converted.SegmentID = segment.SegmentID
		if element != nil {
			converted.ElementPosition = element.Position
		}
		diagnostics = append(diagnostics, converted)
	}
	return diagnostics
}

func starlarkLibraries(
	source []*edi.EDITemplateScriptLibrary,
) []edistarlark.ScriptLibrary {
	libraries := make([]edistarlark.ScriptLibrary, 0, len(source))
	for _, library := range source {
		if library == nil || library.Language != edi.ScriptLanguageStarlark {
			continue
		}
		libraries = append(libraries, edistarlark.ScriptLibrary{
			Name:   library.Name,
			Script: library.Script,
		})
	}
	return libraries
}

func starlarkTemplateDiagnostic(diagnostic edistarlark.Diagnostic) edix12.Diagnostic {
	return edix12.Diagnostic{
		Severity:        edi.ValidationSeverity(diagnostic.Severity),
		Code:            diagnostic.Code,
		SegmentID:       diagnostic.SegmentID,
		ElementPosition: diagnostic.ElementPosition,
		Path:            diagnostic.Path,
		Message:         diagnostic.Message,
		SuggestedFix:    diagnostic.SuggestedFix,
	}
}

func isFunctionReference(value string) bool {
	if value == "" {
		return false
	}
	for idx, char := range value {
		if idx == 0 {
			if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
				continue
			}
			return false
		}
		if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}

func actorID(actor *services.RequestActor) pulid.ID {
	if actor == nil {
		return pulid.Nil
	}
	return actor.UserID
}

func templateDiagnostic(
	path string,
	segmentID string,
	position int,
	code string,
	message string,
	suggestedFix string,
) edix12.Diagnostic {
	return edix12.Diagnostic{
		Severity:        edi.ValidationSeverityError,
		Code:            code,
		SegmentID:       segmentID,
		ElementPosition: position,
		Path:            path,
		Message:         message,
		SuggestedFix:    suggestedFix,
	}
}

func hasTemplateValidationErrors(diagnostics []edix12.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}

	return false
}
