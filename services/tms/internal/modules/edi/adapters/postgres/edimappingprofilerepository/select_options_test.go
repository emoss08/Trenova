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

	assert.Contains(t, sql, `FROM "edi_mapping_profiles" emp LEFT JOIN "edi_partners" p ON p."id" = emp."edi_partner_id"`)
	assert.Contains(t, sql, `emp."edi_partner_id" = $3 AND (emp."name" ILIKE $4 OR emp."description" ILIKE $5) LIMIT 10`)
	assert.Contains(t, args, "edip_1")
	assert.Contains(t, args, "%carrier%")
}
