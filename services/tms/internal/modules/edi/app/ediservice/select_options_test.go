package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SelectDocumentTypeOptionsDelegatesTheRequest(t *testing.T) {
	t.Parallel()

	req := &edipartnerports.EDIDocumentTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
			Pagination: pagination.Info{Limit: 17, Offset: 3},
			Query:      "load",
		},
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
	}
	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	repo.EXPECT().
		SelectDocumentTypeOptions(mock.Anything, req).
		Return(&pagination.ListResult[*edi.EDIDocumentType]{
			Items: []*edi.EDIDocumentType{{Code: "204"}},
			Total: 1,
		}, nil).
		Once()

	svc := &Service{documentTypeRepo: repo}
	documentTypes, err := svc.SelectDocumentTypeOptions(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, "204", documentTypes.Items[0].Code)
}
