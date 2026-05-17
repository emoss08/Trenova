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

	diagnostics := validateTemplateVersionDefinition(version)

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

	diagnostics := validateTemplateVersionDefinition(version)

	requireDiagnosticCode(t, diagnostics, "required_control_segment_missing")
	requireDiagnosticCode(t, diagnostics, "duplicate_sequence")
	requireDiagnosticCode(t, diagnostics, "condition_error")
	requireDiagnosticCode(t, diagnostics, "transform_base_source_required")
	requireDiagnosticCode(t, diagnostics, "unsupported_transform_operation")
	requireDiagnosticCode(t, diagnostics, "starlark_runtime_error")
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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

	service := &Service{documentRepo: repo}
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
