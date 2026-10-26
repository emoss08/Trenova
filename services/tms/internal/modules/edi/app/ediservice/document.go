package ediservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type resolvedDocumentContext struct {
	ctx                context.Context
	profile            *edi.EDIPartnerDocumentProfile
	templateVersion    *edi.EDITemplateVersion
	payload            edi.DocumentPayload
	x12Version         string
	runtime            map[string]any
	partnerDiagnostics []edix12.Diagnostic
	carrierSCAC        string
}

func (s *Service) ListDocumentTypes(
	ctx context.Context,
	req ediports.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return s.documentTypeRepo.ListDocumentTypes(ctx, req)
}

func (s *Service) SelectDocumentTypeOptions(
	ctx context.Context,
	req *ediports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	return s.documentTypeRepo.SelectDocumentTypeOptions(ctx, req)
}

func (s *Service) ListTemplates(
	ctx context.Context,
	req *ediports.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return s.templateRepo.ListTemplates(ctx, req)
}

func (s *Service) SelectTemplateOptions(
	ctx context.Context,
	req *ediports.EDITemplateSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return s.templateRepo.SelectTemplateOptions(ctx, req)
}

func (s *Service) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *ediports.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return s.documentProfileRepo.ListPartnerDocumentProfiles(ctx, req)
}

func (s *Service) SelectPartnerDocumentProfileOptions(
	ctx context.Context,
	req *ediports.EDIPartnerDocumentProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return s.documentProfileRepo.SelectPartnerDocumentProfileOptions(ctx, req)
}

func (s *Service) GetPartnerDocumentProfile(
	ctx context.Context,
	req ediports.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	return s.documentProfileRepo.GetPartnerDocumentProfileByID(ctx, req)
}

//nolint:cyclop,funlen // The profile upsert validates several independent EDI profile invariants explicitly.
func (s *Service) UpsertPartnerDocumentProfile(
	ctx context.Context,
	req *UpsertEDIPartnerDocumentProfileRequest,
	actor *permissionports.RequestActor,
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
	source, err := s.resolveProfileSource(ctx, req)
	if err != nil {
		return nil, err
	}
	documentType := source.documentType

	profile := &edi.EDIPartnerDocumentProfile{
		ID:                req.ProfileID,
		BusinessUnitID:    req.TenantInfo.BuID,
		OrganizationID:    req.TenantInfo.OrgID,
		EDIPartnerID:      req.EDIPartnerID,
		DocumentTypeID:    documentType.ID,
		TemplateID:        source.templateID,
		TemplateVersionID: source.templateVersionID(),
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
			source.functionalGroupID(),
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
	if source.templateVersion != nil {
		if err = validateProfileTemplateVersionCompatibility(
			profile.Status,
			source.templateVersion,
		); err != nil {
			return nil, err
		}
	}
	if profile.ValidationMode == "" {
		profile.ValidationMode = edi.ValidationModeStrict
	}
	if profile.Envelope.ElementSeparator == "" {
		profile.Envelope = edi.DefaultX12EnvelopeSettings()
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

type profileSource struct {
	documentType    *edi.EDIDocumentType
	templateID      pulid.ID
	templateVersion *edi.EDITemplateVersion
}

func (p *profileSource) templateVersionID() pulid.ID {
	if p.templateVersion == nil {
		return pulid.Nil
	}
	return p.templateVersion.ID
}

func (p *profileSource) functionalGroupID() string {
	if p.templateVersion == nil {
		return ""
	}
	return p.templateVersion.FunctionalGroupID
}

func (s *Service) resolveProfileSource(
	ctx context.Context,
	req *UpsertEDIPartnerDocumentProfileRequest,
) (*profileSource, error) {
	if req.TransactionSet.IsAcknowledgment() {
		documentType, err := s.documentTypeFor(
			ctx,
			edi.EDIStandardX12,
			req.TransactionSet,
			edi.DocumentDirectionOutbound,
		)
		if err != nil {
			return nil, err
		}
		return &profileSource{documentType: documentType}, nil
	}
	if req.TemplateID.IsNil() {
		transactionSet := req.TransactionSet
		if transactionSet == "" {
			transactionSet = edi.TransactionSet204
		}
		base, _, err := s.templateRepo.EnsureBaseTemplate(ctx, req.TenantInfo, transactionSet)
		if err != nil {
			return nil, err
		}
		req.TemplateID = base.ID
	}
	templateVersion, err := s.resolveProfileTemplateVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetTemplateByID(ctx, ediports.GetEDITemplateByIDRequest{
		ID:         req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if req.TransactionSet != "" && req.TransactionSet != template.TransactionSet {
		return nil, errortypes.NewValidationError(
			"templateId",
			errortypes.ErrInvalidReference,
			"The template is for a different transaction set than the profile",
		)
	}
	documentType := template.DocumentType
	if documentType == nil {
		documentType, err = s.documentTypeFor(
			ctx,
			template.Standard,
			template.TransactionSet,
			template.Direction,
		)
		if err != nil {
			return nil, err
		}
	}
	return &profileSource{
		documentType:    documentType,
		templateID:      req.TemplateID,
		templateVersion: templateVersion,
	}, nil
}

func (s *Service) documentTypeFor(
	ctx context.Context,
	standard edi.EDIStandard,
	transactionSet edi.TransactionSet,
	direction edi.DocumentDirection,
) (*edi.EDIDocumentType, error) {
	documentTypes, err := s.documentTypeRepo.ListDocumentTypes(
		ctx,
		ediports.ListEDIDocumentTypesRequest{
			Standard:       standard,
			TransactionSet: transactionSet,
			Direction:      direction,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(documentTypes) == 0 {
		return nil, fmt.Errorf("document type %s %s is not seeded", transactionSet, direction)
	}
	return documentTypes[0], nil
}

func (s *Service) resolveProfileTemplateVersion(
	ctx context.Context,
	req *UpsertEDIPartnerDocumentProfileRequest,
) (*edi.EDITemplateVersion, error) {
	if req.TemplateVersionID.IsNotNil() {
		return s.templateRepo.GetTemplateVersionByID(
			ctx,
			ediports.GetEDITemplateVersionByIDRequest{
				TemplateID: req.TemplateID,
				VersionID:  req.TemplateVersionID,
				TenantInfo: req.TenantInfo,
			},
		)
	}

	version, err := s.templateRepo.GetActiveTemplateVersion(
		ctx,
		ediports.GetActiveEDITemplateVersionRequest{
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
		ediports.ListEDITemplateVersionsRequest{
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
	if req != nil && req.SourceMessageID.IsNotNil() {
		return s.previewMessageAcknowledgment(ctx, req)
	}
	resolved, err := s.resolveDocumentContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return resolved.renderPreview()
}

func (r *resolvedDocumentContext) renderPreview() (*EDIDocumentPreview, error) {
	edix12.SetProvisionalControlNumbers(r.runtime)
	result, err := edix12.RenderX12(r.renderInput())
	if err != nil {
		return nil, err
	}
	diagnostics := mergeEDIDiagnostics(result.Diagnostics, r.partnerDiagnostics)
	return &EDIDocumentPreview{
		RawX12:                   result.RawX12,
		SegmentCount:             result.SegmentCount,
		X12Version:               r.x12Version,
		InterchangeControlNumber: fmt.Sprint(r.runtime["isaControlNumber"]),
		GroupControlNumber:       fmt.Sprint(r.runtime["groupControlNumber"]),
		TransactionControlNumber: fmt.Sprint(r.runtime["transactionControlNumber"]),
		Diagnostics:              diagnostics,
		Profile:                  r.profile,
		TemplateVersion:          r.templateVersion,
	}, nil
}

func (r *resolvedDocumentContext) outboundPreview() (*OutboundDocumentPreview, error) {
	provisional := *r
	provisional.runtime = maputils.CloneShallow(r.runtime)
	preview, err := provisional.renderPreview()
	if err != nil {
		return nil, err
	}

	return &OutboundDocumentPreview{
		Document: preview,
		Blocking: edix12.HasBlockingDiagnostics(preview.Diagnostics, r.profile.ValidationMode),
	}, nil
}

func (r *resolvedDocumentContext) provisionalRender() (*EDIDocumentPreview, error) {
	preview, err := r.outboundPreview()
	if err != nil {
		return nil, err
	}
	if preview.Blocking {
		return nil, diagnosticsToValidationError(preview.Document.Diagnostics)
	}

	return preview.Document, nil
}

func (s *Service) resolveGenerateContext(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*resolvedDocumentContext, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"document",
			errortypes.ErrRequired,
			"Document request is required",
		)
	}
	resolved, err := s.resolveDocumentContext(ctx, &PreviewEDIDocumentRequest{
		TenantInfo:               req.TenantInfo,
		PartnerDocumentProfileID: req.PartnerDocumentProfileID,
		EDIPartnerID:             req.EDIPartnerID,
		ShipmentID:               req.ShipmentID,
		TransferID:               req.TransferID,
		InvoiceID:                req.InvoiceID,
		ShipmentEventID:          req.ShipmentEventID,
		ServiceFailureID:         req.ServiceFailureID,
		SourceMessageID:          req.SourceMessageID,
		TransactionSet:           req.TransactionSet,
		Direction:                req.Direction,
		Payload:                  req.Payload,
		CarrierSCAC:              req.CarrierSCAC,
	})
	if err != nil {
		return nil, err
	}
	if err = validateProductionTemplateVersion(resolved.templateVersion); err != nil {
		return nil, err
	}

	return resolved, nil
}

//nolint:funlen // Document generation keeps the validation, render, persist, and audit flow together transactionally.
func (s *Service) GenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, error) {
	if req != nil && req.SourceMessageID.IsNotNil() {
		return s.generateMessageAcknowledgment(ctx, req)
	}
	if existing, found, err := s.generatedDocument(ctx, req); err != nil || found {
		return existing, err
	}

	resolved, err := s.resolveGenerateContext(ctx, req)
	if err != nil {
		return nil, err
	}
	if _, err = resolved.provisionalRender(); err != nil {
		return nil, err
	}

	controlNumbers, err := s.controlNumberRepo.AllocateControlNumbers(
		ctx,
		ediports.AllocateEDIControlNumbersRequest{
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
		InvoiceID:                req.InvoiceID,
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
		AckStatus:                generatedMessageAckStatus(resolved.profile),
		GeneratedByID:            req.GeneratedByID,
		IdempotencyKey:           req.IdempotencyKey,
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
	created, err := s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		ediports.CreateEDIMessageWithDiagnosticsRequest{
			Message:     message,
			Diagnostics: messageDiagnostics,
		},
	)
	if err != nil {
		if req.IdempotencyKey != "" && dberror.IsUniqueConstraintViolation(err) {
			existing, found, lookupErr := s.generatedDocument(ctx, req)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if found {
				return existing, nil
			}
		}
		return nil, err
	}
	if !req.SuppressTenderRecipientUpsert {
		if err = s.upsertExternalTenderRecipient(ctx, created, resolved.profile); err != nil {
			return nil, err
		}
	}
	if !req.DisableDeliveryQueue {
		if err = s.queueMessageForDelivery(ctx, created); err != nil {
			s.l.Warn(
				"failed to queue EDI message for delivery",
				zap.String("messageId", created.ID.String()),
				zap.Error(err),
			)
		}
	}
	return created, nil
}

func (s *Service) generatedDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, bool, error) {
	if req.IdempotencyKey == "" {
		return nil, false, nil
	}

	existing, err := s.messageRepo.GetMessageByIdempotencyKey(
		ctx,
		ediports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     req.TenantInfo,
			IdempotencyKey: req.IdempotencyKey,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if existing.TransactionSet != req.TransactionSet {
		return nil, false, errortypes.NewBusinessError(
			"This idempotency key already names a {0} document, not a {1}",
			existing.TransactionSet,
			req.TransactionSet,
		)
	}

	if !req.DisableDeliveryQueue {
		if err = s.queueMessageForDelivery(ctx, existing); err != nil {
			s.l.Warn(
				"failed to queue EDI message for delivery",
				zap.String("messageId", existing.ID.String()),
				zap.Error(err),
			)
		}
	}

	return existing, true, nil
}

func generatedMessageAckStatus(
	profile *edi.EDIPartnerDocumentProfile,
) edi.MessageAcknowledgmentStatus {
	if profile == nil ||
		profile.Direction != edi.DocumentDirectionOutbound ||
		!profile.Acknowledgment.Expected ||
		profile.Acknowledgment.Type == edi.AcknowledgmentTypeNone {
		return edi.MessageAcknowledgmentStatusNotExpected
	}
	return edi.MessageAcknowledgmentStatusPending
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *ediports.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	return s.messageRepo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	return s.messageRepo.GetMessageByID(ctx, req)
}

func (s *Service) InspectX12(
	ctx context.Context,
	req *InspectX12Request,
) (*edix12inspect.InspectX12Result, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"inspection",
			errortypes.ErrRequired,
			"Inspection request is required",
		)
	}
	return edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         req.RawX12,
		TransactionSet: req.TransactionSet,
		X12Version:     req.X12Version,
		Envelope:       req.Envelope,
		Diagnostics:    req.Diagnostics,
	})
}

func (s *Service) InspectMessage(
	ctx context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*EDIMessageInspection, error) {
	message, err := s.messageRepo.GetMessageByID(ctx, req)
	if err != nil {
		return nil, err
	}
	renderDiagnostics := make([]edix12.Diagnostic, 0, len(message.ValidationErrors))
	for _, diagnostic := range message.ValidationErrors {
		if strings.HasPrefix(diagnostic.Code, edix12inspect.FindingCodePrefix) {
			continue
		}
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
	inspection, err := edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         message.RawX12,
		TransactionSet: message.TransactionSet,
		X12Version:     message.X12Version,
		Envelope:       messageEnvelope(message),
		Diagnostics:    renderDiagnostics,
	})
	if err != nil {
		return nil, err
	}
	return &EDIMessageInspection{
		Message:    message,
		Inspection: *inspection,
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
	req *ediports.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	return s.testCaseRepo.ListTestCases(ctx, req)
}

func (s *Service) PreviewTestCase(
	ctx context.Context,
	testCaseID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*EDIDocumentPreview, error) {
	testCase, err := s.testCaseRepo.GetTestCaseByID(ctx, ediports.GetEDITestCaseByIDRequest{
		ID:         testCaseID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	preview, err := s.PreviewDocument(ctx, &PreviewEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: testCase.PartnerDocumentProfileID,
		Payload:                  &testCase.Payload,
	})
	if err != nil {
		return nil, err
	}
	s.recordTestCaseRun(ctx, testCase, tenantInfo, preview)
	return preview, nil
}

func (s *Service) recordTestCaseRun(
	ctx context.Context,
	testCase *edi.EDITestCase,
	tenantInfo pagination.TenantInfo,
	preview *EDIDocumentPreview,
) {
	verdict := EvaluateTestCaseRun(testCase, preview.Diagnostics)
	if err := s.testCaseRepo.RecordTestCaseRun(ctx, &ediports.RecordEDITestCaseRunRequest{
		ID:         testCase.ID,
		TenantInfo: tenantInfo,
		RanAt:      timeutils.NowUnix(),
		Passed:     verdict.Passed,
		Warnings:   verdict.Warnings,
		Errors:     verdict.Errors,
	}); err != nil {
		s.l.Warn(
			"failed to record EDI test case run outcome",
			zap.String("testCaseId", testCase.ID.String()),
			zap.Error(err),
		)
	}
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
	if profile.TransactionSet.IsAcknowledgment() {
		return nil, errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrRequired,
			"An acknowledgment is written from the message it answers. Choose that message.",
		)
	}
	templateVersion, err := s.templateRepo.GetActiveTemplateVersion(
		ctx,
		ediports.GetActiveEDITemplateVersionRequest{
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
		carrierSCAC:        req.CarrierSCAC,
	}, nil
}

func (s *Service) resolveProfile(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	if !req.PartnerDocumentProfileID.IsNil() {
		return s.documentProfileRepo.GetPartnerDocumentProfileByID(
			ctx,
			ediports.GetEDIPartnerDocumentProfileByIDRequest{
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
		ediports.GetActiveEDIPartnerDocumentProfileRequest{
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
		transfer, err := s.transferRepo.GetTransferByID(ctx, ediports.GetEDITransferByIDRequest{
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
	if !req.InvoiceID.IsNil() {
		if transactionSet != edi.TransactionSet210 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"invoiceId",
				"invoice",
				transactionSet,
				edi.TransactionSet210,
			)
		}
		invoiceEntity, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if invoiceEntity.Status == invoice.StatusVoided {
			return edi.DocumentPayload{}, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Invoice {0} has been voided and cannot be sent as a freight invoice",
				invoiceEntity.Number,
			)
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
		event, err := s.shipmentEventRepo.GetByID(
			ctx,
			shipmenteventports.GetShipmentEventByIDRequest{
				ID:         req.ShipmentEventID,
				TenantInfo: req.TenantInfo,
			},
		)
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
		source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:         event.ShipmentID,
			TenantInfo: req.TenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return buildShipmentEventStatusPayload(event, source), nil
	}
	if !req.ServiceFailureID.IsNil() {
		if transactionSet != edi.TransactionSet214 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"serviceFailureId",
				"service failure",
				transactionSet,
				edi.TransactionSet214,
			)
		}
		result, err := s.BuildShipmentStatusPayloadForServiceFailure(
			ctx,
			&servicefailureports.BuildServiceFailureEDIPayloadRequest{
				TenantInfo:       req.TenantInfo,
				ServiceFailureID: req.ServiceFailureID,
			},
		)
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return result.Payload, nil
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
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
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
		"{0} source cannot be used with transaction set {1}; expected {2}",
		source,
		actual,
		strings.Join(allowedValues, " or "),
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
		CarrierSCAC:     c.carrierSCAC,
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
