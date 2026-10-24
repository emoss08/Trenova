package edimappingprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestMappingProfileOptionsQuery_AppliesThePartnerAndSearch(t *testing.T) {
	t.Parallel()

	q := mappingProfileOptionsQuery(&ediports.EDIMappingProfileSelectOptionsRequest{
		PartnerID: pulid.ID("edip_1"),
	}).Where(mappingProfileSearch("carrier")).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}
