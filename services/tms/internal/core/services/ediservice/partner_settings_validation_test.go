package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/stretchr/testify/require"
)

func TestValidatePartnerSettingsWithIndex(t *testing.T) {
	tests := []struct {
		name         string
		settings     map[string]any
		fields       []*edi.EDIPartnerSettingField
		wantCode     string
		wantSeverity edi.ValidationSeverity
		wantEmpty    bool
	}{
		{
			name: "valid settings",
			settings: map[string]any{
				"carrier":              map[string]any{"scac": "ABCD"},
				"defaultPaymentMethod": "PP",
			},
			fields:    testPartnerSettingFields(),
			wantEmpty: true,
		},
		{
			name:         "missing carrier scac",
			settings:     map[string]any{},
			fields:       testPartnerSettingFields(),
			wantCode:     partnerSettingRequiredCode,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name: "invalid type",
			settings: map[string]any{
				"carrier": map[string]any{"scac": 1234},
			},
			fields:       testPartnerSettingFields(),
			wantCode:     partnerSettingTypeInvalidCode,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name: "invalid enum",
			settings: map[string]any{
				"carrier":              map[string]any{"scac": "ABCD"},
				"defaultPaymentMethod": "XX",
			},
			fields:       testPartnerSettingFields(),
			wantCode:     partnerSettingEnumInvalidCode,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name: "max length",
			settings: map[string]any{
				"carrier": map[string]any{"scac": "ABCDE"},
			},
			fields:       testPartnerSettingFields(),
			wantCode:     partnerSettingMaxLengthCode,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name: "deprecated warning",
			settings: map[string]any{
				"carrier": map[string]any{"scac": "ABCD", "legacyCode": "OLD"},
			},
			fields: append(
				testPartnerSettingFields(),
				testPartnerSettingField(
					"carrier.legacyCode",
					edi.PartnerSettingDataTypeString,
					edi.PartnerSettingStatusDeprecated,
				),
			),
			wantCode:     partnerSettingDeprecatedCode,
			wantSeverity: edi.ValidationSeverityWarning,
		},
		{
			name: "future error",
			settings: map[string]any{
				"carrier":        map[string]any{"scac": "ABCD"},
				"futureSettings": map[string]any{"enabled": true},
			},
			fields: append(
				testPartnerSettingFields(),
				testPartnerSettingField(
					"futureSettings.enabled",
					edi.PartnerSettingDataTypeBoolean,
					edi.PartnerSettingStatusFuture,
				),
			),
			wantCode:     partnerSettingFutureCode,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name: "unknown warning",
			settings: map[string]any{
				"carrier": map[string]any{"scac": "ABCD", "unknown": "value"},
			},
			fields:       testPartnerSettingFields(),
			wantCode:     partnerSettingUnknownCode,
			wantSeverity: edi.ValidationSeverityWarning,
		},
		{
			name: "plaintext secret",
			settings: map[string]any{
				"carrier": map[string]any{"scac": "ABCD"},
				"secrets": map[string]any{"apiToken": "plain-token"},
			},
			fields:       append(testPartnerSettingFields(), testSecretPartnerSettingField("secrets.apiToken")),
			wantCode:     partnerSettingSecretPlaintextCode,
			wantSeverity: edi.ValidationSeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := validatePartnerSettingsWithIndex(
				tt.settings,
				newPartnerSettingIndex(tt.fields),
			)

			if tt.wantEmpty {
				require.Empty(t, diagnostics)
				return
			}
			requirePartnerSettingDiagnostic(t, diagnostics, tt.wantCode, tt.wantSeverity)
		})
	}
}

func TestValidateServiceFailure214PartnerSettings(t *testing.T) {
	t.Parallel()

	profile := &edi.EDIPartnerDocumentProfile{
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet214,
		Direction:      edi.DocumentDirectionOutbound,
	}

	tests := []struct {
		name      string
		settings  map[string]any
		wantPaths []string
	}{
		{
			name: "valid settings",
			settings: map[string]any{
				"serviceFailure214": map[string]any{
					"enabled":             true,
					"sendOnReviewed":      true,
					"mandatoryOnResolved": false,
					"statusCode":          "SD",
					"acceptedReasonCodes": []any{"NS", "CA"},
				},
			},
		},
		{
			name: "non object",
			settings: map[string]any{
				"serviceFailure214": true,
			},
			wantPaths: []string{"partner.serviceFailure214"},
		},
		{
			name: "invalid typed fields",
			settings: map[string]any{
				"serviceFailure214": map[string]any{
					"enabled":             "true",
					"statusCode":          214,
					"acceptedReasonCodes": []any{"NS", 42},
				},
			},
			wantPaths: []string{
				"partner.serviceFailure214.enabled",
				"partner.serviceFailure214.statusCode",
				"partner.serviceFailure214.acceptedReasonCodes[1]",
			},
		},
		{
			name: "accepted codes not array",
			settings: map[string]any{
				"serviceFailure214": map[string]any{
					"acceptedReasonCodes": "NS",
				},
			},
			wantPaths: []string{"partner.serviceFailure214.acceptedReasonCodes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := validateServiceFailure214PartnerSettings(profile, tt.settings)

			if tt.wantPaths == nil {
				require.Empty(t, diagnostics)
				return
			}
			require.Equal(t, tt.wantPaths, diagnosticPaths(diagnostics))
		})
	}
}

func TestValidateServiceFailure214PartnerSettingsIgnoresNonOutboundX12214(t *testing.T) {
	t.Parallel()

	diagnostics := validateServiceFailure214PartnerSettings(
		&edi.EDIPartnerDocumentProfile{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		},
		map[string]any{"serviceFailure214": true},
	)

	require.Empty(t, diagnostics)
}

func testPartnerSettingFields() []*edi.EDIPartnerSettingField {
	paymentMethod := testPartnerSettingField(
		"defaultPaymentMethod",
		edi.PartnerSettingDataTypeEnum,
		edi.PartnerSettingStatusActive,
	)
	paymentMethod.AllowedValues = []string{"CC", "PP", "TP"}
	return []*edi.EDIPartnerSettingField{
		testRequiredPartnerSettingField("carrier.scac"),
		paymentMethod,
	}
}

func testRequiredPartnerSettingField(path string) *edi.EDIPartnerSettingField {
	field := testPartnerSettingField(
		path,
		edi.PartnerSettingDataTypeString,
		edi.PartnerSettingStatusActive,
	)
	field.Required = true
	field.Nullable = false
	field.MinLength = 2
	field.MaxLength = 4
	return field
}

func testPartnerSettingField(
	path string,
	dataType edi.PartnerSettingDataType,
	status edi.PartnerSettingStatus,
) *edi.EDIPartnerSettingField {
	return &edi.EDIPartnerSettingField{
		Path:     path,
		Label:    path,
		DataType: dataType,
		Nullable: true,
		Status:   status,
	}
}

func testSecretPartnerSettingField(path string) *edi.EDIPartnerSettingField {
	field := testPartnerSettingField(
		path,
		edi.PartnerSettingDataTypeSecret,
		edi.PartnerSettingStatusActive,
	)
	field.Secret = true
	return field
}

func requirePartnerSettingDiagnostic(
	t *testing.T,
	diagnostics []edix12.Diagnostic,
	code string,
	severity edi.ValidationSeverity,
) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			require.Equal(t, severity, diagnostic.Severity)
			return
		}
	}
	require.Failf(t, "missing diagnostic code", "code %s not found in %#v", code, diagnostics)
}
