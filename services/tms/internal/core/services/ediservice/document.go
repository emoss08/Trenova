package ediservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/core/services/edix12inspect"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type resolvedDocumentContext struct {
	ctx                context.Context
	profile            *edi.EDIPartnerDocumentProfile
	templateVersion    *edi.EDITemplateVersion
	payload            edi.DocumentPayload
	x12Version         string
	runtime            map[string]any
	partnerDiagnostics []edix12.Diagnostic
}

func (s *Service) ListDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return s.documentTypeRepo.ListDocumentTypes(ctx, req)
}

func (s *Service) SelectDocumentTypeOptions(
	ctx context.Context,
	req *repositories.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	return s.documentTypeRepo.SelectDocumentTypeOptions(ctx, req)
}

func (s *Service) ListTemplates(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return s.templateRepo.ListTemplates(ctx, req)
}

func (s *Service) SelectTemplateOptions(
	ctx context.Context,
	req *repositories.EDITemplateSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return s.templateRepo.SelectTemplateOptions(ctx, req)
}

func (s *Service) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *repositories.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return s.documentProfileRepo.ListPartnerDocumentProfiles(ctx, req)
}

func (s *Service) SelectPartnerDocumentProfileOptions(
	ctx context.Context,
	req *repositories.EDIPartnerDocumentProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return s.documentProfileRepo.SelectPartnerDocumentProfileOptions(ctx, req)
}

func (s *Service) GetPartnerDocumentProfile(
	ctx context.Context,
	req repositories.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	return s.documentProfileRepo.GetPartnerDocumentProfileByID(ctx, req)
}

//nolint:cyclop,funlen // The profile upsert validates several independent EDI profile invariants explicitly.
func (s *Service) UpsertPartnerDocumentProfile(
	ctx context.Context,
	req *UpsertEDIPartnerDocumentProfileRequest,
	actor *services.RequestActor,
) (*edi.EDIPartnerDocumentProfile, error) {
	if req == nil {
		return nil, s.validator.ValidatePartnerDocumentProfileRequest(req)
	}
	if req.Envelope.ElementSeparator == "" {
		req.Envelope = edi.DefaultX12EnvelopeSettings()
	}
	if multiErr := s.validator.ValidatePartnerDocumentProfileRequest(req); multiErr != nil {
		return nil, multiErr
	}
	if req.TemplateID.IsNil() {
		base, _, err := s.templateRepo.EnsureBase204Template(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		req.TemplateID = base.ID
	}
	templateVersion, err := s.resolveProfileTemplateVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = validateProfileTemplateVersionCompatibility(req.Status, templateVersion); err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetTemplateByID(ctx, repositories.GetEDITemplateByIDRequest{
		ID:         req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	documentType := template.DocumentType
	if documentType == nil {
		documentTypes, listErr := s.documentTypeRepo.ListDocumentTypes(
			ctx,
			repositories.ListEDIDocumentTypesRequest{
				Standard:       template.Standard,
				TransactionSet: template.TransactionSet,
				Direction:      template.Direction,
			},
		)
		if listErr != nil {
			return nil, listErr
		}
		if len(documentTypes) == 0 {
			return nil, errors.New("selected template document type is not seeded")
		}
		documentType = documentTypes[0]
	}

	profile := &edi.EDIPartnerDocumentProfile{
		ID:                req.ProfileID,
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		EDIPartnerID:      req.EDIPartnerID,
		DocumentTypeID:    documentType.ID,
		TemplateID:        req.TemplateID,
		TemplateVersionID: req.TemplateVersionID,
		Name: stringutils.FirstNonEmpty(
			req.Name,
			defaultProfileName(documentType.TransactionSet, documentType.Direction),
		),
		Status:             req.Status,
		Direction:          documentType.Direction,
		Standard:           documentType.Standard,
		TransactionSet:     documentType.TransactionSet,
		X12VersionOverride: req.X12VersionOverride,
		FunctionalGroupID: stringutils.FirstNonEmpty(
			req.FunctionalGroupID,
			templateVersion.FunctionalGroupID,
			edi.FunctionalGroupDefault(documentType.TransactionSet),
		),
		Envelope:                     req.Envelope,
		Acknowledgment:               req.Acknowledgment,
		ValidationMode:               req.ValidationMode,
		PartnerSettings:              req.PartnerSettings,
		PartnerSettingsSchemaID:      req.PartnerSettingsSchemaID,
		PartnerSettingsSchemaVersion: req.PartnerSettingsSchemaVersion,
		Version:                      req.Version,
	}
	if profile.Status == "" {
		profile.Status = edi.DocumentStatusActive
	}
	if err = validateProfileTemplateVersionCompatibility(
		profile.Status,
		templateVersion,
	); err != nil {
		return nil, err
	}
	if profile.ValidationMode == "" {
		profile.ValidationMode = edi.ValidationModeStrict
	}
	if profile.Envelope.ElementSeparator == "" {
		profile.Envelope = edi.DefaultX12EnvelopeSettings()
	}
	if profile.TemplateVersionID.IsNil() {
		profile.TemplateVersionID = templateVersion.ID
	}
	partnerDiagnostics, err := s.validateProfilePartnerSettings(
		ctx,
		profile,
		req.TenantInfo,
		profile.PartnerSettings,
	)
	if err != nil {
		return nil, err
	}
	if profile.Status == edi.DocumentStatusActive &&
		hasPartnerSettingErrorDiagnostics(partnerDiagnostics) {
		return nil, partnerSettingsValidationError(partnerDiagnostics)
	}
	if profile.ID.IsNil() {
		created, createErr := s.documentProfileRepo.CreatePartnerDocumentProfile(ctx, profile)
		if createErr != nil {
			return nil, createErr
		}
		s.logAction(
			created,
			actor,
			permission.OpCreate,
			nil,
			created,
			"EDI document profile created",
		)
		return created, nil
	}
	updated, err := s.documentProfileRepo.UpdatePartnerDocumentProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	s.logAction(updated, actor, permission.OpUpdate, nil, updated, "EDI document profile updated")
	return updated, nil
}

func (s *Service) resolveProfileTemplateVersion(
	ctx context.Context,
	req *UpsertEDIPartnerDocumentProfileRequest,
) (*edi.EDITemplateVersion, error) {
	if req.TemplateVersionID.IsNotNil() {
		return s.templateRepo.GetTemplateVersionByID(
			ctx,
			repositories.GetEDITemplateVersionByIDRequest{
				TemplateID: req.TemplateID,
				VersionID:  req.TemplateVersionID,
				TenantInfo: req.TenantInfo,
			},
		)
	}

	version, err := s.templateRepo.GetActiveTemplateVersion(
		ctx,
		repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err == nil {
		return version, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	if req.Status == "" || req.Status == edi.DocumentStatusActive {
		return nil, errortypes.NewValidationError(
			"templateVersionId",
			errortypes.ErrInvalidOperation,
			"Active document profiles require an active or certified template version. Activate a template version or save the profile as inactive.",
		)
	}

	versions, listErr := s.templateRepo.ListTemplateVersions(
		ctx,
		repositories.ListEDITemplateVersionsRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
		},
	)
	if listErr != nil {
		return nil, listErr
	}
	for _, candidate := range versions {
		if candidate.Status != edi.TemplateStatusArchived {
			return candidate, nil
		}
	}
	return nil, errortypes.NewValidationError(
		"templateVersionId",
		errortypes.ErrInvalidReference,
		"Template version is required",
	)
}

func validateProfileTemplateVersionCompatibility(
	status edi.DocumentStatus,
	version *edi.EDITemplateVersion,
) error {
	if version == nil {
		return errortypes.NewValidationError(
			"templateVersionId",
			errortypes.ErrInvalidReference,
			"Template version is required",
		)
	}
	if status == "" {
		status = edi.DocumentStatusActive
	}
	if status == edi.DocumentStatusActive {
		switch version.Status {
		case edi.TemplateStatusActive, edi.TemplateStatusCertified:
			return nil
		case edi.TemplateStatusDraft,
			edi.TemplateStatusDeprecated,
			edi.TemplateStatusArchived,
			edi.TemplateStatusSuperseded:
			return errortypes.NewValidationError(
				"templateVersionId",
				errortypes.ErrInvalidOperation,
				"Active document profiles can only pin active or certified template versions",
			)
		}
	}
	if version.Status == edi.TemplateStatusArchived {
		return errortypes.NewValidationError(
			"templateVersionId",
			errortypes.ErrInvalidOperation,
			"Archived template versions cannot be used for document profiles",
		)
	}
	return nil
}

func validateProductionTemplateVersion(version *edi.EDITemplateVersion) error {
	if version == nil {
		return errortypes.NewValidationError(
			"templateVersionId",
			errortypes.ErrInvalidReference,
			"Template version is required",
		)
	}
	switch version.Status {
	case edi.TemplateStatusActive, edi.TemplateStatusCertified:
		return nil
	case edi.TemplateStatusDraft,
		edi.TemplateStatusDeprecated,
		edi.TemplateStatusArchived,
		edi.TemplateStatusSuperseded:
		return errortypes.NewValidationError(
			"templateVersionId",
			errortypes.ErrInvalidOperation,
			"Production EDI generation requires an active or certified template version",
		)
	}
	return errortypes.NewValidationError(
		"templateVersionId",
		errortypes.ErrInvalidOperation,
		"Production EDI generation requires an active or certified template version",
	)
}

func (s *Service) PreviewDocument(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*EDIDocumentPreview, error) {
	resolved, err := s.resolveDocumentContext(ctx, req)
	if err != nil {
		return nil, err
	}
	edix12.SetProvisionalControlNumbers(resolved.runtime)
	result, err := edix12.RenderX12(resolved.renderInput())
	if err != nil {
		return nil, err
	}
	diagnostics := mergeEDIDiagnostics(result.Diagnostics, resolved.partnerDiagnostics)
	return &EDIDocumentPreview{
		RawX12:                   result.RawX12,
		SegmentCount:             result.SegmentCount,
		X12Version:               resolved.x12Version,
		InterchangeControlNumber: fmt.Sprint(resolved.runtime["isaControlNumber"]),
		GroupControlNumber:       fmt.Sprint(resolved.runtime["groupControlNumber"]),
		TransactionControlNumber: fmt.Sprint(resolved.runtime["transactionControlNumber"]),
		Diagnostics:              diagnostics,
		Profile:                  resolved.profile,
		TemplateVersion:          resolved.templateVersion,
	}, nil
}

//nolint:funlen // Document generation keeps the validation, render, persist, and audit flow together transactionally.
func (s *Service) GenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"document",
			errortypes.ErrRequired,
			"Document request is required",
		)
	}
	previewReq := &PreviewEDIDocumentRequest{
		TenantInfo:               req.TenantInfo,
		PartnerDocumentProfileID: req.PartnerDocumentProfileID,
		EDIPartnerID:             req.EDIPartnerID,
		ShipmentID:               req.ShipmentID,
		TransferID:               req.TransferID,
		InvoiceID:                req.InvoiceID,
		ShipmentEventID:          req.ShipmentEventID,
		SourceMessageID:          req.SourceMessageID,
		TransactionSet:           req.TransactionSet,
		Direction:                req.Direction,
		Payload:                  req.Payload,
	}
	resolved, err := s.resolveDocumentContext(ctx, previewReq)
	if err != nil {
		return nil, err
	}
	if err = validateProductionTemplateVersion(resolved.templateVersion); err != nil {
		return nil, err
	}
	provisional := *resolved
	provisional.runtime = maputils.CloneShallow(resolved.runtime)
	edix12.SetProvisionalControlNumbers(provisional.runtime)
	provisionalResult, err := edix12.RenderX12(provisional.renderInput())
	if err != nil {
		return nil, err
	}
	provisionalDiagnostics := mergeEDIDiagnostics(
		provisionalResult.Diagnostics,
		provisional.partnerDiagnostics,
	)
	if edix12.HasBlockingDiagnostics(
		provisionalDiagnostics,
		resolved.profile.ValidationMode,
	) {
		return nil, diagnosticsToValidationError(provisionalDiagnostics)
	}

	controlNumbers, err := s.controlNumberRepo.AllocateControlNumbers(
		ctx,
		repositories.AllocateEDIControlNumbersRequest{
			TenantInfo:     req.TenantInfo,
			PartnerID:      resolved.profile.EDIPartnerID,
			DocumentTypeID: resolved.profile.DocumentTypeID,
			Kinds: []edi.ControlNumberKind{
				edi.ControlNumberKindInterchange,
				edi.ControlNumberKindGroup,
				edi.ControlNumberKindTransaction,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	resolved.runtime["isaControlNumber"] = fmt.Sprintf(
		"%09d",
		controlNumbers[edi.ControlNumberKindInterchange],
	)
	resolved.runtime["groupControlNumber"] = strconv.FormatInt(
		controlNumbers[edi.ControlNumberKindGroup],
		10,
	)
	resolved.runtime["transactionControlNumber"] = fmt.Sprintf(
		"%04d",
		controlNumbers[edi.ControlNumberKindTransaction],
	)
	result, err := edix12.RenderX12(resolved.renderInput())
	if err != nil {
		return nil, err
	}
	diagnostics := mergeEDIDiagnostics(result.Diagnostics, resolved.partnerDiagnostics)
	if edix12.HasBlockingDiagnostics(diagnostics, resolved.profile.ValidationMode) {
		return nil, diagnosticsToValidationError(diagnostics)
	}
	message := &edi.EDIMessage{
		BusinessUnitID:           req.TenantInfo.BuID,
		OrganizationID:           req.TenantInfo.OrgID,
		EDIPartnerID:             resolved.profile.EDIPartnerID,
		DocumentTypeID:           resolved.profile.DocumentTypeID,
		PartnerDocumentProfileID: resolved.profile.ID,
		TemplateID:               resolved.profile.TemplateID,
		TemplateVersionID:        resolved.templateVersion.ID,
		ShipmentID:               documentShipmentID(resolved.payload),
		TransferID:               req.TransferID,
		Direction:                resolved.profile.Direction,
		Standard:                 resolved.profile.Standard,
		TransactionSet:           resolved.profile.TransactionSet,
		X12Version:               resolved.x12Version,
		Status:                   edi.MessageStatusGenerated,
		ValidationMode:           resolved.profile.ValidationMode,
		InterchangeControlNumber: fmt.Sprint(resolved.runtime["isaControlNumber"]),
		GroupControlNumber:       fmt.Sprint(resolved.runtime["groupControlNumber"]),
		TransactionControlNumber: fmt.Sprint(resolved.runtime["transactionControlNumber"]),
		SegmentCount:             result.SegmentCount,
		RawX12:                   result.RawX12,
		PayloadSnapshot:          resolved.payload,
		GeneratedByID:            req.GeneratedByID,
	}
	messageDiagnostics := make([]*edi.EDIMessageValidationError, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messageDiagnostics = append(messageDiagnostics, &edi.EDIMessageValidationError{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	return s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		repositories.CreateEDIMessageWithDiagnosticsRequest{
			Message:     message,
			Diagnostics: messageDiagnostics,
		},
	)
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	return s.messageRepo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	return s.messageRepo.GetMessageByID(ctx, req)
}

func (s *Service) InspectX12(
	_ context.Context,
	req *InspectX12Request,
) (*edix12inspect.InspectX12Result, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"inspection",
			errortypes.ErrRequired,
			"Inspection request is required",
		)
	}
	result := edix12inspect.InspectX12(&edix12inspect.InspectX12Request{
		RawX12:         req.RawX12,
		TransactionSet: req.TransactionSet,
		X12Version:     req.X12Version,
		Envelope:       req.Envelope,
		Diagnostics:    req.Diagnostics,
	})
	return &result, nil
}

func (s *Service) InspectMessage(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*EDIMessageInspection, error) {
	message, err := s.messageRepo.GetMessageByID(ctx, req)
	if err != nil {
		return nil, err
	}
	renderDiagnostics := make([]edix12.Diagnostic, 0, len(message.ValidationErrors))
	for _, diagnostic := range message.ValidationErrors {
		renderDiagnostics = append(renderDiagnostics, edix12.Diagnostic{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	inspection := edix12inspect.InspectX12(&edix12inspect.InspectX12Request{
		RawX12:         message.RawX12,
		TransactionSet: message.TransactionSet,
		X12Version:     message.X12Version,
		Envelope:       messageEnvelope(message),
		Diagnostics:    renderDiagnostics,
	})
	return &EDIMessageInspection{
		Message:    message,
		Inspection: inspection,
		Provenance: EDIInspectionProvenance{
			MessageID:         message.ID,
			ProfileID:         message.PartnerDocumentProfileID,
			TemplateID:        message.TemplateID,
			TemplateVersionID: message.TemplateVersionID,
			GeneratedAt:       message.GeneratedAt,
			GeneratedByID:     message.GeneratedByID,
		},
	}, nil
}

func (s *Service) ListTestCases(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	return s.testCaseRepo.ListTestCases(ctx, req)
}

func (s *Service) PreviewTestCase(
	ctx context.Context,
	testCaseID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*EDIDocumentPreview, error) {
	testCase, err := s.testCaseRepo.GetTestCaseByID(ctx, repositories.GetEDITestCaseByIDRequest{
		ID:         testCaseID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	return s.PreviewDocument(ctx, &PreviewEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: testCase.PartnerDocumentProfileID,
		Payload:                  &testCase.Payload,
	})
}

func mergeEDIDiagnostics(
	primary []edix12.Diagnostic,
	additional []edix12.Diagnostic,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0, len(primary)+len(additional))
	diagnostics = append(diagnostics, primary...)
	diagnostics = append(diagnostics, additional...)
	return diagnostics
}

func messageEnvelope(message *edi.EDIMessage) *edi.X12EnvelopeSettings {
	if message == nil || message.PartnerDocumentProfile == nil {
		return nil
	}
	envelope := message.PartnerDocumentProfile.Envelope
	return &envelope
}

func (s *Service) resolveDocumentContext(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*resolvedDocumentContext, error) {
	if multiErr := s.validator.ValidatePreviewDocumentRequest(req); multiErr != nil {
		return nil, multiErr
	}
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	templateVersion, err := s.templateRepo.GetActiveTemplateVersion(
		ctx,
		repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: profile.TemplateID,
			TenantInfo: req.TenantInfo,
			VersionID:  profile.TemplateVersionID,
		},
	)
	if err != nil {
		return nil, err
	}
	payload, err := s.resolvePayload(ctx, req, profile)
	if err != nil {
		return nil, err
	}
	x12Version := stringutils.FirstNonEmpty(
		profile.X12VersionOverride,
		templateVersion.X12Version,
		defaultX12Version(profile.TransactionSet),
	)
	runtime := edix12.RuntimeValues(profile, x12Version)
	partnerDiagnostics, err := s.validateProfilePartnerSettings(
		ctx,
		profile,
		req.TenantInfo,
		profile.PartnerSettings,
	)
	if err != nil {
		return nil, err
	}
	return &resolvedDocumentContext{
		ctx:                ctx,
		profile:            profile,
		templateVersion:    templateVersion,
		payload:            payload,
		x12Version:         x12Version,
		runtime:            runtime,
		partnerDiagnostics: partnerDiagnostics,
	}, nil
}

func (s *Service) resolveProfile(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	if !req.PartnerDocumentProfileID.IsNil() {
		return s.documentProfileRepo.GetPartnerDocumentProfileByID(
			ctx,
			repositories.GetEDIPartnerDocumentProfileByIDRequest{
				ID:         req.PartnerDocumentProfileID,
				TenantInfo: req.TenantInfo,
			},
		)
	}
	if req.EDIPartnerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrRequired,
			"EDI partner or document profile is required",
		)
	}
	profile, err := s.documentProfileRepo.GetActivePartnerDocumentProfile(
		ctx,
		repositories.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      req.EDIPartnerID,
			TenantInfo:     req.TenantInfo,
			TransactionSet: req.TransactionSet,
			Direction:      req.Direction,
		},
	)
	if err != nil && strings.Contains(err.Error(), "multiple active EDI document profiles") {
		return nil, errortypes.NewValidationError(
			"transactionSet",
			errortypes.ErrRequired,
			"Transaction set and direction are required when a partner has multiple active EDI document profiles",
		)
	}
	return profile, err
}

//nolint:cyclop,funlen,gocognit // Payload resolution is explicit per transaction set to preserve validation behavior.
func (s *Service) resolvePayload(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
	profile *edi.EDIPartnerDocumentProfile,
) (edi.DocumentPayload, error) {
	if req.Payload != nil {
		payload := *req.Payload
		payload.Normalize()
		return payload, nil
	}
	transactionSet := profile.TransactionSet
	if transactionSet == "" {
		transactionSet = req.TransactionSet
	}
	if transactionSet == "" {
		transactionSet = edi.TransactionSet204
	}
	if !req.TransferID.IsNil() {
		if transactionSet != edi.TransactionSet204 && transactionSet != edi.TransactionSet990 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"transferId",
				"transfer",
				transactionSet,
				edi.TransactionSet204,
				edi.TransactionSet990,
			)
		}
		transfer, err := s.transferRepo.GetTransferByID(ctx, repositories.GetEDITransferByIDRequest{
			ID:         req.TransferID,
			TenantInfo: req.TenantInfo,
			Direction:  "outbound",
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if transactionSet == edi.TransactionSet990 {
			return buildTenderResponsePayload(transfer), nil
		}
		return edi.NewLoadTenderDocumentPayload(transfer.TenderPayload), nil
	}
	if !req.SourceMessageID.IsNil() {
		if transactionSet != edi.TransactionSet997 && transactionSet != edi.TransactionSet999 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"sourceMessageId",
				"source message",
				transactionSet,
				edi.TransactionSet997,
				edi.TransactionSet999,
			)
		}
		return s.buildAcknowledgmentPayload(ctx, req, transactionSet)
	}
	if !req.InvoiceID.IsNil() {
		if transactionSet != edi.TransactionSet210 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"invoiceId",
				"invoice",
				transactionSet,
				edi.TransactionSet210,
			)
		}
		invoiceEntity, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return buildFreightInvoicePayload(invoiceEntity), nil
	}
	if !req.ShipmentEventID.IsNil() {
		if transactionSet != edi.TransactionSet214 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"shipmentEventId",
				"shipment event",
				transactionSet,
				edi.TransactionSet214,
			)
		}
		event, err := s.shipmentEventRepo.GetByID(ctx, repositories.GetShipmentEventByIDRequest{
			ID:         req.ShipmentEventID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if req.ShipmentID.IsNotNil() && req.ShipmentID != event.ShipmentID {
			return edi.DocumentPayload{}, errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalidReference,
				"Shipment ID must match the shipment event",
			)
		}
		source, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
			ID:         event.ShipmentID,
			TenantInfo: req.TenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return buildShipmentEventStatusPayload(event, source), nil
	}
	if req.ShipmentID.IsNil() {
		return edi.DocumentPayload{}, missingSourceError(transactionSet)
	}
	if transactionSet != edi.TransactionSet204 && transactionSet != edi.TransactionSet214 {
		return edi.DocumentPayload{}, sourceTransactionSetError(
			"shipmentId",
			"shipment",
			transactionSet,
			edi.TransactionSet204,
			edi.TransactionSet214,
		)
	}
	source, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return edi.DocumentPayload{}, err
	}
	if transactionSet == edi.TransactionSet214 {
		return buildShipmentStatusPayload(source), nil
	}
	return edi.NewLoadTenderDocumentPayload(buildTenderPayload(source)), nil
}

func sourceTransactionSetError(
	field string,
	source string,
	actual edi.TransactionSet,
	allowed ...edi.TransactionSet,
) error {
	allowedValues := make([]string, 0, len(allowed))
	for _, transactionSet := range allowed {
		allowedValues = append(allowedValues, string(transactionSet))
	}
	return errortypes.NewValidationError(
		field,
		errortypes.ErrInvalidReference,
		fmt.Sprintf(
			"%s source cannot be used with transaction set %s; expected %s",
			source,
			actual,
			strings.Join(allowedValues, " or "),
		),
	)
}

func missingSourceError(transactionSet edi.TransactionSet) error {
	//nolint:exhaustive // Only transaction sets with source-specific validation need custom errors.
	switch transactionSet {
	case edi.TransactionSet210:
		return errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrRequired,
			"Invoice ID or payload is required for 210 documents",
		)
	case edi.TransactionSet214:
		return errortypes.NewValidationError(
			"shipmentEventId",
			errortypes.ErrRequired,
			"Shipment event ID, shipment ID, or payload is required for 214 documents",
		)
	case edi.TransactionSet990:
		return errortypes.NewValidationError(
			"transferId",
			errortypes.ErrRequired,
			"Transfer ID or payload is required for 990 documents",
		)
	case edi.TransactionSet997, edi.TransactionSet999:
		return errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrRequired,
			"Source message ID or payload is required for acknowledgment documents",
		)
	default:
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment, transfer, invoice, shipment event, source message, or payload is required",
		)
	}
}

func (c *resolvedDocumentContext) renderInput() *edix12.RenderInput {
	return &edix12.RenderInput{
		Context:         c.ctx,
		Profile:         c.profile,
		TemplateVersion: c.templateVersion,
		DocumentPayload: c.payload,
		X12Version:      c.x12Version,
		Runtime:         c.runtime,
	}
}

func defaultX12Version(transactionSet edi.TransactionSet) string {
	if transactionSet == edi.TransactionSet999 {
		return "005010"
	}
	return edi.DefaultX12204Version
}

func defaultProfileName(transactionSet edi.TransactionSet, direction edi.DocumentDirection) string {
	parts := []string{"X12"}
	if transactionSet != "" {
		parts = append(parts, string(transactionSet))
	}
	if direction != "" {
		parts = append(parts, string(direction))
	}
	return strings.Join(parts, " ")
}

func documentShipmentID(payload edi.DocumentPayload) pulid.ID {
	payload.Normalize()
	if payload.Shipment != nil {
		return payload.Shipment.ShipmentID
	}
	if payload.ShipmentStatus != nil {
		return payload.ShipmentStatus.ShipmentID
	}
	if payload.TenderResponse != nil {
		return payload.TenderResponse.ShipmentID
	}
	if payload.FreightInvoice != nil {
		return payload.FreightInvoice.ShipmentID
	}
	return pulid.Nil
}

func diagnosticsToValidationError(diagnostics []edix12.Diagnostic) error {
	multiErr := errortypes.NewMultiError()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		field := stringutils.FirstNonEmpty(diagnostic.Path, diagnostic.SegmentID, "edi")
		multiErr.Add(field, errortypes.ErrInvalid, diagnostic.Message)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
