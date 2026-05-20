package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SelectOptionsDelegateRequests(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	selectReq := &pagination.SelectQueryRequest{
		TenantInfo: tenantInfo,
		Pagination: pagination.Info{Limit: 17, Offset: 3},
		Query:      "load",
	}
	queryReq := &pagination.QueryOptions{
		TenantInfo: tenantInfo,
		Pagination: pagination.Info{Limit: 19, Offset: 5},
		Query:      "carrier",
	}
	documentTypeReq := &repositories.EDIDocumentTypeSelectOptionsRequest{
		SelectQueryRequest: selectReq,
		Standard:           edi.EDIStandardX12,
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.DocumentStatusActive,
	}
	templateReq := &repositories.EDITemplateSelectOptionsRequest{
		SelectQueryRequest: selectReq,
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.TemplateStatusDraft,
	}
	profileReq := &repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
		SelectQueryRequest: selectReq,
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.DocumentStatusActive,
		PartnerID:          pulid.MustNew("edip_"),
	}
	repeated := true
	sourceReq := &repositories.ListEDISourceContextFieldsRequest{
		Filter:         queryReq,
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.SourceContextFieldStatusActive,
		SourceKind:     edi.SourceContextKindShipment,
		Repeated:       &repeated,
		PathPrefix:     "shipment.",
	}
	required := true
	secret := false
	partnerSettingReq := &repositories.ListEDIPartnerSettingFieldsRequest{
		Filter:         queryReq,
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.PartnerSettingStatusActive,
		PathPrefix:     "carrier.",
		GroupKey:       "carrier",
		Required:       &required,
		Secret:         &secret,
	}

	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		SelectDocumentTypeOptions(mock.Anything, documentTypeReq).
		Return(&pagination.ListResult[*edi.EDIDocumentType]{Items: []*edi.EDIDocumentType{{Code: "204"}}, Total: 1}, nil).
		Once()
	repo.EXPECT().
		SelectTemplateOptions(mock.Anything, templateReq).
		Return(&pagination.ListResult[*edi.EDITemplate]{Items: []*edi.EDITemplate{{Name: "Outbound 204"}}, Total: 1}, nil).
		Once()
	repo.EXPECT().
		SelectPartnerDocumentProfileOptions(mock.Anything, profileReq).
		Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{Items: []*edi.EDIPartnerDocumentProfile{{Name: "Profile"}}, Total: 1}, nil).
		Once()
	repo.EXPECT().
		SelectSourceContextFieldOptions(mock.Anything, sourceReq).
		Return(&pagination.ListResult[*edi.EDISourceContextField]{Items: []*edi.EDISourceContextField{{Path: "shipment.bol"}}, Total: 1}, nil).
		Once()
	repo.EXPECT().
		SelectPartnerSettingFieldOptions(mock.Anything, partnerSettingReq).
		Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{Items: []*edi.EDIPartnerSettingField{{Path: "carrier.scac"}}, Total: 1}, nil).
		Once()

	svc := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo}

	documentTypes, err := svc.SelectDocumentTypeOptions(t.Context(), documentTypeReq)
	require.NoError(t, err)
	require.Equal(t, "204", documentTypes.Items[0].Code)

	templates, err := svc.SelectTemplateOptions(t.Context(), templateReq)
	require.NoError(t, err)
	require.Equal(t, "Outbound 204", templates.Items[0].Name)

	profiles, err := svc.SelectPartnerDocumentProfileOptions(t.Context(), profileReq)
	require.NoError(t, err)
	require.Equal(t, "Profile", profiles.Items[0].Name)

	sourceFields, err := svc.SelectSourceContextFieldOptions(t.Context(), sourceReq)
	require.NoError(t, err)
	require.Equal(t, "shipment.bol", sourceFields.Items[0].Path)

	partnerFields, err := svc.SelectPartnerSettingFieldOptions(t.Context(), partnerSettingReq)
	require.NoError(t, err)
	require.Equal(t, "carrier.scac", partnerFields.Items[0].Path)
}
