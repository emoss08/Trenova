package ediservice

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	partnerSettingRequiredCode        = "partner_setting_required"
	partnerSettingTypeInvalidCode     = "partner_setting_type_invalid"
	partnerSettingEnumInvalidCode     = "partner_setting_enum_invalid"
	partnerSettingPatternInvalidCode  = "partner_setting_pattern_invalid"
	partnerSettingMinLengthCode       = "partner_setting_min_length"
	partnerSettingMaxLengthCode       = "partner_setting_max_length"
	partnerSettingDeprecatedCode      = "partner_setting_deprecated"
	partnerSettingFutureCode          = "partner_setting_future"
	partnerSettingUnknownCode         = "partner_setting_unknown"
	partnerSettingSecretPlaintextCode = "partner_setting_secret_plaintext"
)

type partnerSettingIndex struct {
	fields map[string]*edi.EDIPartnerSettingField
}

func newPartnerSettingIndex(fields []*edi.EDIPartnerSettingField) *partnerSettingIndex {
	index := &partnerSettingIndex{fields: make(map[string]*edi.EDIPartnerSettingField, len(fields))}
	for _, field := range fields {
		if field == nil {
			continue
		}
		path := strings.TrimSpace(field.Path)
		if path == "" {
			continue
		}
		index.fields[path] = field
	}
	return index
}

func (s *Service) ResolvePartnerSettingSchema(
	ctx context.Context,
	profile *edi.EDIPartnerDocumentProfile,
	tenantInfo pagination.TenantInfo,
) (*edi.EDIPartnerSettingSchema, error) {
	if profile == nil {
		return nil, errortypes.NewValidationError(
			"profile",
			errortypes.ErrRequired,
			"Document profile is required",
		)
	}
	if profile.PartnerSettingsSchemaID.IsNotNil() {
		schema, err := s.partnerSettingRepo.GetPartnerSettingSchema(
			ctx,
			repositories.GetEDIPartnerSettingSchemaRequest{
				ID:         profile.PartnerSettingsSchemaID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}
		if profile.PartnerSettingsSchemaVersion > 0 &&
			schema.SchemaVersion != profile.PartnerSettingsSchemaVersion {
			return nil, errortypes.NewValidationError(
				"partnerSettingsSchemaVersion",
				errortypes.ErrInvalid,
				"Partner settings schema version does not match the pinned schema",
			)
		}
		return schema, nil
	}

	return s.partnerSettingRepo.GetActivePartnerSettingSchema(
		ctx,
		repositories.GetActiveEDIPartnerSettingSchemaRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: profile.DocumentTypeID,
			Standard:       profile.Standard,
			TransactionSet: profile.TransactionSet,
			Direction:      profile.Direction,
			X12Version: stringutils.FirstNonEmpty(
				profile.X12VersionOverride,
				edi.DefaultX12204Version,
			),
			SchemaVersion: profile.PartnerSettingsSchemaVersion,
		},
	)
}

func (s *Service) ValidatePartnerSettings(
	ctx context.Context,
	req *ValidatePartnerSettingsRequest,
) ([]edix12.Diagnostic, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"settings",
			errortypes.ErrRequired,
			"Partner settings validation request is required",
		)
	}

	profile, err := s.partnerSettingsValidationProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.validateProfilePartnerSettings(ctx, profile, req.TenantInfo, req.Settings)
}

func (s *Service) validateProfilePartnerSettings(
	ctx context.Context,
	profile *edi.EDIPartnerDocumentProfile,
	tenantInfo pagination.TenantInfo,
	settings map[string]any,
) ([]edix12.Diagnostic, error) {
	diagnostics := validateServiceFailure214PartnerSettings(profile, settings)
	schema, err := s.ResolvePartnerSettingSchema(ctx, profile, tenantInfo)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return diagnostics, nil
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
	diagnostics = append(
		diagnostics,
		validatePartnerSettingsWithIndex(settings, newPartnerSettingIndex(fields.Items))...,
	)
	return diagnostics, nil
}

func (s *Service) partnerSettingsValidationProfile(
	ctx context.Context,
	req *ValidatePartnerSettingsRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	if req.PartnerDocumentProfileID.IsNotNil() {
		profile, err := s.documentProfileRepo.GetPartnerDocumentProfileByID(
			ctx,
			repositories.GetEDIPartnerDocumentProfileByIDRequest{
				ID:         req.PartnerDocumentProfileID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}
		if req.Settings == nil {
			req.Settings = profile.PartnerSettings
		}
		return profile, nil
	}

	return &edi.EDIPartnerDocumentProfile{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		DocumentTypeID: req.DocumentTypeID,
		Standard:       defaultEDIStandard(req.Standard),
		TransactionSet: defaultTransactionSet(req.TransactionSet),
		Direction:      defaultDocumentDirection(req.Direction),
		X12VersionOverride: stringutils.FirstNonEmpty(
			req.X12Version,
			edi.DefaultX12204Version,
		),
		PartnerSettingsSchemaID:      req.PartnerSettingsSchemaID,
		PartnerSettingsSchemaVersion: req.PartnerSettingsSchemaVersion,
		PartnerSettings:              req.Settings,
	}, nil
}

func validatePartnerSettingsWithIndex(
	settings map[string]any,
	index *partnerSettingIndex,
) []edix12.Diagnostic {
	if settings == nil {
		settings = map[string]any{}
	}
	diagnostics := make([]edix12.Diagnostic, 0)
	for _, field := range index.fields {
		value := maputils.Path(settings, field.Path)
		present := partnerSettingPresent(value)
		if field.Required && !field.Nullable && !present {
			diagnostics = append(diagnostics, partnerSettingDiagnostic(
				edi.ValidationSeverityError,
				partnerSettingRequiredCode,
				field.Path,
				fmt.Sprintf("%s is required", field.Label),
				"Provide this partner setting before activating or generating EDI.",
			))
			continue
		}
		if !present {
			continue
		}
		diagnostics = append(diagnostics, validatePartnerSettingValue(field, value)...)
		diagnostics = append(diagnostics, partnerSettingStatusDiagnostics(field)...)
	}

	for _, path := range flattenPartnerSettingPaths(settings) {
		if _, ok := index.fields[path]; ok {
			continue
		}
		diagnostics = append(diagnostics, partnerSettingDiagnostic(
			edi.ValidationSeverityWarning,
			partnerSettingUnknownCode,
			path,
			fmt.Sprintf(
				"Partner setting %s is not registered in the partner settings schema",
				path,
			),
			"Register this partner setting in schema metadata if templates should depend on it.",
		))
	}
	return diagnostics
}

func validateServiceFailure214PartnerSettings(
	profile *edi.EDIPartnerDocumentProfile,
	settings map[string]any,
) []edix12.Diagnostic {
	if !serviceFailure214SettingsApply(profile) || settings == nil {
		return nil
	}
	raw, ok := settings["serviceFailure214"]
	if !ok {
		return nil
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return []edix12.Diagnostic{serviceFailure214PartnerSettingDiagnostic(
			partnerSettingTypeInvalidCode,
			"serviceFailure214",
			"serviceFailure214 must be an object",
			"Use an object with enabled, trigger, requirement, and optional code settings.",
		)}
	}

	diagnostics := make([]edix12.Diagnostic, 0)
	allowedKeys := make(map[string]struct{}, len(serviceFailure214BooleanSettingKeys())+3)
	for _, key := range serviceFailure214BooleanSettingKeys() {
		allowedKeys[key] = struct{}{}
		value, present := object[key]
		if !present {
			continue
		}
		if _, ok := value.(bool); ok {
			continue
		}
		diagnostics = append(diagnostics, serviceFailure214PartnerSettingDiagnostic(
			partnerSettingTypeInvalidCode,
			"serviceFailure214."+key,
			fmt.Sprintf("serviceFailure214.%s must be boolean", key),
			"Use true or false.",
		))
	}
	for _, key := range []string{"statusCode", "timeCode"} {
		allowedKeys[key] = struct{}{}
		value, present := object[key]
		if !present || value == nil {
			continue
		}
		if _, ok := value.(string); ok {
			continue
		}
		diagnostics = append(diagnostics, serviceFailure214PartnerSettingDiagnostic(
			partnerSettingTypeInvalidCode,
			"serviceFailure214."+key,
			fmt.Sprintf("serviceFailure214.%s must be string", key),
			"Use an X12 status code string such as SD.",
		))
	}
	allowedKeys["acceptedReasonCodes"] = struct{}{}
	for key := range object {
		if _, ok := allowedKeys[key]; ok {
			continue
		}
		diagnostics = append(diagnostics, serviceFailure214PartnerSettingDiagnostic(
			partnerSettingTypeInvalidCode,
			"serviceFailure214."+key,
			fmt.Sprintf("serviceFailure214.%s is not supported", key),
			"Remove unsupported serviceFailure214 settings before activating the profile.",
		))
	}
	if value, present := object["acceptedReasonCodes"]; present && value != nil {
		diagnostics = append(diagnostics, validateServiceFailure214ReasonCodes(value)...)
	}
	return diagnostics
}

func serviceFailure214SettingsApply(profile *edi.EDIPartnerDocumentProfile) bool {
	return profile != nil &&
		profile.Standard == edi.EDIStandardX12 &&
		profile.TransactionSet == edi.TransactionSet214 &&
		profile.Direction == edi.DocumentDirectionOutbound
}

func serviceFailure214BooleanSettingKeys() []string {
	return []string{
		"enabled",
		"sendOnReviewed",
		"sendOnResolved",
		"mandatoryOnReviewed",
		"mandatoryOnResolved",
		"requireStatusReasonCode",
		"requireLocation",
		"requireLocationName",
		"requireCityState",
		"requirePostalCode",
		"requireTimeCode",
		"requireStop",
		"requireProNumber",
		"requireBol",
	}
}

func validateServiceFailure214ReasonCodes(value any) []edix12.Diagnostic {
	switch typed := value.(type) {
	case []string:
		return nil
	case []any:
		diagnostics := make([]edix12.Diagnostic, 0)
		for index, item := range typed {
			if _, ok := item.(string); ok {
				continue
			}
			diagnostics = append(diagnostics, serviceFailure214PartnerSettingDiagnostic(
				partnerSettingTypeInvalidCode,
				fmt.Sprintf("serviceFailure214.acceptedReasonCodes[%d]", index),
				"serviceFailure214.acceptedReasonCodes entries must be strings",
				"Use X12 status reason code strings such as NS.",
			))
		}
		return diagnostics
	default:
		return []edix12.Diagnostic{serviceFailure214PartnerSettingDiagnostic(
			partnerSettingTypeInvalidCode,
			"serviceFailure214.acceptedReasonCodes",
			"serviceFailure214.acceptedReasonCodes must be an array of strings",
			"Use X12 status reason code strings such as NS.",
		)}
	}
}

func serviceFailure214PartnerSettingDiagnostic(
	code string,
	path string,
	message string,
	suggestedFix string,
) edix12.Diagnostic {
	return partnerSettingDiagnostic(
		edi.ValidationSeverityError,
		code,
		path,
		message,
		suggestedFix,
	)
}

func validatePartnerSettingValue(
	field *edi.EDIPartnerSettingField,
	value any,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0)
	if !partnerSettingTypeMatches(field.DataType, value) {
		diagnostics = append(diagnostics, partnerSettingDiagnostic(
			edi.ValidationSeverityError,
			partnerSettingTypeInvalidCode,
			field.Path,
			fmt.Sprintf("%s must be %s", field.Label, field.DataType),
			"Use a value that matches the partner setting schema type.",
		))
	}
	if field.Secret && plaintextSecret(value) {
		diagnostics = append(diagnostics, partnerSettingDiagnostic(
			edi.ValidationSeverityError,
			partnerSettingSecretPlaintextCode,
			field.Path,
			fmt.Sprintf("%s must be supplied through secret storage", field.Label),
			"Store secret values through encrypted partner setting secret support.",
		))
	}

	valueString, isString := value.(string)
	//nolint:nestif // String validations are grouped to keep each field's diagnostics together.
	if isString {
		trimmed := strings.TrimSpace(valueString)
		if field.MinLength > 0 && len(trimmed) < field.MinLength {
			diagnostics = append(diagnostics, partnerSettingDiagnostic(
				edi.ValidationSeverityError,
				partnerSettingMinLengthCode,
				field.Path,
				fmt.Sprintf("%s must be at least %d characters", field.Label, field.MinLength),
				"Provide a value that meets the minimum length.",
			))
		}
		if field.MaxLength > 0 && len(trimmed) > field.MaxLength {
			diagnostics = append(diagnostics, partnerSettingDiagnostic(
				edi.ValidationSeverityError,
				partnerSettingMaxLengthCode,
				field.Path,
				fmt.Sprintf("%s must be no more than %d characters", field.Label, field.MaxLength),
				"Shorten the partner setting value.",
			))
		}
		if strings.TrimSpace(field.ValidationPattern) != "" {
			matched, err := regexp.MatchString(field.ValidationPattern, trimmed)
			if err != nil || !matched {
				diagnostics = append(diagnostics, partnerSettingDiagnostic(
					edi.ValidationSeverityError,
					partnerSettingPatternInvalidCode,
					field.Path,
					fmt.Sprintf("%s does not match the required format", field.Label),
					"Provide a value that matches the schema validation pattern.",
				))
			}
		}
	}
	if len(field.AllowedValues) > 0 && !stringInSlice(fmt.Sprint(value), field.AllowedValues) {
		diagnostics = append(diagnostics, partnerSettingDiagnostic(
			edi.ValidationSeverityError,
			partnerSettingEnumInvalidCode,
			field.Path,
			fmt.Sprintf(
				"%s must be one of %s",
				field.Label,
				strings.Join(field.AllowedValues, ", "),
			),
			"Choose an allowed partner setting value.",
		))
	}
	return diagnostics
}

func partnerSettingStatusDiagnostics(field *edi.EDIPartnerSettingField) []edix12.Diagnostic {
	switch field.Status {
	case edi.PartnerSettingStatusActive:
		return nil
	case edi.PartnerSettingStatusDeprecated:
		return []edix12.Diagnostic{partnerSettingDiagnostic(
			edi.ValidationSeverityWarning,
			partnerSettingDeprecatedCode,
			field.Path,
			fmt.Sprintf("Partner setting %s is deprecated", field.Path),
			"Move templates and profiles to an active partner setting path.",
		)}
	case edi.PartnerSettingStatusFuture:
		return []edix12.Diagnostic{partnerSettingDiagnostic(
			edi.ValidationSeverityError,
			partnerSettingFutureCode,
			field.Path,
			fmt.Sprintf("Partner setting %s is reserved for future use", field.Path),
			"Use an active partner setting path for live outbound templates.",
		)}
	default:
		return nil
	}
}

func partnerSettingTypeMatches(dataType edi.PartnerSettingDataType, value any) bool {
	switch dataType {
	case edi.PartnerSettingDataTypeString,
		edi.PartnerSettingDataTypeEnum,
		edi.PartnerSettingDataTypeSecret:
		_, ok := value.(string)
		return ok
	case edi.PartnerSettingDataTypeNumber,
		edi.PartnerSettingDataTypeDecimal:
		return partnerSettingNumeric(value)
	case edi.PartnerSettingDataTypeInteger:
		return partnerSettingInteger(value)
	case edi.PartnerSettingDataTypeBoolean:
		_, ok := value.(bool)
		return ok
	case edi.PartnerSettingDataTypeObject,
		edi.PartnerSettingDataTypeMap:
		_, ok := value.(map[string]any)
		return ok
	case edi.PartnerSettingDataTypeArray:
		switch value.(type) {
		case []any, []string:
			return true
		default:
			return false
		}
	case edi.PartnerSettingDataTypeUnknown, "":
		return true
	default:
		return true
	}
}

func partnerSettingPresent(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func partnerSettingNumeric(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, decimal.Decimal:
		return true
	default:
		return false
	}
}

func partnerSettingInteger(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return math.Trunc(typed) == typed
	case float32:
		return math.Trunc(float64(typed)) == float64(typed)
	default:
		return false
	}
}

func plaintextSecret(value any) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != "" && !strings.HasPrefix(text, "secret://")
}

func flattenPartnerSettingPaths(settings map[string]any) []string {
	paths := make([]string, 0)
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		nested, ok := value.(map[string]any)
		if !ok {
			if prefix != "" {
				paths = append(paths, prefix)
			}
			return
		}
		if len(nested) == 0 && prefix != "" {
			paths = append(paths, prefix)
			return
		}
		for key, item := range nested {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			walk(path, item)
		}
	}
	walk("", settings)
	return paths
}

func partnerSettingDiagnostic(
	severity edi.ValidationSeverity,
	code string,
	path string,
	message string,
	suggestedFix string,
) edix12.Diagnostic {
	return edix12.Diagnostic{
		Severity:     severity,
		Code:         code,
		Path:         "partner." + strings.TrimPrefix(path, "partner."),
		Message:      message,
		SuggestedFix: suggestedFix,
	}
}

func stringInSlice(value string, allowed []string) bool {
	return slices.Contains(allowed, value)
}

func defaultEDIStandard(value edi.EDIStandard) edi.EDIStandard {
	if value != "" {
		return value
	}
	return edi.EDIStandardX12
}

func defaultTransactionSet(value edi.TransactionSet) edi.TransactionSet {
	if value != "" {
		return value
	}
	return edi.TransactionSet204
}

func defaultDocumentDirection(value edi.DocumentDirection) edi.DocumentDirection {
	if value != "" {
		return value
	}
	return edi.DocumentDirectionOutbound
}

func hasPartnerSettingErrorDiagnostics(diagnostics []edix12.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}
	return false
}

func validateTemplatePartnerSettings(
	version *edi.EDITemplateVersion,
	index *partnerSettingIndex,
	schemaMissing bool,
) []edix12.Diagnostic {
	if schemaMissing || version == nil || index == nil {
		return nil
	}

	diagnostics := make([]edix12.Diagnostic, 0)
	for _, segment := range version.Segments {
		if segment == nil {
			continue
		}
		diagnostics = append(
			diagnostics,
			validatePartnerSettingReferences(index, conditionSourceReferences(segment, nil))...,
		)
		for idx := range segment.Elements {
			element := &segment.Elements[idx]
			diagnostics = append(
				diagnostics,
				validatePartnerSettingReferences(
					index,
					elementSourceReferences(segment, element),
				)...,
			)
		}
	}
	return diagnostics
}

func validatePartnerSettingReferences(
	index *partnerSettingIndex,
	references []sourceContextReference,
) []edix12.Diagnostic {
	diagnostics := make([]edix12.Diagnostic, 0, len(references))
	for _, reference := range references {
		path := strings.TrimSpace(reference.Path)
		if sourceContextRoot(path) != string(edi.SourceContextKindPartner) {
			continue
		}
		path = strings.TrimPrefix(path, "partner.")
		field := index.fields[path]
		if field == nil {
			diagnostics = append(diagnostics, templatePartnerSettingDiagnostic(
				reference,
				edi.ValidationSeverityWarning,
				partnerSettingUnknownCode,
				fmt.Sprintf(
					"Partner setting %s is not registered in the partner settings schema",
					path,
				),
				"Register this partner setting path before depending on it in templates.",
			))
			continue
		}
		diagnostics = append(
			diagnostics,
			templatePartnerSettingStatusDiagnostics(reference, field)...)
	}
	return diagnostics
}

func templatePartnerSettingStatusDiagnostics(
	reference sourceContextReference,
	field *edi.EDIPartnerSettingField,
) []edix12.Diagnostic {
	switch field.Status {
	case edi.PartnerSettingStatusActive:
		return nil
	case edi.PartnerSettingStatusDeprecated:
		return []edix12.Diagnostic{templatePartnerSettingDiagnostic(
			reference,
			edi.ValidationSeverityWarning,
			partnerSettingDeprecatedCode,
			fmt.Sprintf("Partner setting %s is deprecated", field.Path),
			"Use an active partner setting path.",
		)}
	case edi.PartnerSettingStatusFuture:
		return []edix12.Diagnostic{templatePartnerSettingDiagnostic(
			reference,
			edi.ValidationSeverityError,
			partnerSettingFutureCode,
			fmt.Sprintf("Partner setting %s is reserved for future use", field.Path),
			"Use an active partner setting path for live outbound templates.",
		)}
	default:
		return nil
	}
}

func templatePartnerSettingDiagnostic(
	reference sourceContextReference,
	severity edi.ValidationSeverity,
	code string,
	message string,
	suggestedFix string,
) edix12.Diagnostic {
	position := 0
	if reference.Element != nil {
		position = reference.Element.Position
	}
	segmentID := ""
	if reference.Segment != nil {
		segmentID = reference.Segment.SegmentID
	}
	return edix12.Diagnostic{
		Severity:        severity,
		Code:            code,
		SegmentID:       segmentID,
		ElementPosition: position,
		Path:            stringutils.FirstNonEmpty(reference.Field, reference.Path),
		Message:         message,
		SuggestedFix:    suggestedFix,
	}
}

func partnerSettingsValidationError(diagnostics []edix12.Diagnostic) error {
	multiErr := errortypes.NewMultiError()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		path := stringutils.FirstNonEmpty(diagnostic.Path, "partnerSettings")
		multiErr.Add(path, errortypes.ErrInvalid, diagnostic.Message)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
