package permit_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validPermit() *permit.Permit {
	issued, expires := int64(1000), int64(2000)

	return &permit.Permit{
		ShipmentID:   pulid.MustNew("shp_"),
		StateID:      pulid.MustNew("us_"),
		PermitNumber: "GA-1234",
		Status:       permit.StatusActive,
		IssuedAt:     &issued,
		ExpiresAt:    &expires,
	}
}

func TestPermit_RejectsAnUnknownStatus(t *testing.T) {
	entity := validPermit()
	entity.Status = permit.Status("Revoked")

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "status"))
}

func TestPermit_AcceptsEveryDeclaredStatus(t *testing.T) {
	for _, status := range []permit.Status{
		permit.StatusPending,
		permit.StatusActive,
		permit.StatusExpired,
		permit.StatusVoid,
	} {
		t.Run(status.String(), func(t *testing.T) {
			entity := validPermit()
			entity.Status = status

			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			assert.Empty(t, validationtest.FieldCodes(multiErr, "status"))
		})
	}
}
