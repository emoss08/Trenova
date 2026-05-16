package ediservice

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/shopspring/decimal"
)

type resolvedDocumentContext struct {
	profile         *edi.EDIPartnerDocumentProfile
	templateVersion *edi.EDITemplateVersion
	payload         edi.LoadTenderPayload
	x12Version      string
	runtime         map[string]any
}

type renderResult struct {
	rawX12       string
	segmentCount int64
	diagnostics  []EDIDiagnostic
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
	if _, _, err := s.documentRepo.EnsureBase204Template(ctx, req.Filter.TenantInfo); err != nil {
		return nil, err
	}
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
		return nil, errortypes.NewValidationError("profile", errortypes.ErrRequired, "Profile is required")
	}
	if req.EDIPartnerID.IsNil() {
		return nil, errortypes.NewValidationError("ediPartnerId", errortypes.ErrRequired, "EDI partner is required")
	}
	if req.TemplateID.IsNil() {
		base, _, err := s.documentRepo.EnsureBase204Template(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		req.TemplateID = base.ID
	}
	templateVersion, err := s.documentRepo.GetActiveTemplateVersion(ctx, repositories.GetActiveEDITemplateVersionRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
		VersionID:  req.TemplateVersionID,
	})
	if err != nil {
		return nil, err
	}
	documentTypes, err := s.documentRepo.ListDocumentTypes(ctx, repositories.ListEDIDocumentTypesRequest{
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
	})
	if err != nil {
		return nil, err
	}
	if len(documentTypes) == 0 {
		return nil, fmt.Errorf("x12 204 outbound document type is not seeded")
	}

	profile := &edi.EDIPartnerDocumentProfile{
		ID:                 req.ProfileID,
		BusinessUnitID:     req.TenantInfo.BuID,
		OrganizationID:     req.TenantInfo.OrgID,
		EDIPartnerID:       req.EDIPartnerID,
		DocumentTypeID:     documentTypes[0].ID,
		TemplateID:         req.TemplateID,
		TemplateVersionID:  req.TemplateVersionID,
		Name:               firstNonEmpty(req.Name, "Outbound X12 204"),
		Status:             req.Status,
		Direction:          edi.DocumentDirectionOutbound,
		Standard:           edi.EDIStandardX12,
		TransactionSet:     edi.TransactionSet204,
		X12VersionOverride: req.X12VersionOverride,
		FunctionalGroupID:  firstNonEmpty(req.FunctionalGroupID, templateVersion.FunctionalGroupID, "SM"),
		Envelope:           req.Envelope,
		Acknowledgment:     req.Acknowledgment,
		ValidationMode:     req.ValidationMode,
		PartnerSettings:    req.PartnerSettings,
		Version:            req.Version,
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
		s.logAction(created, actor, permission.OpCreate, nil, created, "EDI document profile created")
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
	setProvisionalControlNumbers(resolved.runtime)
	result, err := renderX12204(resolved)
	if err != nil {
		return nil, err
	}
	return &EDIDocumentPreview{
		RawX12:                   result.rawX12,
		SegmentCount:             result.segmentCount,
		X12Version:               resolved.x12Version,
		InterchangeControlNumber: fmt.Sprint(resolved.runtime["isaControlNumber"]),
		GroupControlNumber:       fmt.Sprint(resolved.runtime["groupControlNumber"]),
		TransactionControlNumber: fmt.Sprint(resolved.runtime["transactionControlNumber"]),
		Diagnostics:              result.diagnostics,
		Profile:                  resolved.profile,
		TemplateVersion:          resolved.templateVersion,
	}, nil
}

func (s *Service) GenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, error) {
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
	provisional.runtime = cloneMap(resolved.runtime)
	setProvisionalControlNumbers(provisional.runtime)
	provisionalResult, err := renderX12204(&provisional)
	if err != nil {
		return nil, err
	}
	if hasBlockingDiagnostics(provisionalResult.diagnostics, resolved.profile.ValidationMode) {
		return nil, diagnosticsToValidationError(provisionalResult.diagnostics)
	}

	controlNumbers, err := s.documentRepo.AllocateControlNumbers(ctx, repositories.AllocateEDIControlNumbersRequest{
		TenantInfo:     req.TenantInfo,
		PartnerID:      resolved.profile.EDIPartnerID,
		DocumentTypeID: resolved.profile.DocumentTypeID,
		Kinds: []edi.ControlNumberKind{
			edi.ControlNumberKindInterchange,
			edi.ControlNumberKindGroup,
			edi.ControlNumberKindTransaction,
		},
	})
	if err != nil {
		return nil, err
	}
	resolved.runtime["isaControlNumber"] = fmt.Sprintf("%09d", controlNumbers[edi.ControlNumberKindInterchange])
	resolved.runtime["groupControlNumber"] = strconv.FormatInt(controlNumbers[edi.ControlNumberKindGroup], 10)
	resolved.runtime["transactionControlNumber"] = fmt.Sprintf("%04d", controlNumbers[edi.ControlNumberKindTransaction])
	result, err := renderX12204(resolved)
	if err != nil {
		return nil, err
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
		SegmentCount:             result.segmentCount,
		RawX12:                   result.rawX12,
		PayloadSnapshot:          resolved.payload,
		GeneratedByID:            req.GeneratedByID,
	}
	diagnostics := make([]*edi.EDIMessageValidationError, 0, len(result.diagnostics))
	for _, diagnostic := range result.diagnostics {
		diagnostics = append(diagnostics, &edi.EDIMessageValidationError{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
		})
	}
	return s.documentRepo.CreateMessageWithDiagnostics(ctx, repositories.CreateEDIMessageWithDiagnosticsRequest{
		Message:     message,
		Diagnostics: diagnostics,
	})
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

func (s *Service) PreviewTestCase(ctx context.Context, testCaseID pulid.ID, tenantInfo pagination.TenantInfo) (*EDIDocumentPreview, error) {
	cases, err := s.documentRepo.ListTestCases(ctx, &repositories.ListEDITestCasesRequest{
		Filter: &pagination.QueryOptions{TenantInfo: tenantInfo},
	})
	if err != nil {
		return nil, err
	}
	for _, testCase := range cases.Items {
		if testCase.ID == testCaseID {
			return s.PreviewDocument(ctx, &PreviewEDIDocumentRequest{
				TenantInfo:               tenantInfo,
				PartnerDocumentProfileID: testCase.PartnerDocumentProfileID,
				Payload:                  &testCase.Payload,
			})
		}
	}
	return nil, dberror.HandleNotFoundError(sql.ErrNoRows, "EDITestCase")
}

func (s *Service) resolveDocumentContext(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*resolvedDocumentContext, error) {
	if req == nil {
		return nil, errortypes.NewValidationError("document", errortypes.ErrRequired, "Document request is required")
	}
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	templateVersion, err := s.documentRepo.GetActiveTemplateVersion(ctx, repositories.GetActiveEDITemplateVersionRequest{
		TemplateID: profile.TemplateID,
		TenantInfo: req.TenantInfo,
		VersionID:  profile.TemplateVersionID,
	})
	if err != nil {
		return nil, err
	}
	payload, err := s.resolvePayload(ctx, req)
	if err != nil {
		return nil, err
	}
	x12Version := firstNonEmpty(profile.X12VersionOverride, templateVersion.X12Version, edi.DefaultX12204Version)
	runtime := runtimeValues(profile, x12Version)
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
		return s.documentRepo.GetPartnerDocumentProfileByID(ctx, repositories.GetEDIPartnerDocumentProfileByIDRequest{
			ID:         req.PartnerDocumentProfileID,
			TenantInfo: req.TenantInfo,
		})
	}
	if req.EDIPartnerID.IsNil() {
		return nil, errortypes.NewValidationError("ediPartnerId", errortypes.ErrRequired, "EDI partner or document profile is required")
	}
	return s.documentRepo.GetActivePartnerDocumentProfile(ctx, repositories.GetActiveEDIPartnerDocumentProfileRequest{
		PartnerID:      req.EDIPartnerID,
		TenantInfo:     req.TenantInfo,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
	})
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
		return edi.LoadTenderPayload{}, errortypes.NewValidationError("shipmentId", errortypes.ErrRequired, "Shipment, transfer, or payload is required")
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

func renderX12204(ctx *resolvedDocumentContext) (*renderResult, error) {
	payloadMap, err := jsonutils.ToJSON(ctx.payload)
	if err != nil {
		return nil, err
	}
	segments := append([]*edi.EDITemplateSegment{}, ctx.templateVersion.Segments...)
	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].Sequence < segments[j].Sequence
	})
	rendered := make([]string, 0, len(segments)+8)
	diagnostics := make([]EDIDiagnostic, 0)
	for _, segment := range segments {
		repeats := repeatValues(payloadMap, segment.RepeatPath)
		if len(repeats) == 0 {
			repeats = []any{nil}
		}
		for _, repeatValue := range repeats {
			env := expressionEnv(payloadMap, ctx.profile.PartnerSettings, ctx.runtime, repeatValue)
			include, err := evaluateCondition(segment.Condition, env)
			if err != nil {
				diagnostics = append(diagnostics, expressionDiagnostic(segment, 0, segment.Condition, err))
				continue
			}
			if !include {
				continue
			}
			elements := make([]string, 0, len(segment.Elements))
			segmentHasValue := segment.Required
			for _, element := range segment.Elements {
				value, elementDiagnostics := resolveElement(segment, element, env)
				diagnostics = append(diagnostics, elementDiagnostics...)
				if value != "" {
					segmentHasValue = true
				}
				elements = append(elements, sanitizeX12Value(value, ctx.profile.Envelope))
			}
			if !segmentHasValue && !segment.Required {
				continue
			}
			rendered = append(rendered, strings.Join(append([]string{segment.SegmentID}, trimTrailingEmpty(elements)...), ctx.profile.Envelope.ElementSeparator))
		}
	}
	applyTrailerCounts(rendered, ctx.profile.Envelope.ElementSeparator)
	raw := strings.Join(rendered, ctx.profile.Envelope.SegmentTerminator) + ctx.profile.Envelope.SegmentTerminator
	return &renderResult{rawX12: raw, segmentCount: int64(len(rendered)), diagnostics: filterDiagnostics(diagnostics, ctx.profile.ValidationMode)}, nil
}

func resolveElement(
	segment *edi.EDITemplateSegment,
	element edi.TemplateElement,
	env map[string]any,
) (string, []EDIDiagnostic) {
	diagnostics := []EDIDiagnostic{}
	include, err := evaluateCondition(element.Condition, env)
	if err != nil {
		return "", []EDIDiagnostic{expressionDiagnostic(segment, element.Position, element.Condition, err)}
	}
	if !include {
		return "", diagnostics
	}
	value := ""
	switch element.Source {
	case edi.TemplateElementSourceConstant:
		value = element.Value
	case edi.TemplateElementSourceFieldPath:
		value = valueToString(getPath(env, "shipment."+element.FieldPath))
	case edi.TemplateElementSourcePartnerSetting:
		value = valueToString(getPath(env, "partner."+firstNonEmpty(element.PartnerSettingPath, element.Name)))
	case edi.TemplateElementSourceRuntime:
		value = valueToString(getPath(env, "runtime."+element.RuntimeKey))
	case edi.TemplateElementSourceRepeat:
		value = valueToString(getPath(env, "repeat."+element.RepeatPath))
	case edi.TemplateElementSourceExpression:
		value, err = evaluateExpression(element.Expression, env)
		if err != nil {
			diagnostics = append(diagnostics, expressionDiagnostic(segment, element.Position, element.Expression, err))
		}
	case edi.TemplateElementSourceMapping:
		value = valueToString(getPath(env, "mapping."+element.MappingSourcePath))
	}
	if value == "" {
		value = element.Default
	}
	if element.Validation.Required && strings.TrimSpace(value) == "" {
		diagnostics = append(diagnostics, EDIDiagnostic{
			Severity:        edi.ValidationSeverityError,
			Code:            firstNonEmpty(element.Validation.Code, "required"),
			SegmentID:       segment.SegmentID,
			ElementPosition: element.Position,
			Path:            firstNonEmpty(element.FieldPath, element.RuntimeKey, element.Expression),
			Message:         firstNonEmpty(element.Validation.Message, element.Name+" is required"),
		})
	}
	if element.Validation.MaxLength > 0 && len(value) > element.Validation.MaxLength {
		diagnostics = append(diagnostics, EDIDiagnostic{
			Severity:        edi.ValidationSeverityWarning,
			Code:            "max_length",
			SegmentID:       segment.SegmentID,
			ElementPosition: element.Position,
			Message:         fmt.Sprintf("%s exceeds max length %d", element.Name, element.Validation.MaxLength),
		})
		value = value[:element.Validation.MaxLength]
	}
	return value, diagnostics
}

func runtimeValues(profile *edi.EDIPartnerDocumentProfile, x12Version string) map[string]any {
	now := time.Now().UTC()
	envelope := profile.Envelope
	return map[string]any{
		"interchangeSenderId":     padISAID(envelope.InterchangeSenderID),
		"interchangeReceiverId":   padISAID(envelope.InterchangeReceiverID),
		"applicationSenderCode":   firstNonEmpty(envelope.ApplicationSenderCode, envelope.InterchangeSenderID),
		"applicationReceiverCode": firstNonEmpty(envelope.ApplicationReceiverCode, envelope.InterchangeReceiverID),
		"usageIndicator":          firstNonEmpty(envelope.InterchangeUsageIndicator, "T"),
		"componentSeparator":      firstNonEmpty(envelope.ComponentSeparator, ">"),
		"repetitionSeparator":     firstNonEmpty(envelope.RepetitionSeparator, "^"),
		"functionalGroupId":       firstNonEmpty(profile.FunctionalGroupID, "SM"),
		"x12Version":              x12Version,
		"interchangeDate":         now.Format("060102"),
		"interchangeTime":         now.Format("1504"),
		"groupDate":               now.Format("20060102"),
		"groupTime":               now.Format("1504"),
	}
}

func setProvisionalControlNumbers(runtime map[string]any) {
	runtime["isaControlNumber"] = "000000000"
	runtime["groupControlNumber"] = "0"
	runtime["transactionControlNumber"] = "0000"
}

func applyTrailerCounts(rendered []string, separator string) {
	stIndex := -1
	transactionCount := 0
	controlNumber := ""
	for i, segment := range rendered {
		parts := strings.Split(segment, separator)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "ST":
			stIndex = i
			if len(parts) > 2 {
				controlNumber = parts[2]
			}
		case "SE":
			if stIndex >= 0 {
				transactionCount = i - stIndex + 1
			}
			if len(parts) > 1 {
				parts[1] = strconv.Itoa(transactionCount)
			}
			if len(parts) > 2 && controlNumber != "" {
				parts[2] = controlNumber
			}
			rendered[i] = strings.Join(parts, separator)
		}
	}
}

func expressionEnv(
	shipment map[string]any,
	partner map[string]any,
	runtime map[string]any,
	repeat any,
) map[string]any {
	if partner == nil {
		partner = map[string]any{}
	}
	return map[string]any{
		"shipment": shipment,
		"partner":  partner,
		"mapping":  map[string]any{},
		"runtime":  runtime,
		"repeat":   repeat,
	}
}

func repeatValues(payload map[string]any, path string) []any {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	value := getPath(map[string]any{"shipment": payload}, "shipment."+path)
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	return items
}

func getPath(root any, path string) any {
	current := root
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		switch typed := current.(type) {
		case map[string]any:
			current = typed[part]
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	return current
}

func evaluateCondition(condition string, env map[string]any) (bool, error) {
	if strings.TrimSpace(condition) == "" {
		return true, nil
	}
	program, err := expr.Compile(condition, expr.Env(env))
	if err != nil {
		return false, err
	}
	result, err := vm.Run(program, env)
	if err != nil {
		return false, err
	}
	value, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition did not return a boolean")
	}
	return value, nil
}

func evaluateExpression(expression string, env map[string]any) (string, error) {
	if strings.TrimSpace(expression) == "" {
		return "", nil
	}
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return "", err
	}
	result, err := vm.Run(program, env)
	if err != nil {
		return "", err
	}
	return valueToString(result), nil
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return typed.String()
	case decimal.NullDecimal:
		if !typed.Valid {
			return ""
		}
		return typed.Decimal.StringFixed(2)
	case decimal.Decimal:
		return typed.StringFixed(2)
	case float64:
		return trimFloat(typed)
	case float32:
		return trimFloat(float64(typed))
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "Y"
		}
		return "N"
	case map[string]any:
		if valid, ok := typed["Valid"].(bool); ok && !valid {
			return ""
		}
		if decimalValue, ok := typed["Decimal"]; ok {
			return valueToString(decimalValue)
		}
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func trimFloat(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func sanitizeX12Value(value string, envelope edi.X12EnvelopeSettings) string {
	replacer := strings.NewReplacer(
		envelope.ElementSeparator, " ",
		envelope.SegmentTerminator, " ",
		envelope.ComponentSeparator, " ",
	)
	return strings.TrimSpace(replacer.Replace(value))
}

func trimTrailingEmpty(values []string) []string {
	last := len(values)
	for last > 0 && values[last-1] == "" {
		last--
	}
	return values[:last]
}

func filterDiagnostics(diagnostics []EDIDiagnostic, mode edi.ValidationMode) []EDIDiagnostic {
	if mode == edi.ValidationModeDisabled {
		filtered := make([]EDIDiagnostic, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "render_error" {
				filtered = append(filtered, diagnostic)
			}
		}
		return filtered
	}
	if mode == edi.ValidationModeWarnOnly {
		for i := range diagnostics {
			if diagnostics[i].Severity == edi.ValidationSeverityError {
				diagnostics[i].Severity = edi.ValidationSeverityWarning
			}
		}
	}
	return diagnostics
}

func hasBlockingDiagnostics(diagnostics []EDIDiagnostic, mode edi.ValidationMode) bool {
	if mode != edi.ValidationModeStrict {
		return false
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}
	return false
}

func diagnosticsToValidationError(diagnostics []EDIDiagnostic) error {
	multiErr := errortypes.NewMultiError()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		field := firstNonEmpty(diagnostic.Path, diagnostic.SegmentID, "edi")
		multiErr.Add(field, errortypes.ErrInvalid, diagnostic.Message)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func expressionDiagnostic(segment *edi.EDITemplateSegment, position int, expression string, err error) EDIDiagnostic {
	return EDIDiagnostic{
		Severity:        edi.ValidationSeverityError,
		Code:            "expression_error",
		SegmentID:       segment.SegmentID,
		ElementPosition: position,
		Path:            expression,
		Message:         err.Error(),
	}
}

func padISAID(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) > 15 {
		return value[:15]
	}
	return value + strings.Repeat(" ", 15-len(value))
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
