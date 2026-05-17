package ediservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type resolvedDocumentContext struct {
	profile         *edi.EDIPartnerDocumentProfile
	templateVersion *edi.EDITemplateVersion
	payload         edi.LoadTenderPayload
	x12Version      string
	runtime         map[string]any
}

func (s *Service) ListDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return s.documentRepo.ListDocumentTypes(ctx, req)
}

func (s *Service) ListTemplates(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return s.documentRepo.ListTemplates(ctx, req)
}

func (s *Service) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *repositories.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return s.documentRepo.ListPartnerDocumentProfiles(ctx, req)
}

func (s *Service) GetPartnerDocumentProfile(
	ctx context.Context,
	req repositories.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	return s.documentRepo.GetPartnerDocumentProfileByID(ctx, req)
}

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
		base, _, err := s.documentRepo.EnsureBase204Template(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		req.TemplateID = base.ID
	}
	templateVersion, err := s.documentRepo.GetActiveTemplateVersion(
		ctx,
		repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			VersionID:  req.TemplateVersionID,
		},
	)
	if err != nil {
		return nil, err
	}
	documentTypes, err := s.documentRepo.ListDocumentTypes(
		ctx,
		repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(documentTypes) == 0 {
		return nil, errors.New("x12 204 outbound document type is not seeded")
	}

	profile := &edi.EDIPartnerDocumentProfile{
		ID:                 req.ProfileID,
		BusinessUnitID:     req.TenantInfo.BuID,
		OrganizationID:     req.TenantInfo.OrgID,
		EDIPartnerID:       req.EDIPartnerID,
		DocumentTypeID:     documentTypes[0].ID,
		TemplateID:         req.TemplateID,
		TemplateVersionID:  req.TemplateVersionID,
		Name:               stringutils.FirstNonEmpty(req.Name, "Outbound X12 204"),
		Status:             req.Status,
		Direction:          edi.DocumentDirectionOutbound,
		Standard:           edi.EDIStandardX12,
		TransactionSet:     edi.TransactionSet204,
		X12VersionOverride: req.X12VersionOverride,
		FunctionalGroupID: stringutils.FirstNonEmpty(
			req.FunctionalGroupID,
			templateVersion.FunctionalGroupID,
			"SM",
		),
		Envelope:        req.Envelope,
		Acknowledgment:  req.Acknowledgment,
		ValidationMode:  req.ValidationMode,
		PartnerSettings: req.PartnerSettings,
		Version:         req.Version,
	}
	if profile.Status == "" {
		profile.Status = edi.DocumentStatusActive
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
	if profile.ID.IsNil() {
		created, createErr := s.documentRepo.CreatePartnerDocumentProfile(ctx, profile)
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
	updated, err := s.documentRepo.UpdatePartnerDocumentProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	s.logAction(updated, actor, permission.OpUpdate, nil, updated, "EDI document profile updated")
	return updated, nil
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
	result, err := edix12.Render204(resolved.renderInput())
	if err != nil {
		return nil, err
	}
	return &EDIDocumentPreview{
		RawX12:                   result.RawX12,
		SegmentCount:             result.SegmentCount,
		X12Version:               resolved.x12Version,
		InterchangeControlNumber: fmt.Sprint(resolved.runtime["isaControlNumber"]),
		GroupControlNumber:       fmt.Sprint(resolved.runtime["groupControlNumber"]),
		TransactionControlNumber: fmt.Sprint(resolved.runtime["transactionControlNumber"]),
		Diagnostics:              result.Diagnostics,
		Profile:                  resolved.profile,
		TemplateVersion:          resolved.templateVersion,
	}, nil
}

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
		Payload:                  req.Payload,
	}
	resolved, err := s.resolveDocumentContext(ctx, previewReq)
	if err != nil {
		return nil, err
	}
	provisional := *resolved
	provisional.runtime = maputils.CloneShallow(resolved.runtime)
	edix12.SetProvisionalControlNumbers(provisional.runtime)
	provisionalResult, err := edix12.Render204(provisional.renderInput())
	if err != nil {
		return nil, err
	}
	if edix12.HasBlockingDiagnostics(
		provisionalResult.Diagnostics,
		resolved.profile.ValidationMode,
	) {
		return nil, diagnosticsToValidationError(provisionalResult.Diagnostics)
	}

	controlNumbers, err := s.documentRepo.AllocateControlNumbers(
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
	result, err := edix12.Render204(resolved.renderInput())
	if err != nil {
		return nil, err
	}
	if edix12.HasBlockingDiagnostics(result.Diagnostics, resolved.profile.ValidationMode) {
		return nil, diagnosticsToValidationError(result.Diagnostics)
	}
	message := &edi.EDIMessage{
		BusinessUnitID:           req.TenantInfo.BuID,
		OrganizationID:           req.TenantInfo.OrgID,
		EDIPartnerID:             resolved.profile.EDIPartnerID,
		DocumentTypeID:           resolved.profile.DocumentTypeID,
		PartnerDocumentProfileID: resolved.profile.ID,
		TemplateID:               resolved.profile.TemplateID,
		TemplateVersionID:        resolved.templateVersion.ID,
		ShipmentID:               resolved.payload.ShipmentID,
		TransferID:               req.TransferID,
		Direction:                edi.DocumentDirectionOutbound,
		Standard:                 edi.EDIStandardX12,
		TransactionSet:           edi.TransactionSet204,
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
	diagnostics := make([]*edi.EDIMessageValidationError, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, &edi.EDIMessageValidationError{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
		})
	}
	return s.documentRepo.CreateMessageWithDiagnostics(
		ctx,
		repositories.CreateEDIMessageWithDiagnosticsRequest{
			Message:     message,
			Diagnostics: diagnostics,
		},
	)
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	return s.documentRepo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	return s.documentRepo.GetMessageByID(ctx, req)
}

func (s *Service) ListTestCases(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	return s.documentRepo.ListTestCases(ctx, req)
}

func (s *Service) PreviewTestCase(
	ctx context.Context,
	testCaseID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*EDIDocumentPreview, error) {
	testCase, err := s.documentRepo.GetTestCaseByID(ctx, repositories.GetEDITestCaseByIDRequest{
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
	templateVersion, err := s.documentRepo.GetActiveTemplateVersion(
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
	payload, err := s.resolvePayload(ctx, req)
	if err != nil {
		return nil, err
	}
	x12Version := stringutils.FirstNonEmpty(
		profile.X12VersionOverride,
		templateVersion.X12Version,
		edi.DefaultX12204Version,
	)
	runtime := edix12.RuntimeValues(profile, x12Version)
	return &resolvedDocumentContext{
		profile:         profile,
		templateVersion: templateVersion,
		payload:         payload,
		x12Version:      x12Version,
		runtime:         runtime,
	}, nil
}

func (s *Service) resolveProfile(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	if !req.PartnerDocumentProfileID.IsNil() {
		return s.documentRepo.GetPartnerDocumentProfileByID(
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
	return s.documentRepo.GetActivePartnerDocumentProfile(
		ctx,
		repositories.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      req.EDIPartnerID,
			TenantInfo:     req.TenantInfo,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		},
	)
}

func (s *Service) resolvePayload(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (edi.LoadTenderPayload, error) {
	if req.Payload != nil {
		return *req.Payload, nil
	}
	if !req.TransferID.IsNil() {
		transfer, err := s.transferRepo.GetTransferByID(ctx, repositories.GetEDITransferByIDRequest{
			ID:         req.TransferID,
			TenantInfo: req.TenantInfo,
			Direction:  "outbound",
		})
		if err != nil {
			return edi.LoadTenderPayload{}, err
		}
		return transfer.TenderPayload, nil
	}
	if req.ShipmentID.IsNil() {
		return edi.LoadTenderPayload{}, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment, transfer, or payload is required",
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
		return edi.LoadTenderPayload{}, err
	}
	return buildTenderPayload(source), nil
}

func (c *resolvedDocumentContext) renderInput() *edix12.RenderInput {
	return &edix12.RenderInput{
		Profile:         c.profile,
		TemplateVersion: c.templateVersion,
		Payload:         c.payload,
		X12Version:      c.x12Version,
		Runtime:         c.runtime,
	}
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
