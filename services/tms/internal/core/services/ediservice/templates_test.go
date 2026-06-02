package ediservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateTemplateVersionDefinition_ValidBase204(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(version, nil, false)

	require.Empty(t, diagnostics)
}

func TestValidateTemplateVersionDefinition_CatchesDraftDefinitionErrors(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.Segments[2].SegmentID = "BAD"
	version.Segments[3].Sequence = version.Segments[4].Sequence
	version.Segments[5].Condition = "shipment.bol = bad"
	version.Segments[6].Elements[1].Source = edi.TemplateElementSourceTransform
	version.Segments[6].Elements[1].BaseSource = nil
	version.Segments[7].Elements[1].Source = edi.TemplateElementSourceTransform
	version.Segments[7].Elements[1].BaseSource = &edi.TemplateElementBaseSource{
		Source:    edi.TemplateElementSourceFieldPath,
		FieldPath: "ratingDetail.note",
	}
	version.Segments[7].Elements[1].TransformPipeline = []edi.TemplateTransformStep{
		{Operation: "unknown"},
	}
	version.Segments[8].Elements[1].Source = edi.TemplateElementSourceStarlark
	version.Segments[8].Elements[1].StarlarkScript = "def other(ctx):\n    return 'x'"
	version.Segments[8].Elements[1].StarlarkFunction = "missing"

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(version, nil, false)

	requireDiagnosticCode(t, diagnostics, "required_control_segment_missing")
	requireDiagnosticCode(t, diagnostics, "duplicate_sequence")
	requireDiagnosticCode(t, diagnostics, "condition_error")
	requireDiagnosticCode(t, diagnostics, "transform_base_source_required")
	requireDiagnosticCode(t, diagnostics, "unsupported_transform_operation")
	requireDiagnosticCode(t, diagnostics, "script_function_not_found")
}

func TestValidateTemplateVersionDefinition_SourceContextPaths(t *testing.T) {
	tenantInfo := testTenantInfo()

	tests := []struct {
		name       string
		mutate     func(*edi.EDITemplateVersion)
		wantCode   string
		wantAbsent string
	}{
		{
			name: "valid shipment bol",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "L11").Elements[0].FieldPath = "bol"
			},
			wantAbsent: sourceContextPathUnknownCode,
		},
		{
			name: "invalid shipment path",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "L11").Elements[0].FieldPath = "notReal"
			},
			wantCode: sourceContextPathUnknownCode,
		},
		{
			name: "valid repeat path on stop context",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "N1").Elements[1].RepeatPath = "locationName"
			},
			wantAbsent: sourceContextRepeatMismatchCode,
		},
		{
			name: "repeat mismatch on commodity context",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "L5").Elements[1].RepeatPath = "locationName"
			},
			wantCode: sourceContextRepeatMismatchCode,
		},
		{
			name: "invalid runtime key",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "ST").Elements[1].RuntimeKey = "notReal"
			},
			wantCode: sourceContextPathUnknownCode,
		},
		{
			name: "invalid transform reference",
			mutate: func(version *edi.EDITemplateVersion) {
				segment := findTemplateSegment(t, version, "L11")
				segment.Elements[0].Source = edi.TemplateElementSourceTransform
				segment.Elements[0].BaseSource = &edi.TemplateElementBaseSource{
					Source:    edi.TemplateElementSourceFieldPath,
					FieldPath: "bol",
				}
				segment.Elements[0].TransformPipeline = []edi.TemplateTransformStep{
					{
						Operation: "concat",
						Arguments: map[string]any{
							"values": []any{"$shipment.notReal"},
						},
					},
				}
			},
			wantCode: sourceContextPathUnknownCode,
		},
		{
			name: "invalid declarative condition path",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "L11").Condition = "shipment.notReal"
			},
			wantCode: sourceContextPathUnknownCode,
		},
		{
			name: "unknown partner setting path warns",
			mutate: func(version *edi.EDITemplateVersion) {
				segment := findTemplateSegment(t, version, "B2")
				segment.Elements[0].PartnerSettingPath = "customThing"
			},
			wantCode: partnerSettingUnknownCode,
		},
		{
			name: "unknown transform partner argument warns",
			mutate: func(version *edi.EDITemplateVersion) {
				segment := findTemplateSegment(t, version, "L11")
				segment.Elements[0].Source = edi.TemplateElementSourceTransform
				segment.Elements[0].BaseSource = &edi.TemplateElementBaseSource{
					Source:             edi.TemplateElementSourcePartnerSetting,
					PartnerSettingPath: "carrier.scac",
				}
				segment.Elements[0].TransformPipeline = []edi.TemplateTransformStep{
					{
						Operation: "concat",
						Arguments: map[string]any{
							"values": []any{"$partner.notReal"},
						},
					},
				}
			},
			wantCode: partnerSettingUnknownCode,
		},
		{
			name: "deprecated partner condition warns",
			mutate: func(version *edi.EDITemplateVersion) {
				findTemplateSegment(t, version, "L11").Condition = "partner.carrier.legacyCode"
			},
			wantCode: partnerSettingDeprecatedCode,
		},
		{
			name: "future partner reference errors",
			mutate: func(version *edi.EDITemplateVersion) {
				segment := findTemplateSegment(t, version, "B2")
				segment.Elements[0].PartnerSettingPath = "carrier.futureCode"
			},
			wantCode: partnerSettingFutureCode,
		},
		{
			name: "future mapping path errors",
			mutate: func(version *edi.EDITemplateVersion) {
				segment := findTemplateSegment(t, version, "B2")
				segment.Elements[0].Source = edi.TemplateElementSourceMapping
				segment.Elements[0].MappingSourcePath = "customer"
			},
			wantCode: sourceContextPathFutureCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := validTemplateVersion(tenantInfo)
			tt.mutate(version)

			diagnostics := validateTemplateVersionDefinitionWithSourceContext(
				version,
				testSourceContextIndex(),
				false,
				testTemplatePartnerSettingIndex(),
			)

			if tt.wantCode != "" {
				requireDiagnosticCode(t, diagnostics, tt.wantCode)
			}
			if tt.wantAbsent != "" {
				requireNoDiagnosticCode(t, diagnostics, tt.wantAbsent)
			}
			if tt.name == "unknown partner setting path warns" ||
				tt.name == "unknown transform partner argument warns" ||
				tt.name == "deprecated partner condition warns" {
				requireDiagnosticSeverity(
					t,
					diagnostics,
					tt.wantCode,
					edi.ValidationSeverityWarning,
				)
			}
			if tt.name == "future partner reference errors" {
				requireDiagnosticSeverity(
					t,
					diagnostics,
					partnerSettingFutureCode,
					edi.ValidationSeverityError,
				)
			}
			if tt.name == "future mapping path errors" {
				requireDiagnosticSeverity(
					t,
					diagnostics,
					sourceContextPathFutureCode,
					edi.ValidationSeverityError,
				)
			}
		})
	}
}

func TestValidateTemplateVersionDefinition_SourceContextPathsFor214ShipmentStatus(t *testing.T) {
	tenantInfo := testTenantInfo()
	versionID := pulid.MustNew("editv_")
	version := &edi.EDITemplateVersion{
		ID:                versionID,
		BusinessUnitID:    tenantInfo.BuID,
		OrganizationID:    tenantInfo.OrgID,
		TemplateID:        pulid.MustNew("editpl_"),
		VersionNumber:     1,
		X12Version:        edi.DefaultX12204Version,
		FunctionalGroupID: "QM",
		Status:            edi.TemplateStatusDraft,
		Segments:          editemplates.Base214Segments(tenantInfo, versionID),
	}

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(
		version,
		testSourceContextIndex(),
		false,
		testTemplatePartnerSettingIndex(),
	)

	requireNoDiagnosticCode(t, diagnostics, sourceContextPathUnknownCode)
	requireNoDiagnosticCode(t, diagnostics, sourceContextRootInvalidCode)
}

func TestService_CertifyTemplateVersionRequiresCleanValidation(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.Segments = version.Segments[:2]
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
			TenantInfo: tenantInfo,
		}).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.CertifyTemplateVersion(
		t.Context(),
		&EDIActionNotesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
		},
		testActor(tenantInfo),
	)

	require.Error(t, err)
}

func TestService_CertifyTemplateVersionMarksDraftCertified(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
			TenantInfo: tenantInfo,
		}).
		Return(version, nil)
	repo.EXPECT().
		UpdateTemplateVersionMetadata(mock.Anything, mock.MatchedBy(func(entity *edi.EDITemplateVersion) bool {
			return entity.Status == edi.TemplateStatusCertified &&
				entity.CertifiedAt != nil &&
				entity.CertifiedByID.IsNotNil()
		})).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	updated, err := service.CertifyTemplateVersion(
		t.Context(),
		&EDIActionNotesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
			Notes:      "ready",
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.Equal(t, edi.TemplateStatusCertified, updated.Status)
}

func TestService_ActivateTemplateVersionRequiresCertified(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, mock.Anything).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.ActivateTemplateVersion(
		t.Context(),
		&EDIActionNotesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
		},
		testActor(tenantInfo),
	)

	require.Error(t, err)
}

func TestService_ActivateTemplateVersionPromotesCertified(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.Status = edi.TemplateStatusCertified
	active := *version
	active.Status = edi.TemplateStatusActive
	active.IsActive = true
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, mock.Anything).
		Return(version, nil).
		Times(2)
	repo.EXPECT().
		ActivateTemplateVersion(mock.Anything, mock.MatchedBy(func(req repositories.ActivateEDITemplateVersionRequest) bool {
			return req.VersionID == version.ID &&
				req.TemplateID == version.TemplateID &&
				!req.IsRollback
		})).
		Return(&active, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	updated, err := service.ActivateTemplateVersion(
		t.Context(),
		&EDIActionNotesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.True(t, updated.IsActive)
	require.Equal(t, edi.TemplateStatusActive, updated.Status)
}

func TestService_ReplaceDraftScriptLibrariesReplacesAuthoritatively(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, mock.Anything).
		Return(version, nil)
	repo.EXPECT().
		ReplaceTemplateVersionScriptLibraries(
			mock.Anything,
			mock.MatchedBy(func(req repositories.ReplaceEDITemplateVersionScriptLibrariesRequest) bool {
				if req.Version.ID != version.ID || req.Version.Version != 7 {
					return false
				}
				if len(req.ScriptLibraries) != 1 {
					return false
				}
				library := req.ScriptLibraries[0]
				return library.ID.IsNil() &&
					library.TemplateVersionID == version.ID &&
					library.OrganizationID == tenantInfo.OrgID &&
					library.BusinessUnitID == tenantInfo.BuID &&
					library.Status == edi.TemplateStatusDraft &&
					library.Version == 0
			}),
		).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	updated, err := service.ReplaceDraftScriptLibraries(
		t.Context(),
		&ReplaceEDITemplateScriptLibrariesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
			Version:    7,
			ScriptLibraries: []*edi.EDITemplateScriptLibrary{
				{
					ID:       pulid.MustNew("edisl_"),
					Name:     "refs",
					Language: edi.ScriptLanguageStarlark,
					Script:   "def ref(ctx):\n    return 'x'",
					Status:   edi.TemplateStatusActive,
					Version:  99,
				},
			},
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.Same(t, version, updated)
}

func TestService_ReplaceDraftScriptLibrariesRejectsNonDraft(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.Status = edi.TemplateStatusCertified
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, mock.Anything).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.ReplaceDraftScriptLibraries(
		t.Context(),
		&ReplaceEDITemplateScriptLibrariesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
		},
		testActor(tenantInfo),
	)

	requireValidationError(t, err, "status", errortypes.ErrInvalidOperation)
}

func TestService_CreateDraftVersionClonesScriptLibraries(t *testing.T) {
	tenantInfo := testTenantInfo()
	source := validTemplateVersion(tenantInfo)
	source.Status = edi.TemplateStatusActive
	source.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			ID:       pulid.MustNew("edisl_"),
			Name:     "refs",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'x'",
			Status:   edi.TemplateStatusActive,
		},
	}
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetActiveTemplateVersion(mock.Anything, mock.Anything).
		Return(source, nil)
	repo.EXPECT().
		ListTemplateVersions(mock.Anything, mock.Anything).
		Return([]*edi.EDITemplateVersion{source}, nil)
	repo.EXPECT().
		CreateTemplateVersion(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.CreateEDITemplateVersionRequest) bool {
				if req.Version.Status != edi.TemplateStatusDraft || len(req.ScriptLibraries) != 1 {
					return false
				}
				library := req.ScriptLibraries[0]
				return library.ID.IsNil() &&
					library.Name == "refs" &&
					library.Status == edi.TemplateStatusDraft
			}),
		).
		Return(&edi.EDITemplateVersion{Status: edi.TemplateStatusDraft}, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.CreateDraftVersion(
		t.Context(),
		&CreateEDITemplateDraftRequest{
			TenantInfo: tenantInfo,
			TemplateID: source.TemplateID,
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
}

func TestValidateTemplateVersionDefinition_ValidatesScriptLibraries(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "Refs",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'x'",
		},
		{
			Name:     "refs",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'y'",
		},
		{
			Name:     "bad",
			Language: edi.ScriptLanguage("Python"),
			Script:   "def bad(ctx)\n    return 'z'",
		},
	}

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(version, nil, false)

	requireDiagnosticCode(t, diagnostics, "script_library_duplicate_name")
	requireDiagnosticCode(t, diagnostics, "script_library_duplicate_function")
	requireDiagnosticCode(t, diagnostics, "script_library_language_invalid")
	requireDiagnosticCode(t, diagnostics, "script_library_syntax_error")
}

func TestValidateTemplateVersionDefinition_RejectsReservedLibraryFunction(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "helpers",
			Language: edi.ScriptLanguageStarlark,
			Script: `def trim(ctx):
    return "unsafe"`,
		},
	}

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(version, nil, false)

	requireDiagnosticCode(t, diagnostics, "script_library_reserved_function")
}

func TestValidateTemplateVersionDefinition_ValidatesLibraryReferences(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.Segments[5].Condition = "starlark:missing_condition"
	version.Segments[8].Elements[1].Source = edi.TemplateElementSourceStarlark
	version.Segments[8].Elements[1].StarlarkFunction = "missing_element"

	diagnostics := validateTemplateVersionDefinitionWithSourceContext(version, nil, false)

	requireDiagnosticCode(t, diagnostics, "script_function_not_found")
}

func TestService_CertifyTemplateVersionAcceptsValidScriptLibraries(t *testing.T) {
	tenantInfo := testTenantInfo()
	version := validTemplateVersion(tenantInfo)
	version.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "refs",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'x'",
		},
	}
	version.Segments[8].Elements[1].Source = edi.TemplateElementSourceStarlark
	version.Segments[8].Elements[1].StarlarkFunction = "ref"
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateVersionByID(mock.Anything, mock.Anything).
		Return(version, nil)
	repo.EXPECT().
		UpdateTemplateVersionMetadata(mock.Anything, mock.Anything).
		Return(version, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.CertifyTemplateVersion(
		t.Context(),
		&EDIActionNotesRequest{
			TenantInfo: tenantInfo,
			TemplateID: version.TemplateID,
			VersionID:  version.ID,
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
}

func TestService_CreateTemplateAcceptsMatchingDocumentType(t *testing.T) {
	tenantInfo := testTenantInfo()
	documentTypeID := pulid.MustNew("edidt_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		}).
		Return([]*edi.EDIDocumentType{{ID: documentTypeID}}, nil)
	repo.EXPECT().
		CreateTemplate(mock.Anything, mock.MatchedBy(func(req *repositories.CreateEDITemplateRequest) bool {
			return req.Template.DocumentTypeID == documentTypeID &&
				req.Template.Status == edi.TemplateStatusDraft &&
				req.Version.Status == edi.TemplateStatusDraft
		})).
		Return(
			&edi.EDITemplate{
				ID:             pulid.MustNew("editpl_"),
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				DocumentTypeID: documentTypeID,
				Status:         edi.TemplateStatusDraft,
			},
			&edi.EDITemplateVersion{
				ID:             pulid.MustNew("editv_"),
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				Status:         edi.TemplateStatusDraft,
			},
			nil,
		)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	created, err := service.CreateTemplate(
		t.Context(),
		&CreateEDITemplateRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: documentTypeID,
			Name:           "Outbound 204",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.Equal(t, documentTypeID, created.DocumentTypeID)
}

func TestService_CreateTemplateRejectsMismatchedDocumentType(t *testing.T) {
	tenantInfo := testTenantInfo()
	documentTypeID := pulid.MustNew("edidt_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		}).
		Return([]*edi.EDIDocumentType{{ID: pulid.MustNew("edidt_")}}, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.CreateTemplate(
		t.Context(),
		&CreateEDITemplateRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: documentTypeID,
			Name:           "Outbound 204",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		},
		testActor(tenantInfo),
	)

	requireValidationError(t, err, "documentTypeId", errortypes.ErrInvalidReference)
}

func TestService_CreateTemplateReturnsDraftWithoutActiveVersion(t *testing.T) {
	tenantInfo := testTenantInfo()
	documentTypeID := pulid.MustNew("edidt_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, mock.Anything).
		Return([]*edi.EDIDocumentType{{ID: documentTypeID}}, nil)
	repo.EXPECT().
		CreateTemplate(mock.Anything, mock.Anything).
		Return(
			&edi.EDITemplate{
				ID:             pulid.MustNew("editpl_"),
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				DocumentTypeID: documentTypeID,
				Status:         edi.TemplateStatusDraft,
			},
			&edi.EDITemplateVersion{
				ID:             pulid.MustNew("editv_"),
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				Status:         edi.TemplateStatusDraft,
				IsActive:       false,
			},
			nil,
		)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	created, err := service.CreateTemplate(
		t.Context(),
		&CreateEDITemplateRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: documentTypeID,
			Name:           "Outbound 204",
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.Nil(t, created.ActiveVersion)
}

func TestService_CreateTemplateReturnsActiveVersionWhenRepositoryCreatesActiveVersion(
	t *testing.T,
) {
	tenantInfo := testTenantInfo()
	documentTypeID := pulid.MustNew("edidt_")
	activeVersion := &edi.EDITemplateVersion{
		ID:             pulid.MustNew("editv_"),
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Status:         edi.TemplateStatusActive,
		IsActive:       true,
	}
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, mock.Anything).
		Return([]*edi.EDIDocumentType{{ID: documentTypeID}}, nil)
	repo.EXPECT().
		CreateTemplate(mock.Anything, mock.Anything).
		Return(
			&edi.EDITemplate{
				ID:             pulid.MustNew("editpl_"),
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				DocumentTypeID: documentTypeID,
				Status:         edi.TemplateStatusActive,
			},
			activeVersion,
			nil,
		)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	created, err := service.CreateTemplate(
		t.Context(),
		&CreateEDITemplateRequest{
			TenantInfo:     tenantInfo,
			DocumentTypeID: documentTypeID,
			Name:           "Outbound 204",
		},
		testActor(tenantInfo),
	)

	require.NoError(t, err)
	require.Same(t, activeVersion, created.ActiveVersion)
}

func TestService_UpdateTemplateRejectsUnknownStatus(t *testing.T) {
	tenantInfo := testTenantInfo()
	templateID := pulid.MustNew("editpl_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateByID(mock.Anything, repositories.GetEDITemplateByIDRequest{
			ID:         templateID,
			TenantInfo: tenantInfo,
		}).
		Return(&edi.EDITemplate{
			ID:             templateID,
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			Status:         edi.TemplateStatusDraft,
		}, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.UpdateTemplate(
		t.Context(),
		&UpdateEDITemplateRequest{
			TenantInfo: tenantInfo,
			TemplateID: templateID,
			Status:     edi.TemplateStatus("Published"),
		},
		testActor(tenantInfo),
	)

	requireValidationError(t, err, "status", errortypes.ErrInvalid)
}

func TestService_UpdateTemplateRejectsArchivedTemplate(t *testing.T) {
	tenantInfo := testTenantInfo()
	templateID := pulid.MustNew("editpl_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetTemplateByID(mock.Anything, repositories.GetEDITemplateByIDRequest{
			ID:         templateID,
			TenantInfo: tenantInfo,
		}).
		Return(&edi.EDITemplate{
			ID:             templateID,
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			Status:         edi.TemplateStatusArchived,
		}, nil)

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}
	_, err := service.UpdateTemplate(
		t.Context(),
		&UpdateEDITemplateRequest{
			TenantInfo: tenantInfo,
			TemplateID: templateID,
			Name:       "New name",
		},
		testActor(tenantInfo),
	)

	requireValidationError(t, err, "status", errortypes.ErrInvalidOperation)
}

func TestValidateProfileTemplateVersionCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		status  edi.DocumentStatus
		version *edi.EDITemplateVersion
		wantErr bool
	}{
		{
			name:    "active profile can pin active version",
			status:  edi.DocumentStatusActive,
			version: &edi.EDITemplateVersion{Status: edi.TemplateStatusActive},
		},
		{
			name:    "active profile can pin certified version",
			status:  edi.DocumentStatusActive,
			version: &edi.EDITemplateVersion{Status: edi.TemplateStatusCertified},
		},
		{
			name:    "active profile cannot pin draft version",
			status:  edi.DocumentStatusActive,
			version: &edi.EDITemplateVersion{Status: edi.TemplateStatusDraft},
			wantErr: true,
		},
		{
			name:    "inactive profile can pin draft version",
			status:  edi.DocumentStatusInactive,
			version: &edi.EDITemplateVersion{Status: edi.TemplateStatusDraft},
		},
		{
			name:    "inactive profile cannot pin archived version",
			status:  edi.DocumentStatusInactive,
			version: &edi.EDITemplateVersion{Status: edi.TemplateStatusArchived},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfileTemplateVersionCompatibility(tt.status, tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateProductionTemplateVersion(t *testing.T) {
	require.NoError(t, validateProductionTemplateVersion(&edi.EDITemplateVersion{
		Status: edi.TemplateStatusActive,
	}))
	require.NoError(t, validateProductionTemplateVersion(&edi.EDITemplateVersion{
		Status: edi.TemplateStatusCertified,
	}))
	require.Error(t, validateProductionTemplateVersion(&edi.EDITemplateVersion{
		Status: edi.TemplateStatusDraft,
	}))
	require.Error(t, validateProductionTemplateVersion(&edi.EDITemplateVersion{
		Status: edi.TemplateStatusArchived,
	}))
}

func validTemplateVersion(tenantInfo pagination.TenantInfo) *edi.EDITemplateVersion {
	templateID := pulid.MustNew("editpl_")
	versionID := pulid.MustNew("editv_")
	return &edi.EDITemplateVersion{
		ID:                versionID,
		BusinessUnitID:    tenantInfo.BuID,
		OrganizationID:    tenantInfo.OrgID,
		TemplateID:        templateID,
		VersionNumber:     1,
		X12Version:        edi.DefaultX12204Version,
		FunctionalGroupID: "SM",
		Status:            edi.TemplateStatusDraft,
		Segments:          editemplates.Base204Segments(tenantInfo, versionID),
	}
}

func testSourceContextIndex() *sourceContextIndex {
	fields := []*edi.EDISourceContextField{
		sourceContextField("shipment.shipmentId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.purposeCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.bol", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.weight", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.pieces", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.totalChargeAmount", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.ratingDetail.paymentMethod", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipment.ratingDetail.note", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.shipmentId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.bol", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.proNumber", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.statusCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.statusReasonCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.eventDate", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.eventTime", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.stopId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.stopType", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.stopSequence", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.locationId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.locationName", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.locationCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.addressLine", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.city", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.stateCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.postalCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.countryCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.appointmentNumber", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.scheduledWindowStart", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.scheduledWindowEnd", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.actualArrival", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.actualDeparture", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.equipmentNumber", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.equipmentType", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.exceptionCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.reasonCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.reasonDescription", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.lateMinutes", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.serviceFailureId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.serviceFailureNumber", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.serviceFailureReasonCodeId", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("shipmentStatus.serviceFailureReasonCode", "", edi.SourceContextKindShipment, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationName", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationAddressLine1", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationAddressLine2", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationCity", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationStateCode", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.locationPostalCode", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.sequence", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.sequence", "commodities", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.type", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.scheduledWindowStart", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.weight", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.pieces", "moves.0.stops", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("repeat.commodityDescription", "commodities", edi.SourceContextKindRepeat, edi.SourceContextFieldStatusActive),
		sourceContextField("partner.carrier.scac", "", edi.SourceContextKindPartner, edi.SourceContextFieldStatusActive),
		sourceContextField("partner.contact.name", "", edi.SourceContextKindPartner, edi.SourceContextFieldStatusActive),
		sourceContextField("partner.contact.phone", "", edi.SourceContextKindPartner, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.interchangeSenderId", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.interchangeReceiverId", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.interchangeDate", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.interchangeTime", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.repetitionSeparator", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.componentSeparator", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.usageIndicator", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.applicationSenderCode", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.applicationReceiverCode", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.groupDate", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.groupTime", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.groupControlNumber", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.transactionControlNumber", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.functionalGroupId", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.x12Version", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.isaControlNumber", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("runtime.transactionSegmentCount", "", edi.SourceContextKindRuntime, edi.SourceContextFieldStatusActive),
		sourceContextField("mapping.customer", "", edi.SourceContextKindMapping, edi.SourceContextFieldStatusFuture),
	}
	return newSourceContextIndex(fields)
}

func testTemplatePartnerSettingIndex() *partnerSettingIndex {
	return newPartnerSettingIndex([]*edi.EDIPartnerSettingField{
		testPartnerSettingField(
			"carrier.scac",
			edi.PartnerSettingDataTypeString,
			edi.PartnerSettingStatusActive,
		),
		testPartnerSettingField(
			"contact.name",
			edi.PartnerSettingDataTypeString,
			edi.PartnerSettingStatusActive,
		),
		testPartnerSettingField(
			"contact.phone",
			edi.PartnerSettingDataTypeString,
			edi.PartnerSettingStatusActive,
		),
		testPartnerSettingField(
			"carrier.legacyCode",
			edi.PartnerSettingDataTypeString,
			edi.PartnerSettingStatusDeprecated,
		),
		testPartnerSettingField(
			"carrier.futureCode",
			edi.PartnerSettingDataTypeString,
			edi.PartnerSettingStatusFuture,
		),
	})
}

func sourceContextField(
	path string,
	repeatPath string,
	kind edi.SourceContextKind,
	status edi.SourceContextFieldStatus,
) *edi.EDISourceContextField {
	return &edi.EDISourceContextField{
		Path:       path,
		SourceKind: kind,
		Repeated:   repeatPath != "",
		RepeatPath: repeatPath,
		Status:     status,
	}
}

func testTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func testActor(tenantInfo pagination.TenantInfo) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}
}

func findTemplateSegment(
	t *testing.T,
	version *edi.EDITemplateVersion,
	segmentID string,
) *edi.EDITemplateSegment {
	t.Helper()
	for _, segment := range version.Segments {
		if segment.SegmentID == segmentID {
			return segment
		}
	}
	require.Failf(t, "missing segment", "segment %s not found", segmentID)
	return nil
}

func requireDiagnosticCode(
	t *testing.T,
	diagnostics []edix12.Diagnostic,
	code string,
) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	require.Failf(t, "missing diagnostic code", "code %s not found in %#v", code, diagnostics)
}

func requireNoDiagnosticCode(
	t *testing.T,
	diagnostics []edix12.Diagnostic,
	code string,
) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		require.NotEqual(t, code, diagnostic.Code, "unexpected diagnostic: %#v", diagnostic)
	}
}

func requireDiagnosticSeverity(
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

func requireValidationError(
	t *testing.T,
	err error,
	field string,
	code errortypes.ErrorCode,
) {
	t.Helper()
	require.Error(t, err)

	var validationErr *errortypes.Error
	if errors.As(err, &validationErr) {
		require.Equal(t, field, validationErr.Field)
		require.Equal(t, code, validationErr.Code)
		return
	}

	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) {
		for _, validationErr := range multiErr.Errors {
			if validationErr.Field == field && validationErr.Code == code {
				return
			}
		}
	}

	require.Failf(
		t,
		"missing validation error",
		"field %s with code %s not found in %#v",
		field,
		code,
		err,
	)
}
