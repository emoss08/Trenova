package edipartnerdocumentprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestProfileOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	q := profileOptionsQuery(&ediports.EDIPartnerDocumentProfileSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: " profile "},
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.DocumentStatusActive,
		PartnerID:          pulid.ID("edip_1"),
	}).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}
